package auth

import (
	"context"
	"encoding/base64"
	"net/http/httptest"
	"testing"
	"uuid"

	"github.com/gofiber/fiber/v3"
)

const validPayload = `{"aud":"cronos-clients","iss":"cronos-core","exp":1793236778,` +
	`"sub":"0c86f3ff-15e0-4fd4-9c5a-52ad237386f5","unique_name":"t2k02",` +
	`"role":"CASHIER","node_id":"e2cb4a13-d777-47b3-ab44-9193e5501a4c"}`

var wantIdentity = Identity{
	UserID:   uuid.MustParse("0c86f3ff-15e0-4fd4-9c5a-52ad237386f5"),
	Username: "t2k02",
	Role:     RoleCashier,
	NodeID:   uuid.MustParse("e2cb4a13-d777-47b3-ab44-9193e5501a4c"),
}

func TestFromContextReadsFiberAndStandardContexts(t *testing.T) {
	var fromFiber, fromContext Identity
	var fiberOK, contextOK bool
	app := newTestApp(func(ctx fiber.Ctx) error {
		fromFiber, fiberOK = FromContext(ctx)
		fromContext, contextOK = fromStandardContext(ctx.Context())
		return ctx.SendStatus(204)
	})

	if status := request(t, app, "bearer "+token(validPayload)); status != 204 {
		t.Fatalf("status %d, want 204", status)
	}
	if !fiberOK || fromFiber != wantIdentity {
		t.Fatalf("FromContext(ctx) = %+v, %v", fromFiber, fiberOK)
	}
	if !contextOK || fromContext != wantIdentity {
		t.Fatalf("FromContext(ctx.Context()) = %+v, %v", fromContext, contextOK)
	}
}

func TestFromContextWithoutMiddleware(t *testing.T) {
	if _, ok := FromContext(context.Background()); ok {
		t.Fatal("identity found in an empty context")
	}
}

func TestNewRejectsUnreadableTokens(t *testing.T) {
	for name, header := range map[string]string{
		"missing header":  "",
		"other scheme":    "Basic dXNlcjpwYXNz",
		"empty token":     "Bearer   ",
		"not a JWT":       "Bearer not-a-jwt",
		"invalid base64":  "Bearer header.%%%.signature",
		"invalid JSON":    "Bearer " + token(`not-json`),
		"invalid user ID": "Bearer " + token(`{"sub":"user","unique_name":"t2k02","role":"CASHIER","node_id":"e2cb4a13-d777-47b3-ab44-9193e5501a4c"}`),
		"missing node":    "Bearer " + token(`{"sub":"0c86f3ff-15e0-4fd4-9c5a-52ad237386f5","unique_name":"t2k02","role":"CASHIER"}`),
		"missing role":    "Bearer " + token(`{"sub":"0c86f3ff-15e0-4fd4-9c5a-52ad237386f5","unique_name":"t2k02","node_id":"e2cb4a13-d777-47b3-ab44-9193e5501a4c"}`),
	} {
		t.Run(name, func(t *testing.T) {
			app := newTestApp(func(ctx fiber.Ctx) error {
				t.Fatal("protected handler must not run")
				return nil
			})
			if status := request(t, app, header); status != 401 {
				t.Fatalf("status %d, want 401", status)
			}
		})
	}
}

func fromStandardContext(ctx context.Context) (Identity, bool) {
	return FromContext(ctx)
}

func newTestApp(protected fiber.Handler) *fiber.App {
	app := fiber.New(fiber.Config{PassLocalsToContext: true})
	app.Get("/protected", New(), protected)
	return app
}

func request(t *testing.T, app *fiber.App, authorization string) int {
	t.Helper()
	req := httptest.NewRequest("GET", "/protected", nil)
	if authorization != "" {
		req.Header.Set(fiber.HeaderAuthorization, authorization)
	}
	response, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode == 401 && response.Header.Get(fiber.HeaderWWWAuthenticate) != bearerScheme {
		t.Fatalf("401 without %s: %s", fiber.HeaderWWWAuthenticate, bearerScheme)
	}
	return response.StatusCode
}

func token(payload string) string {
	encode := base64.RawURLEncoding.EncodeToString
	return encode([]byte(`{"alg":"HS256","typ":"JWT"}`)) + "." + encode([]byte(payload)) + ".signature"
}
