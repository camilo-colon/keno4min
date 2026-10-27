package dynamodb

import (
	"context"
	"fmt"
	"uuid"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/cronos/keno4min/internal/config"
	draw "github.com/cronos/keno4min/internal/draws"
	"github.com/cronos/keno4min/internal/draws/ports"
)

type DynamodbRepository struct {
	client    *dynamodb.Client
	tableName string
}

func NewRepository(client *dynamodb.Client, cfg *config.Config) ports.Repository {
	return &DynamodbRepository{client: client, tableName: cfg.Dynamodb.TableName}
}

func (r *DynamodbRepository) GetCurrent(ctx context.Context) (*draw.Draw, error) {
	runtime, err := r.GetRuntime(ctx)
	if err != nil {
		return nil, err
	}
	current, err := r.FindByID(ctx, runtime.Current.DrawId)
	if err != nil || current == nil {
		return current, err
	}
	return current, nil
}

func (r *DynamodbRepository) FindByID(ctx context.Context, id uuid.UUID) (*draw.Draw, error) {
	output, err := r.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(r.tableName),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: "DRAW#" + id.String()},
			"SK": &types.AttributeValueMemberS{Value: "META"},
		},
		ConsistentRead: aws.Bool(true),
	})
	if err != nil {
		return nil, fmt.Errorf("find draw %s: %w", id, err)
	}
	if len(output.Item) == 0 {
		return nil, nil
	}
	var item drawItem
	if err := attributevalue.UnmarshalMap(output.Item, &item); err != nil {
		return nil, fmt.Errorf("unmarshal draw %s: %w", id, err)
	}
	result, err := fromDrawItem(item)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetLatestDraws returns up to ten results in descending GSI1SK order.
func (r *DynamodbRepository) GetLatestDraws(ctx context.Context) ([]draw.Draw, error) {
	output, err := r.client.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(r.tableName),
		IndexName:              aws.String("GSI1"),
		KeyConditionExpression: aws.String("GSI1PK = :pk"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk": &types.AttributeValueMemberS{Value: "DRAW#DRAWN"},
		},
		ScanIndexForward: aws.Bool(false),
		Limit:            aws.Int32(10),
	})
	if err != nil {
		return nil, fmt.Errorf("get latest draws: %w", err)
	}
	results := make([]draw.Draw, 0, len(output.Items))
	for _, attributes := range output.Items {
		var item drawItem
		if err := attributevalue.UnmarshalMap(attributes, &item); err != nil {
			return nil, fmt.Errorf("unmarshal latest draw: %w", err)
		}
		result, err := fromDrawItem(item)
		if err != nil {
			return nil, err
		}
		results = append(results, result)
	}
	return results, nil
}

func (r *DynamodbRepository) GetRuntime(ctx context.Context) (*ports.Runtime, error) {
	output, err := r.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(r.tableName),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: "DRAW_RUNTIME"},
			"SK": &types.AttributeValueMemberS{Value: "META"},
		},
		ConsistentRead: aws.Bool(true),
	})
	if err != nil {
		return nil, fmt.Errorf("get draw runtime: %w", err)
	}
	if len(output.Item) == 0 {
		return nil, fmt.Errorf("draw runtime not found")
	}

	var item runtimeItem
	if err := attributevalue.UnmarshalMap(output.Item, &item); err != nil {
		return nil, fmt.Errorf("unmarshal draw runtime: %w", err)
	}

	return fromRuntimeItem(item)
}
