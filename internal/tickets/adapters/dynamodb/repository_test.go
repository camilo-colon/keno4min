package dynamodb

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"uuid"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/cronos/keno4min/internal/config"
	"github.com/cronos/keno4min/internal/tickets"
	"github.com/cronos/keno4min/internal/tickets/events"
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

func TestSaveWritesTicketCouponAndOutboxTogether(t *testing.T) {
	ticket, err := tickets.Create(uuid.NewV7(), "0001", uuid.NewV7(), 42, []tickets.Bet{{Nums: []uint{1, 80}, ValueCents: 1000}}, "seller-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var request struct {
		TransactItems []struct {
			Put struct {
				TableName           string
				ConditionExpression string
				Item                map[string]map[string]any
			}
		}
	}
	calls := 0
	repository := newTestRepository(func(req *http.Request) (*http.Response, error) {
		calls++
		if target := req.Header.Get("X-Amz-Target"); target != "DynamoDB_20120810.TransactWriteItems" {
			t.Errorf("unexpected operation: %s", target)
		}
		if err := json.NewDecoder(req.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": {"application/x-amz-json-1.0"}},
			Body:       io.NopCloser(strings.NewReader(`{}`)),
		}, nil
	})

	if err := repository.Save(context.Background(), ticket); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 1 || len(request.TransactItems) != 3 {
		t.Fatalf("calls = %d, items = %d, want one transaction with ticket, coupon and outbox", calls, len(request.TransactItems))
	}
	for _, item := range request.TransactItems {
		if item.Put.TableName != "keno-test" || item.Put.ConditionExpression != "attribute_not_exists(PK)" {
			t.Fatalf("unexpected put %+v", item.Put)
		}
	}

	wantKeys := []string{"TICKET#" + ticket.ID.String(), "COUPON#0001", "OUTBOX#"}
	for i, want := range wantKeys {
		if pk, _ := request.TransactItems[i].Put.Item["PK"]["S"].(string); !strings.HasPrefix(pk, want) {
			t.Fatalf("item %d PK = %q, want prefix %q", i, pk, want)
		}
	}
	outbox := request.TransactItems[2].Put.Item
	if outbox["eventType"]["S"] != events.TicketCreatedType || outbox["entityType"]["S"] != "OUTBOX" {
		t.Fatalf("unexpected outbox item %v", outbox)
	}
	if len(ticket.PullDomainEvents()) != 0 {
		t.Fatal("the saved ticket must not keep its events")
	}
}
