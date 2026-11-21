package dynamodb

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/cronos/keno4min/internal/tickets"
)

type capturedTransaction struct {
	TransactItems []capturedWrite
}

type capturedWrite struct {
	Put    *capturedPut
	Delete *capturedDelete
}

type capturedPut struct {
	Item map[string]capturedValue
}

type capturedDelete struct {
	Key map[string]capturedValue
}

type capturedValue struct {
	S string
	B []byte
	N string
}

func TestSaveWritesExactlyOneBetsRecord(t *testing.T) {
	repository, captured := newCapturingRepository(t)
	ticket := newPendingTicket(t)

	if err := repository.Save(t.Context(), ticket); err != nil {
		t.Fatal(err)
	}

	records := betsRecordPuts(captured)
	if len(records) != 1 {
		t.Fatalf("bets records written = %d, want 1", len(records))
	}
	record := records[0]
	if record["PK"].S != "BETS#4321" {
		t.Fatalf("PK = %q, want BETS#4321", record["PK"].S)
	}
	if wantSK := ticket.AdminID.String() + "#" + ticket.ID.String(); record["SK"].S != wantSK {
		t.Fatalf("SK = %q, want %q", record["SK"].S, wantSK)
	}
	wantBets := []byte{
		0x01, 0, 0, 0, 0, 0, 0, 0x80,
		0x01, 0x80, 0, 0, 0, 0, 0, 0,
		0x02, 0x01, 0, 0, 0, 0, 0, 0,
	}
	if !bytes.Equal(record["b"].B, wantBets) {
		t.Fatalf("b = % x, want % x", record["b"].B, wantBets)
	}
	wantTTL := strconv.FormatInt(ticket.CreatedAt.Add(24*time.Hour).Unix(), 10)
	if record["ttl"].N != wantTTL {
		t.Fatalf("ttl = %s, want %s", record["ttl"].N, wantTTL)
	}
	if deleted := betsRecordDeletes(captured); len(deleted) != 0 {
		t.Fatalf("bets records deleted = %d, want 0", len(deleted))
	}
}

func TestUpdateOfCancelledTicketDeletesItsBetsRecord(t *testing.T) {
	repository, captured := newCapturingRepository(t)
	ticket := newPendingTicket(t)
	if err := ticket.Cancel("cashier", time.Now()); err != nil {
		t.Fatal(err)
	}

	if err := repository.Update(t.Context(), ticket); err != nil {
		t.Fatal(err)
	}

	deleted := betsRecordDeletes(captured)
	if len(deleted) != 1 {
		t.Fatalf("bets records deleted = %d, want 1", len(deleted))
	}
	if deleted[0]["PK"].S != "BETS#4321" {
		t.Fatalf("PK = %q, want BETS#4321", deleted[0]["PK"].S)
	}
	if wantSK := ticket.AdminID.String() + "#" + ticket.ID.String(); deleted[0]["SK"].S != wantSK {
		t.Fatalf("SK = %q, want %q", deleted[0]["SK"].S, wantSK)
	}
}

func TestUpdateOfPaidTicketKeepsItsBetsRecord(t *testing.T) {
	repository, captured := newCapturingRepository(t)
	ticket := newPendingTicket(t)
	ticket.Status = tickets.StatusPaid

	if err := repository.Update(t.Context(), ticket); err != nil {
		t.Fatal(err)
	}

	if len(captured.TransactItems) == 0 {
		t.Fatal("no transaction was sent")
	}
	if deleted := betsRecordDeletes(captured); len(deleted) != 0 {
		t.Fatalf("bets records deleted = %d, want 0", len(deleted))
	}
}

func newCapturingRepository(t *testing.T) (Repository, *capturedTransaction) {
	captured := &capturedTransaction{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(captured); err != nil {
			t.Error(err)
		}
		w.Header().Set("Content-Type", "application/x-amz-json-1.0")
		_, _ = w.Write([]byte("{}"))
	}))
	t.Cleanup(server.Close)

	client := dynamodb.New(dynamodb.Options{
		Region:       "us-east-1",
		BaseEndpoint: aws.String(server.URL),
		Credentials:  aws.AnonymousCredentials{},
	})
	return Repository{client: client, tableName: "tickets-test"}, captured
}

func newPendingTicket(t *testing.T) *tickets.Ticket {
	bets := []tickets.Bet{{Nums: []uint{1, 64, 65, 80}, ValueCents: 258}}
	ticket, err := tickets.Create(uuid.NewV7(), "0001", uuid.NewV7(), uuid.NewV7(), 4321, bets, "cashier")
	if err != nil {
		t.Fatal(err)
	}
	return ticket
}

func betsRecordPuts(captured *capturedTransaction) []map[string]capturedValue {
	var records []map[string]capturedValue
	for _, write := range captured.TransactItems {
		if write.Put != nil && strings.HasPrefix(write.Put.Item["PK"].S, "BETS#") {
			records = append(records, write.Put.Item)
		}
	}
	return records
}

func betsRecordDeletes(captured *capturedTransaction) []map[string]capturedValue {
	var keys []map[string]capturedValue
	for _, write := range captured.TransactItems {
		if write.Delete != nil && strings.HasPrefix(write.Delete.Key["PK"].S, "BETS#") {
			keys = append(keys, write.Delete.Key)
		}
	}
	return keys
}
