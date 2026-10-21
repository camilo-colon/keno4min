package dynamodb

import (
	"context"
	"errors"
	"fmt"
	"uuid"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/cronos/keno4min/internal/config"
	"github.com/cronos/keno4min/internal/nodes"
	"github.com/cronos/keno4min/internal/nodes/ports"
)

var _ ports.Repository = (*Repository)(nil)

type Repository struct {
	client    *dynamodb.Client
	tableName string
}

func NewDynamodbRepository(client *dynamodb.Client, cfg *config.Config) *Repository {
	return &Repository{
		client:    client,
		tableName: cfg.Dynamodb.TableName,
	}
}

func (r Repository) FindById(ctx context.Context, id uuid.UUID) (*nodes.Node, error) {
	id_str := id.String()
	fmt.Println(id_str)
	output, err := r.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(r.tableName),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: "NODE#" + id.String()},
			"SK": &types.AttributeValueMemberS{Value: "META"},
		},
		ConsistentRead: aws.Bool(true),
	})
	if err != nil {
		return nil, fmt.Errorf("find node %s: %w", id, err)
	}
	if len(output.Item) == 0 {
		return nil, nil
	}
	var item nodeItem
	if err := attributevalue.UnmarshalMap(output.Item, &item); err != nil {
		return nil, fmt.Errorf("unmarshal node %s: %w", id, err)
	}
	node, err := fromItem(item)
	if err != nil {
		return nil, fmt.Errorf("decode node %s: %w", id, err)
	}
	return node, nil
}

func (r Repository) Save(ctx context.Context, node *nodes.Node) error {
	item, err := attributevalue.MarshalMap(toItem(node))
	if err != nil {
		return fmt.Errorf("marshal node %s: %w", node.ID, err)
	}
	_, err = r.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName:           aws.String(r.tableName),
		Item:                item,
		ConditionExpression: aws.String("attribute_not_exists(PK)"),
	})
	if _, ok := errors.AsType[*types.ConditionalCheckFailedException](err); ok {
		return nil
	}
	if err != nil {
		return fmt.Errorf("save node %s: %w", node.ID, err)
	}
	return nil
}
