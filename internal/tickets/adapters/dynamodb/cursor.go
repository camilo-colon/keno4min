package dynamodb

import (
	"encoding/base64"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/cronos/keno4min/internal/tickets/ports"
)

func decodeCursor(cursor string) (map[string]types.AttributeValue, error) {
	if cursor == "" {
		return nil, nil
	}
	if len(cursor) > 4096 {
		return nil, ports.ErrInvalidCursor
	}
	raw, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return nil, ports.ErrInvalidCursor
	}
	key, err := attributevalue.UnmarshalMapJSON(raw)
	if err != nil || len(key) == 0 {
		return nil, ports.ErrInvalidCursor
	}
	return key, nil
}

func encodeCursor(key map[string]types.AttributeValue) (string, error) {
	if len(key) == 0 {
		return "", nil
	}
	raw, err := attributevalue.MarshalMapJSON(key)
	if err != nil {
		return "", fmt.Errorf("encode pagination key: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}
