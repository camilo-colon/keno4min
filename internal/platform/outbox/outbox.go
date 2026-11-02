package outbox

import (
	"errors"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/cronos/keno4min/internal/common"
)

const (
	source    = "cronos.keno4min"
	retention = 7 * 24 * time.Hour
	// DynamoDB rejects transactions with more than 100 actions.
	maxTransactItems = 100
)

var ErrTooManyEvents = errors.New("too many domain events for one transaction")

// TransactItems pulls the aggregate events and returns one conditional Put per event.
// Each item expires a retention period after its event occurred.
// The caller must keep the whole transaction within the DynamoDB action limit.
func TransactItems(tableName string, aggregate common.Aggregate) ([]types.TransactWriteItem, error) {
	events := aggregate.PullDomainEvents()
	if len(events) > maxTransactItems {
		return nil, fmt.Errorf("%w: %d", ErrTooManyEvents, len(events))
	}

	items := make([]types.TransactWriteItem, 0, len(events))
	for _, event := range events {
		item, err := toItem(event)
		if err != nil {
			return nil, err
		}
		av, err := attributevalue.MarshalMap(item)
		if err != nil {
			return nil, fmt.Errorf("marshal outbox item %s: %w", item.EventID, err)
		}
		items = append(items, types.TransactWriteItem{Put: &types.Put{
			TableName:           aws.String(tableName),
			Item:                av,
			ConditionExpression: aws.String("attribute_not_exists(PK)"),
		}})
	}
	return items, nil
}
