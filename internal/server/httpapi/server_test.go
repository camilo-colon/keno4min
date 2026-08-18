package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttputil"
)

func TestAppReturnsProblemDetailsForMissingRoute(t *testing.T) {
	t.Parallel()

	server := newTestServer("local-request-id")
	response := performRequest(t, server.App(), httptest.NewRequest(http.MethodGet, "/missing", http.NoBody))

	assertProblem(t, response, fiber.StatusNotFound, "not_found", "local-request-id")
}

func TestAppReturnsMethodNotAllowedAndAllowHeader(t *testing.T) {
	t.Parallel()

	server := newTestServer("method-request-id")
	server.App().Get("/fixture/method", func(c fiber.Ctx) error {
		return c.SendStatus(fiber.StatusNoContent)
	})

	response := performRequest(t, server.App(), httptest.NewRequest(http.MethodPost, "/fixture/method", http.NoBody))

	assertProblem(t, response, fiber.StatusMethodNotAllowed, "method_not_allowed", "method-request-id")
	if allow := response.Header.Get(fiber.HeaderAllow); !strings.Contains(allow, fiber.MethodGet) {
		t.Errorf("Allow = %q, want GET", allow)
	}
}

func TestAppPreservesFiberErrorStatus(t *testing.T) {
	t.Parallel()

	server := newTestServer("fiber-error-request-id")
	server.App().Get("/fixture/conflict", func(fiber.Ctx) error {
		return fiber.ErrConflict
	})

	response := performRequest(t, server.App(), httptest.NewRequest(http.MethodGet, "/fixture/conflict", http.NoBody))
	assertProblem(t, response, fiber.StatusConflict, "conflict", "fiber-error-request-id")
}

func TestAppRecoversPanicWithoutLeakingItsValue(t *testing.T) {
	t.Parallel()

	server := newTestServer("panic-request-id")
	server.App().Get("/fixture/panic", func(fiber.Ctx) error {
		panic("database-password-is-secret")
	})

	response := performRequest(t, server.App(), httptest.NewRequest(http.MethodGet, "/fixture/panic", http.NoBody))
	body := assertProblem(t, response, fiber.StatusInternalServerError, "internal_server_error", "panic-request-id")
	if bytes.Contains(body, []byte("database-password-is-secret")) {
		t.Fatal("panic value leaked in response")
	}
}

func TestAppIgnoresIncomingRequestID(t *testing.T) {
	t.Parallel()

	server := newTestServer("service-generated-id")
	request := httptest.NewRequest(http.MethodGet, "/missing", http.NoBody)
	request.Header.Set(fiber.HeaderXRequestID, "untrusted-incoming-id")

	response := performRequest(t, server.App(), request)
	assertProblem(t, response, fiber.StatusNotFound, "not_found", "service-generated-id")
}

func TestAppAppliesBodyLimit(t *testing.T) {
	server := New(Config{
		BodyLimit:       8,
		ReadTimeout:     time.Second,
		WriteTimeout:    time.Second,
		IdleTimeout:     time.Second,
		ShutdownTimeout: time.Second,
	}, withRequestIDGenerator(func() string { return "body-request-id" }))
	server.App().Post("/fixture/body", func(c fiber.Ctx) error {
		return c.SendStatus(fiber.StatusNoContent)
	})

	listener := fasthttputil.NewInmemoryListener()
	ctx, cancel := context.WithCancel(context.Background())
	serveResult := make(chan error, 1)
	go func() {
		serveResult <- server.Serve(ctx, listener)
	}()
	awaitListener(t, listener)
	t.Cleanup(func() {
		cancel()
		if err := <-serveResult; err != nil {
			t.Errorf("stop body-limit server: %v", err)
		}
	})

	request := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(request)
	request.SetRequestURI("http://example.com/fixture/body")
	request.Header.SetMethod(fiber.MethodPost)
	request.Header.SetContentType(fiber.MIMEApplicationJSON)
	request.SetBodyString("body-is-too-large")

	response := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(response)
	client := fasthttp.Client{Dial: func(string) (net.Conn, error) { return listener.Dial() }}
	if err := client.Do(request, response); err != nil {
		t.Fatalf("oversized request: %v", err)
	}

	assertProblemValues(
		t,
		response.StatusCode(),
		string(response.Header.Peek(fiber.HeaderContentType)),
		response.Body(),
		fiber.StatusRequestEntityTooLarge,
		"payload_too_large",
		"body-request-id",
	)
}

func TestNewAppliesFiberLimitsAndTimeouts(t *testing.T) {
	t.Parallel()

	want := Config{
		BodyLimit:       1234,
		ReadTimeout:     2 * time.Second,
		WriteTimeout:    3 * time.Second,
		IdleTimeout:     4 * time.Second,
		ShutdownTimeout: 5 * time.Second,
	}
	server := New(want)
	got := server.App().Config()

	if got.BodyLimit != want.BodyLimit || got.ReadTimeout != want.ReadTimeout ||
		got.WriteTimeout != want.WriteTimeout || got.IdleTimeout != want.IdleTimeout {
		t.Fatalf("Fiber config = %+v, want limits %+v", got, want)
	}
	if got.TrustProxy {
		t.Fatal("TrustProxy enabled without a configured trust boundary")
	}
}

func TestServeDrainsActiveRequestWithinShutdownBudget(t *testing.T) {
	server := newLifecycleServer(time.Second)
	entered := make(chan struct{})
	release := make(chan struct{})
	server.App().Get("/fixture/lifecycle", func(c fiber.Ctx) error {
		close(entered)
		<-release
		return c.SendStatus(fiber.StatusNoContent)
	})

	running := startServer(t, server)
	requestResult := startLifecycleRequest(running.listener)
	awaitSignal(t, entered, "request did not enter handler")

	running.cancel()
	select {
	case err := <-running.result:
		t.Fatalf("Serve returned before active request drained: %v", err)
	case <-time.After(50 * time.Millisecond):
	}

	close(release)
	response := awaitLifecycleResponse(t, requestResult)
	if response.status != fiber.StatusNoContent || response.err != nil {
		t.Fatalf("active request result = status %d, err %v; want 204", response.status, response.err)
	}
	if err := awaitServeResult(t, running.result); err != nil {
		t.Fatalf("Serve after clean drain: %v", err)
	}
}

func TestServePropagatesShutdownTimeout(t *testing.T) {
	server := newLifecycleServer(25 * time.Millisecond)
	entered := make(chan struct{})
	release := make(chan struct{})
	server.App().Get("/fixture/lifecycle-timeout", func(c fiber.Ctx) error {
		close(entered)
		<-release
		return c.SendStatus(fiber.StatusNoContent)
	})

	running := startServer(t, server)
	requestResult := startLifecycleRequestAt(running.listener, "/fixture/lifecycle-timeout")
	awaitSignal(t, entered, "request did not enter timeout handler")

	running.cancel()
	err := awaitServeResult(t, running.result)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Serve error = %v, want context deadline exceeded", err)
	}

	close(release)
	_ = awaitLifecycleResponse(t, requestResult)
}

func TestServeWithCanceledContextDoesNotStartListener(t *testing.T) {
	server := newLifecycleServer(time.Second)
	listener := fasthttputil.NewInmemoryListener()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := server.Serve(ctx, listener); err != nil {
		t.Fatalf("Serve with canceled context: %v", err)
	}
	if connection, err := listener.Dial(); err == nil {
		_ = connection.Close()
		t.Fatal("listener remained open after canceled start")
	}
}

func TestServeCombinesShutdownAndListenerErrors(t *testing.T) {
	shutdownFailure := errors.New("shutdown failure")
	listenerFailure := errors.New("listener failure")
	listener := newFailingListener(shutdownFailure, listenerFailure)
	server := newLifecycleServer(time.Second)
	ctx, cancel := context.WithCancel(context.Background())
	serveResult := make(chan error, 1)
	go func() {
		serveResult <- server.Serve(ctx, listener)
	}()

	awaitSignal(t, listener.accepted, "listener did not enter Accept")
	cancel()
	err := awaitServeResult(t, serveResult)
	if !errors.Is(err, shutdownFailure) || !errors.Is(err, listenerFailure) {
		t.Fatalf("Serve error = %v, want both shutdown and listener failures", err)
	}
}

func newTestServer(requestID string) *Server {
	return New(Config{
		BodyLimit:       1024,
		ReadTimeout:     time.Second,
		WriteTimeout:    time.Second,
		IdleTimeout:     time.Second,
		ShutdownTimeout: time.Second,
	}, withRequestIDGenerator(func() string { return requestID }))
}

func newLifecycleServer(shutdownTimeout time.Duration) *Server {
	return New(Config{
		BodyLimit:       1024,
		ReadTimeout:     time.Second,
		WriteTimeout:    time.Second,
		IdleTimeout:     time.Second,
		ShutdownTimeout: shutdownTimeout,
	})
}

type runningServer struct {
	listener *fasthttputil.InmemoryListener
	cancel   context.CancelFunc
	result   <-chan error
}

func startServer(t *testing.T, server *Server) runningServer {
	t.Helper()

	listener := fasthttputil.NewInmemoryListener()
	ctx, cancel := context.WithCancel(context.Background())
	serveResult := make(chan error, 1)
	go func() {
		serveResult <- server.Serve(ctx, listener)
	}()
	awaitListener(t, listener)

	return runningServer{listener: listener, cancel: cancel, result: serveResult}
}

type lifecycleResponse struct {
	err    error
	status int
}

func startLifecycleRequest(listener *fasthttputil.InmemoryListener) <-chan lifecycleResponse {
	return startLifecycleRequestAt(listener, "/fixture/lifecycle")
}

func startLifecycleRequestAt(listener *fasthttputil.InmemoryListener, path string) <-chan lifecycleResponse {
	result := make(chan lifecycleResponse, 1)
	go func() {
		request := fasthttp.AcquireRequest()
		defer fasthttp.ReleaseRequest(request)
		request.SetRequestURI("http://example.com" + path)
		request.Header.SetMethod(fiber.MethodGet)

		response := fasthttp.AcquireResponse()
		defer fasthttp.ReleaseResponse(response)
		client := fasthttp.Client{Dial: func(string) (net.Conn, error) { return listener.Dial() }}
		err := client.Do(request, response)
		result <- lifecycleResponse{status: response.StatusCode(), err: err}
	}()
	return result
}

func awaitSignal(t *testing.T, signal <-chan struct{}, message string) {
	t.Helper()
	select {
	case <-signal:
	case <-time.After(2 * time.Second):
		t.Fatal(message)
	}
}

func awaitServeResult(t *testing.T, result <-chan error) error {
	t.Helper()
	select {
	case err := <-result:
		return err
	case <-time.After(2 * time.Second):
		t.Fatal("Serve did not return")
		return nil
	}
}

func awaitLifecycleResponse(t *testing.T, result <-chan lifecycleResponse) lifecycleResponse {
	t.Helper()
	select {
	case response := <-result:
		return response
	case <-time.After(2 * time.Second):
		t.Fatal("request did not return")
		return lifecycleResponse{}
	}
}

func performRequest(t *testing.T, app *fiber.App, request *http.Request) *http.Response {
	t.Helper()

	response, err := app.Test(request)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	return response
}

func assertProblem(t *testing.T, response *http.Response, status int, code, requestID string) []byte {
	t.Helper()
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read response: %v", err)
	}
	assertProblemValues(t, response.StatusCode, response.Header.Get(fiber.HeaderContentType), body, status, code, requestID)
	return body
}

func assertProblemValues(t *testing.T, gotStatus int, contentType string, body []byte, status int, code, requestID string) {
	t.Helper()

	if gotStatus != status {
		t.Fatalf("status = %d, want %d; body = %s", gotStatus, status, body)
	}
	if contentType != problemMediaType {
		t.Errorf("Content-Type = %q, want %q", contentType, problemMediaType)
	}

	var problem problemDetails
	if err := json.Unmarshal(body, &problem); err != nil {
		t.Fatalf("decode problem: %v; body = %s", err, body)
	}
	if problem.Status != status || problem.Code != code || problem.RequestID != requestID {
		t.Errorf("problem = %+v, want status=%d code=%q requestId=%q", problem, status, code, requestID)
	}
}

func awaitListener(t *testing.T, listener *fasthttputil.InmemoryListener) {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for {
		connection, err := listener.Dial()
		if err == nil {
			_ = connection.Close()
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("server did not become ready: %v", err)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

type failingListener struct {
	accepted      chan struct{}
	closed        chan struct{}
	acceptOnce    sync.Once
	closeOnce     sync.Once
	shutdownError error
	listenerError error
}

func newFailingListener(shutdownError, listenerError error) *failingListener {
	return &failingListener{
		accepted:      make(chan struct{}),
		closed:        make(chan struct{}),
		shutdownError: shutdownError,
		listenerError: listenerError,
	}
}

func (l *failingListener) Accept() (net.Conn, error) {
	l.acceptOnce.Do(func() { close(l.accepted) })
	<-l.closed
	return nil, l.listenerError
}

func (l *failingListener) Close() error {
	l.closeOnce.Do(func() { close(l.closed) })
	return l.shutdownError
}

func (*failingListener) Addr() net.Addr {
	return testAddress("in-memory")
}

type testAddress string

func (a testAddress) Network() string { return string(a) }
func (a testAddress) String() string  { return string(a) }
