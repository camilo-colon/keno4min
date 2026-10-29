package dynamodb

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/cronos/keno4min/internal/config"
	"github.com/cronos/keno4min/internal/tickets"
	"github.com/cronos/keno4min/internal/tickets/ports"
)

var _ ports.Repository = (*Repository)(nil)

type Repository struct {
	client    *dynamodb.Client
	tableName string
}

func NewDynamodbRepository(client *dynamodb.Client, cfg *config.Config) *Repository {
	return &Repository{client: client, tableName: cfg.Dynamodb.TableName}
}

func (r Repository) GenerateCoupon(ctx context.Context) (string, error) {
	const maxCounter uint64 = 9_999_999_999_999_999

	result, err := r.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(r.tableName),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: "COUNTER#TICKET"},
			"SK": &types.AttributeValueMemberS{Value: "META"},
		},
		UpdateExpression: aws.String("ADD #value :one"),
		ConditionExpression: aws.String(
			"attribute_exists(PK) AND #value >= :zero AND #value < :max",
		),
		ExpressionAttributeNames: map[string]string{
			"#value": "value",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":zero": &types.AttributeValueMemberN{Value: "0"},
			":one":  &types.AttributeValueMemberN{Value: "1"},
			":max":  &types.AttributeValueMemberN{Value: strconv.FormatUint(maxCounter, 10)},
		},
		ReturnValues: types.ReturnValueUpdatedNew,
	})
	if err != nil {
		return "", fmt.Errorf("increment coupon counter: %w", err)
	}

	value, ok := result.Attributes["value"].(*types.AttributeValueMemberN)
	if !ok {
		return "", errors.New("coupon counter value is missing or invalid")
	}

	counter, err := strconv.ParseUint(value.Value, 10, 64)
	if err != nil {
		return "", fmt.Errorf("decode coupon counter: %w", err)
	}
	if counter == 0 || counter > maxCounter {
		return "", errors.New("coupon counter is outside the allowed range")
	}

	return fmt.Sprintf("%04d", counter), nil
}

func (r Repository) Save(ctx context.Context, ticket *tickets.Ticket) error {
	if ticket == nil {
		return errors.New("ticket is required")
	}

	item, err := attributevalue.MarshalMap(toItem(ticket))
	if err != nil {
		return fmt.Errorf("marshal ticket %s: %w", ticket.ID, err)
	}
	coupon, err := attributevalue.MarshalMap(couponItem{
		PK:       "COUPON#" + ticket.Coupon,
		SK:       "META",
		TicketID: ticket.ID.String(),
	})
	if err != nil {
		return fmt.Errorf("marshal coupon for ticket %s: %w", ticket.ID, err)
	}

	_, err = r.client.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{
		TransactItems: []types.TransactWriteItem{
			{Put: &types.Put{
				TableName:           aws.String(r.tableName),
				Item:                item,
				ConditionExpression: aws.String("attribute_not_exists(PK)"),
			}},
			{Put: &types.Put{
				TableName:           aws.String(r.tableName),
				Item:                coupon,
				ConditionExpression: aws.String("attribute_not_exists(PK)"),
			}},
		},
	})
	if err != nil {
		return fmt.Errorf("save ticket %s: %w", ticket.ID, err)
	}
	return nil
}

func (r Repository) ListByNode(ctx context.Context, params ports.ListParams) (ports.TicketPage, error) {
	partition := "TICKETS#NODE#" + params.NodeID.String()

	startKey, err := decodeCursor(params.Cursor)
	if err != nil {
		return ports.TicketPage{}, err
	}
	if startKey != nil {
		// The cursor must resume the same node; otherwise it would leak another club's page.
		value, ok := startKey["GSI2PK"].(*types.AttributeValueMemberS)
		if !ok || value.Value != partition {
			return ports.TicketPage{}, ports.ErrInvalidCursor
		}
	}

	// BETWEEN includes both ends, so the exclusive upper bound becomes to-1ms.
	result, err := r.client.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(r.tableName),
		IndexName:              aws.String("GSI2"),
		KeyConditionExpression: aws.String("GSI2PK = :pk AND createdAt BETWEEN :from AND :to"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk":   &types.AttributeValueMemberS{Value: partition},
			":from": &types.AttributeValueMemberN{Value: strconv.FormatInt(params.From.UnixMilli(), 10)},
			":to":   &types.AttributeValueMemberN{Value: strconv.FormatInt(params.To.UnixMilli()-1, 10)},
		},
		ExclusiveStartKey: startKey,
		Limit:             aws.Int32(params.Limit),
	})
	if err != nil {
		return ports.TicketPage{}, fmt.Errorf("list tickets of node %s: %w", params.NodeID, err)
	}

	var items []ticketItem
	if err := attributevalue.UnmarshalListOfMaps(result.Items, &items); err != nil {
		return ports.TicketPage{}, fmt.Errorf("unmarshal tickets of node %s: %w", params.NodeID, err)
	}
	page := ports.TicketPage{Items: make([]*tickets.Ticket, len(items))}
	for i, item := range items {
		if page.Items[i], err = fromItem(item); err != nil {
			return ports.TicketPage{}, err
		}
	}
	if page.NextCursor, err = encodeCursor(result.LastEvaluatedKey); err != nil {
		return ports.TicketPage{}, err
	}
	return page, nil
}
