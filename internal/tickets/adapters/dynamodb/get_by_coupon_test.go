package dynamodb

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
	"uuid"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/cronos/keno4min/internal/tickets/ports"
)

func TestGetByCouponReadsTheReservationAndThenTheTicket(t *testing.T) {
	ticketID := uuid.NewV7()
	repository, requested := newGetItemRepository(t, map[string]string{
		"COUPON#0001":                 `{"Item":{"PK":{"S":"COUPON#0001"},"SK":{"S":"META"},"ticketId":{"S":"` + ticketID.String() + `"}}}`,
		"TICKET#" + ticketID.String(): `{"Item":` + ticketItemJSON(ticketID) + `}`,
	})

	ticket, err := repository.GetByCoupon(t.Context(), "0001")
	if err != nil {
		t.Fatal(err)
	}
	if ticket.ID != ticketID || ticket.Coupon != "0001" {
		t.Fatalf("ticket = %s / %s, want %s / 0001", ticket.ID, ticket.Coupon, ticketID)
	}
	if want := []string{"COUPON#0001", "TICKET#" + ticketID.String()}; !slices.Equal(*requested, want) {
		t.Fatalf("requested keys = %v, want %v", *requested, want)
	}
}

func TestGetByCouponOfMissingCouponIsNotFound(t *testing.T) {
	repository, requested := newGetItemRepository(t, nil)

	_, err := repository.GetByCoupon(t.Context(), "9999")
	if !errors.Is(err, ports.ErrTicketNotFound) {
		t.Fatalf("error = %v, want ErrTicketNotFound", err)
	}
	if want := []string{"COUPON#9999"}; !slices.Equal(*requested, want) {
		t.Fatalf("requested keys = %v, want %v", *requested, want)
	}
}

// newGetItemRepository answers each GetItem with the body registered for its PK, or an empty item.
func newGetItemRepository(t *testing.T, bodies map[string]string) (Repository, *[]string) {
	var requested []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if target := r.Header.Get("X-Amz-Target"); target != "DynamoDB_20120810.GetItem" {
			t.Errorf("operation = %s, want GetItem", target)
		}
		var input struct {
			Key            map[string]capturedValue
			ConsistentRead bool
		}
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			t.Error(err)
		}
		if !input.ConsistentRead {
			t.Errorf("GetItem of %s is not a strong read", input.Key["PK"].S)
		}
		requested = append(requested, input.Key["PK"].S)

		body, ok := bodies[input.Key["PK"].S]
		if !ok {
			body = "{}"
		}
		w.Header().Set("Content-Type", "application/x-amz-json-1.0")
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)

	client := dynamodb.New(dynamodb.Options{
		Region:       "us-east-1",
		BaseEndpoint: aws.String(server.URL),
		Credentials:  aws.AnonymousCredentials{},
	})
	return Repository{client: client, tableName: "tickets-test"}, &requested
}

func ticketItemJSON(id uuid.UUID) string {
	return `{
		"PK": {"S": "TICKET#` + id.String() + `"}, "SK": {"S": "META"},
		"ticketId": {"S": "` + id.String() + `"}, "coupon": {"S": "0001"},
		"nodeId": {"S": "` + uuid.NewV7().String() + `"}, "tir": {"N": "4321"},
		"bets": {"L": [{"M": {"nums": {"L": [{"N": "1"}, {"N": "2"}]}, "valueCents": {"N": "500"}}}]},
		"status": {"S": "pending"}, "prizeCents": {"N": "0"},
		"createdBy": {"S": "cashier"}, "createdAt": {"S": "2026-11-21T12:00:00.000Z"}
	}`
}
