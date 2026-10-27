package dynamodb

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"math"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/cronos/keno4min/internal/config"
	"github.com/cronos/keno4min/internal/tickets"
)

type httpClientFunc func(*http.Request) (*http.Response, error)

func (f httpClientFunc) Do(req *http.Request) (*http.Response, error) {
	return f(req)
}

func newTestRepository(send httpClientFunc) *Repository {
	client := dynamodb.NewFromConfig(aws.Config{
		Region:      "us-east-1",
		Credentials: aws.AnonymousCredentials{},
		HTTPClient:  send,
		Retryer:     func() aws.Retryer { return aws.NopRetryer{} },
	})
	return NewDynamodbRepository(client, &config.Config{
		Dynamodb: config.DynamodbConfig{TableName: "keno-test"},
	})
}

func jsonResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": {"application/x-amz-json-1.0"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func TestGenerateCouponUsesAtomicCounterAndPadding(t *testing.T) {
	for _, tt := range []struct {
		value string
		want  string
	}{
		{value: "1", want: "0001"},
		{value: "42", want: "0042"},
		{value: "9999", want: "9999"},
		{value: "10000", want: "10000"},
		{value: "9999999999999999", want: "9999999999999999"},
	} {
		t.Run(tt.value, func(t *testing.T) {
			type contextKey struct{}
			ctx := context.WithValue(context.Background(), contextKey{}, "counter")
			calls := 0
			repository := newTestRepository(func(req *http.Request) (*http.Response, error) {
				calls++
				if req.Header.Get("X-Amz-Target") != "DynamoDB_20120810.UpdateItem" {
					t.Fatalf("expected atomic update, got %s", req.Header.Get("X-Amz-Target"))
				}
				if req.Context().Value(contextKey{}) != "counter" {
					t.Fatal("caller context was not forwarded")
				}
				var input struct {
					TableName, UpdateExpression, ConditionExpression, ReturnValues string
					Key, ExpressionAttributeValues                                 map[string]map[string]string
					ExpressionAttributeNames                                       map[string]string
				}
				if err := json.NewDecoder(req.Body).Decode(&input); err != nil {
					t.Fatal(err)
				}
				if input.TableName != "keno-test" || input.UpdateExpression != "ADD #value :one" || input.ReturnValues != "UPDATED_NEW" {
					t.Fatalf("expected increment returning its own updated value: %+v", input)
				}
				if input.ConditionExpression != "attribute_exists(PK) AND #value >= :zero AND #value < :max" {
					t.Fatalf("expected protection for missing or exhausted counter: %+v", input)
				}
				wantKey := map[string]map[string]string{
					"PK": {"S": "COUNTER#TICKET"}, "SK": {"S": "META"},
				}
				wantValues := map[string]map[string]string{
					":zero": {"N": "0"}, ":one": {"N": "1"}, ":max": {"N": "9999999999999999"},
				}
				if !reflect.DeepEqual(input.Key, wantKey) || !reflect.DeepEqual(input.ExpressionAttributeValues, wantValues) ||
					!reflect.DeepEqual(input.ExpressionAttributeNames, map[string]string{"#value": "value"}) {
					t.Fatalf("unexpected counter key or numeric parameters: %+v", input)
				}
				return jsonResponse(http.StatusOK, `{"Attributes":{"value":{"N":"`+tt.value+`"}}}`), nil
			})
			coupon, err := repository.GenerateCoupon(ctx)
			if err != nil || coupon != tt.want || calls != 1 {
				t.Fatalf("GenerateCoupon = (%q, %v), calls=%d; want %q in one update", coupon, err, calls, tt.want)
			}
		})
	}
}

func TestGenerateCouponRejectsInvalidCounterResponse(t *testing.T) {
	for _, tt := range []struct {
		name string
		body string
	}{
		{name: "missing value", body: `{}`},
		{name: "wrong type", body: `{"Attributes":{"value":{"S":"1"}}}`},
		{name: "invalid number", body: `{"Attributes":{"value":{"N":"invalid"}}}`},
		{name: "fractional", body: `{"Attributes":{"value":{"N":"1.5"}}}`},
		{name: "negative", body: `{"Attributes":{"value":{"N":"-1"}}}`},
		{name: "zero", body: `{"Attributes":{"value":{"N":"0"}}}`},
		{name: "above maximum", body: `{"Attributes":{"value":{"N":"10000000000000000"}}}`},
		{name: "uint64 overflow", body: `{"Attributes":{"value":{"N":"18446744073709551616"}}}`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			repository := newTestRepository(func(*http.Request) (*http.Response, error) {
				return jsonResponse(http.StatusOK, tt.body), nil
			})
			coupon, err := repository.GenerateCoupon(context.Background())
			if err == nil || coupon != "" {
				t.Fatalf("expected empty coupon and error, got (%q, %v)", coupon, err)
			}
		})
	}
}

func TestGenerateCouponPropagatesInfrastructureErrors(t *testing.T) {
	failure := errors.New("connection failed")
	for _, tt := range []struct {
		name  string
		send  httpClientFunc
		check func(error) bool
	}{
		{
			name: "counter missing or exhausted",
			send: func(*http.Request) (*http.Response, error) {
				return jsonResponse(http.StatusBadRequest, `{"__type":"com.amazonaws.dynamodb.v20120810#ConditionalCheckFailedException","message":"condition failed"}`), nil
			},
			check: func(err error) bool {
				var condition *types.ConditionalCheckFailedException
				return errors.As(err, &condition)
			},
		},
		{
			name: "table missing",
			send: func(*http.Request) (*http.Response, error) {
				return jsonResponse(http.StatusBadRequest, `{"__type":"com.amazonaws.dynamodb.v20120810#ResourceNotFoundException","message":"table missing"}`), nil
			},
			check: func(err error) bool {
				var missing *types.ResourceNotFoundException
				return errors.As(err, &missing)
			},
		},
		{
			name:  "transport failed",
			send:  func(*http.Request) (*http.Response, error) { return nil, failure },
			check: func(err error) bool { return errors.Is(err, failure) },
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			calls := 0
			repository := newTestRepository(func(req *http.Request) (*http.Response, error) {
				calls++
				return tt.send(req)
			})
			coupon, err := repository.GenerateCoupon(context.Background())
			if !tt.check(err) || coupon != "" || calls != 1 {
				t.Fatalf("expected original error without recreation or extra writes: coupon=%q err=%v calls=%d", coupon, err, calls)
			}
		})
	}
}

func TestGenerateCouponHonorsCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	repository := newTestRepository(func(*http.Request) (*http.Response, error) {
		t.Fatal("cancelled request must not reach DynamoDB")
		return nil, nil
	})
	coupon, err := repository.GenerateCoupon(ctx)
	if !errors.Is(err, context.Canceled) || coupon != "" {
		t.Fatalf("expected context cancellation, got (%q, %v)", coupon, err)
	}
}

func sampleTicket() *tickets.Ticket {
	return &tickets.Ticket{
		ID:     uuid.MustParse("019a052b-3000-7000-8000-000000000001"),
		Coupon: "0001",
		NodeID: uuid.MustParse("f2a0dc45-9ab8-4553-bd7c-2d53b345a5f6"),
		TIR:    math.MaxUint64,
		Bets: []tickets.Bet{
			{Nums: []uint8{1, 80}, ValueCents: math.MaxInt64 / 2},
			{Nums: []uint8{1, 80}, ValueCents: math.MaxInt64 / 2},
		},
		Status:    tickets.StatusPending,
		CreatedBy: "SellerName",
		CreatedAt: time.Date(2026, 10, 21, 8, 30, 45, 123456789, time.FixedZone("Bogota", -5*60*60)),
	}
}

func TestSaveWritesTicketAndCouponInOneTransaction(t *testing.T) {
	ticket := sampleTicket()
	type contextKey struct{}
	ctx := context.WithValue(context.Background(), contextKey{}, "sale")
	calls := 0
	repository := newTestRepository(func(req *http.Request) (*http.Response, error) {
		calls++
		if req.Header.Get("X-Amz-Target") != "DynamoDB_20120810.TransactWriteItems" {
			t.Fatalf("unexpected operation: %s", req.Header.Get("X-Amz-Target"))
		}
		if req.Context().Value(contextKey{}) != "sale" {
			t.Fatal("caller context was not forwarded")
		}
		var input struct {
			TransactItems []struct {
				Put struct {
					TableName           string
					ConditionExpression string
					Item                map[string]any
				}
			}
		}
		if err := json.NewDecoder(req.Body).Decode(&input); err != nil {
			t.Fatal(err)
		}
		if len(input.TransactItems) != 2 {
			t.Fatalf("expected ticket and coupon in one transaction, got %+v", input)
		}
		for _, action := range input.TransactItems {
			if action.Put.TableName != "keno-test" || action.Put.ConditionExpression != "attribute_not_exists(PK)" {
				t.Fatalf("expected conditional put in configured table: %+v", action)
			}
		}
		bet := map[string]any{"M": map[string]any{
			"nums":       map[string]any{"L": []any{map[string]any{"N": "1"}, map[string]any{"N": "80"}}},
			"valueCents": map[string]any{"N": "4611686018427387903"},
		}}
		wantTicket := map[string]any{
			"PK":        map[string]any{"S": "TICKET#" + ticket.ID.String()},
			"SK":        map[string]any{"S": "META"},
			"GSI1PK":    map[string]any{"S": "TICKETS#TIR#18446744073709551615"},
			"GSI1SK":    map[string]any{"S": "TICKET#" + ticket.ID.String()},
			"GSI2PK":    map[string]any{"S": "TICKETS#NODE#" + ticket.NodeID.String()},
			"ticketId":  map[string]any{"S": ticket.ID.String()},
			"coupon":    map[string]any{"S": "0001"},
			"nodeId":    map[string]any{"S": ticket.NodeID.String()},
			"tir":       map[string]any{"N": "18446744073709551615"},
			"bets":      map[string]any{"L": []any{bet, bet}},
			"status":    map[string]any{"S": "pending"},
			"createdBy": map[string]any{"S": "SellerName"},
			"createdAt": map[string]any{"N": "1792589445123"},
		}
		wantCoupon := map[string]any{
			"PK":       map[string]any{"S": "COUPON#0001"},
			"SK":       map[string]any{"S": "META"},
			"ticketId": map[string]any{"S": ticket.ID.String()},
		}
		if !reflect.DeepEqual(input.TransactItems[0].Put.Item, wantTicket) {
			t.Fatalf("unexpected ticket item: got %#v, want %#v", input.TransactItems[0].Put.Item, wantTicket)
		}
		if !reflect.DeepEqual(input.TransactItems[1].Put.Item, wantCoupon) {
			t.Fatalf("unexpected coupon item: got %#v, want %#v", input.TransactItems[1].Put.Item, wantCoupon)
		}
		return jsonResponse(http.StatusOK, `{}`), nil
	})
	createdAt := ticket.CreatedAt
	if err := repository.Save(ctx, ticket); err != nil {
		t.Fatal(err)
	}
	if calls != 1 || ticket.CreatedAt != createdAt {
		t.Fatalf("expected one transaction without changing ticket: calls=%d ticket=%+v", calls, ticket)
	}
}

func TestSavePropagatesDuplicateTicketAndCoupon(t *testing.T) {
	for _, tt := range []struct {
		name    string
		reasons string
	}{
		{name: "ticket exists", reasons: `[{"Code":"ConditionalCheckFailed"},{"Code":"None"}]`},
		{name: "coupon reserved", reasons: `[{"Code":"None"},{"Code":"ConditionalCheckFailed"}]`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			calls := 0
			repository := newTestRepository(func(*http.Request) (*http.Response, error) {
				calls++
				return jsonResponse(http.StatusBadRequest,
					`{"__type":"com.amazonaws.dynamodb.v20120810#TransactionCanceledException","Message":"condition failed","CancellationReasons":`+tt.reasons+`}`), nil
			})
			ticket := sampleTicket()
			err := repository.Save(context.Background(), ticket)
			var cancelled *types.TransactionCanceledException
			if !errors.As(err, &cancelled) || len(cancelled.CancellationReasons) != 2 || !strings.Contains(err.Error(), ticket.ID.String()) {
				t.Fatalf("expected cancellation with original reasons and ticket identity: %v", err)
			}
			if calls != 1 {
				t.Fatalf("expected no fallback write after conflict, got %d calls", calls)
			}
		})
	}
}

func TestSavePreservesInfrastructureErrors(t *testing.T) {
	failure := errors.New("connection failed")
	for _, tt := range []struct {
		name  string
		send  httpClientFunc
		check func(error) bool
	}{
		{
			name: "table missing",
			send: func(*http.Request) (*http.Response, error) {
				return jsonResponse(http.StatusBadRequest, `{"__type":"com.amazonaws.dynamodb.v20120810#ResourceNotFoundException","message":"table missing"}`), nil
			},
			check: func(err error) bool {
				var missing *types.ResourceNotFoundException
				return errors.As(err, &missing)
			},
		},
		{
			name: "transport failed",
			send: func(*http.Request) (*http.Response, error) { return nil, failure },
			check: func(err error) bool {
				return errors.Is(err, failure)
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ticket := sampleTicket()
			err := newTestRepository(tt.send).Save(context.Background(), ticket)
			if !tt.check(err) || !strings.Contains(err.Error(), ticket.ID.String()) {
				t.Fatalf("expected original error with ticket identity: %v", err)
			}
		})
	}
}

func TestSaveRejectsNilTicket(t *testing.T) {
	repository := newTestRepository(func(*http.Request) (*http.Response, error) {
		t.Fatal("nil ticket must not reach DynamoDB")
		return nil, nil
	})
	if err := repository.Save(context.Background(), nil); err == nil {
		t.Fatal("expected error for nil ticket")
	}
}

func TestSaveHonorsCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	repository := newTestRepository(func(*http.Request) (*http.Response, error) {
		t.Fatal("cancelled request must not reach DynamoDB")
		return nil, nil
	})
	if err := repository.Save(ctx, sampleTicket()); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context cancellation, got %v", err)
	}
}
