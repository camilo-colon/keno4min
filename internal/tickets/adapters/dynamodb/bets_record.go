package dynamodb

import (
	"encoding/binary"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/cronos/keno4min/internal/tickets"
)

const (
	encodedBetSize     = 24
	betsRecordLifetime = 24 * time.Hour
)

type betsRecordItem struct {
	PK          string `dynamodbav:"PK"`
	SK          string `dynamodbav:"SK"`
	EncodedBets []byte `dynamodbav:"b"`
	ExpiresAt   int64  `dynamodbav:"ttl"`
}

func betsRecordPK(ticket *tickets.Ticket) string {
	return "BETS#" + strconv.FormatUint(ticket.TIR, 10)
}

func betsRecordSK(ticket *tickets.Ticket) string {
	return ticket.AdminID.String() + "#" + ticket.ID.String()
}

func betsRecordKey(ticket *tickets.Ticket) map[string]types.AttributeValue {
	return map[string]types.AttributeValue{
		"PK": &types.AttributeValueMemberS{Value: betsRecordPK(ticket)},
		"SK": &types.AttributeValueMemberS{Value: betsRecordSK(ticket)},
	}
}

func toBetsRecordItem(ticket *tickets.Ticket) betsRecordItem {
	return betsRecordItem{
		PK:          betsRecordPK(ticket),
		SK:          betsRecordSK(ticket),
		EncodedBets: encodeBets(ticket.Bets),
		ExpiresAt:   ticket.CreatedAt.Add(betsRecordLifetime).Unix(),
	}
}

func encodeBets(bets []tickets.Bet) []byte {
	encoded := make([]byte, 0, len(bets)*encodedBetSize)
	for _, bet := range bets {
		var numbersUpTo64, numbersFrom65 uint64
		for _, number := range bet.Nums {
			if number <= 64 {
				numbersUpTo64 |= uint64(1) << (number - 1)
			} else {
				numbersFrom65 |= uint64(1) << (number - 65)
			}
		}
		encoded = binary.LittleEndian.AppendUint64(encoded, numbersUpTo64)
		encoded = binary.LittleEndian.AppendUint64(encoded, numbersFrom65)
		encoded = binary.LittleEndian.AppendUint64(encoded, uint64(bet.ValueCents))
	}
	return encoded
}
