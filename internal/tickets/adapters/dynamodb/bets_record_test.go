package dynamodb

import (
	"bytes"
	"math"
	"math/rand/v2"
	"reflect"
	"testing"
	"time"
	"uuid"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/cronos/keno4min/internal/tickets"
)

func TestBetsRecordItemMatchesTheContract(t *testing.T) {
	ticket := &tickets.Ticket{
		ID:        uuid.MustParse("0193a5c0-0000-7000-8000-000000000001"),
		AdminID:   uuid.MustParse("0193a5c0-0000-7000-8000-000000000002"),
		TIR:       4321,
		Bets:      []tickets.Bet{{Nums: []uint{1, 64, 65, 80}, ValueCents: 258}},
		CreatedAt: time.Date(2026, time.November, 20, 15, 0, 0, 0, time.UTC),
	}

	got, err := attributevalue.MarshalMap(toBetsRecordItem(ticket))
	if err != nil {
		t.Fatal(err)
	}

	want := map[string]types.AttributeValue{
		"PK": &types.AttributeValueMemberS{Value: "BETS#4321"},
		"SK": &types.AttributeValueMemberS{Value: "0193a5c0-0000-7000-8000-000000000002#0193a5c0-0000-7000-8000-000000000001"},
		"b": &types.AttributeValueMemberB{Value: []byte{
			0x01, 0, 0, 0, 0, 0, 0, 0x80,
			0x01, 0x80, 0, 0, 0, 0, 0, 0,
			0x02, 0x01, 0, 0, 0, 0, 0, 0,
		}},
		"ttl": &types.AttributeValueMemberN{Value: "1795273200"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("item = %v, want %v", got, want)
	}
}

func TestBetsRecordKeyPointsToTheWrittenItem(t *testing.T) {
	ticket := &tickets.Ticket{
		ID:        uuid.NewV7(),
		AdminID:   uuid.NewV7(),
		TIR:       4321,
		Bets:      []tickets.Bet{{Nums: []uint{7}, ValueCents: 100}},
		CreatedAt: time.Now().UTC(),
	}

	item, err := attributevalue.MarshalMap(toBetsRecordItem(ticket))
	if err != nil {
		t.Fatal(err)
	}

	want := map[string]types.AttributeValue{"PK": item["PK"], "SK": item["SK"]}
	if got := betsRecordKey(ticket); !reflect.DeepEqual(got, want) {
		t.Fatalf("key = %v, want %v", got, want)
	}
}

func TestEncodeBetsAtTheBoundaryBetween64And65(t *testing.T) {
	bets := []tickets.Bet{{Nums: []uint{1, 64, 65, 80}, ValueCents: 258}}

	want := []byte{
		0x01, 0, 0, 0, 0, 0, 0, 0x80,
		0x01, 0x80, 0, 0, 0, 0, 0, 0,
		0x02, 0x01, 0, 0, 0, 0, 0, 0,
	}
	if got := encodeBets(bets); !bytes.Equal(got, want) {
		t.Fatalf("encoded = % x, want % x", got, want)
	}
}

func TestEncodeBetsMatchesTheSlowVersionOnRandomBets(t *testing.T) {
	random := rand.New(rand.NewPCG(1, 2))

	for range 10_000 {
		bets := randomBets(random)

		var want []byte
		for _, bet := range bets {
			want = append(want, slowEncodeBet(bet)...)
		}
		if got := encodeBets(bets); !bytes.Equal(got, want) {
			t.Fatalf("bets %v: encoded = % x, want % x", bets, got, want)
		}
	}
}

func slowEncodeBet(bet tickets.Bet) []byte {
	encoded := make([]byte, 24)
	for _, number := range bet.Nums {
		bit := number - 1
		encoded[bit/8] |= 1 << (bit % 8)
	}
	for i := range 8 {
		encoded[16+i] = byte(bet.ValueCents >> (8 * i))
	}
	return encoded
}

func randomBets(random *rand.Rand) []tickets.Bet {
	bets := make([]tickets.Bet, 1+random.IntN(20))
	for i := range bets {
		shuffled := random.Perm(80)
		nums := make([]uint, 1+random.IntN(10))
		for j := range nums {
			nums[j] = uint(shuffled[j] + 1)
		}
		bets[i] = tickets.Bet{Nums: nums, ValueCents: 1 + random.Int64N(math.MaxInt64)}
	}
	return bets
}
