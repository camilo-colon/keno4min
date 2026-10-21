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
	"github.com/cronos/keno4min/internal/users"
	"github.com/cronos/keno4min/internal/users/ports"
)

var _ ports.Repository = (*Repository)(nil)

type Repository struct {
	client    *dynamodb.Client
	tableName string
}

func NewDynamodbRepository(client *dynamodb.Client, cfg *config.Config) *Repository {
	return &Repository{client: client, tableName: cfg.Dynamodb.TableName}
}

func (r Repository) FindById(ctx context.Context, id uuid.UUID) (*users.User, error) {
	output, err := r.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(r.tableName),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: "USER#" + id.String()},
			"SK": &types.AttributeValueMemberS{Value: "META"},
		},
		ConsistentRead: aws.Bool(true),
	})
	if err != nil {
		return nil, fmt.Errorf("find user %q: %w", id, err)
	}
	if len(output.Item) == 0 {
		return nil, nil
	}
	var item userItem
	if err := attributevalue.UnmarshalMap(output.Item, &item); err != nil {
		return nil, fmt.Errorf("unmarshal user %q: %w", id, err)
	}
	user, err := fromItem(item)
	if err != nil {
		return nil, fmt.Errorf("decode user %q: %w", id, err)
	}
	return user, nil
}

func (r Repository) FindByUsername(ctx context.Context, username string) (*users.User, error) {
	output, err := r.client.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(r.tableName),
		IndexName:              aws.String("GSI1"),
		KeyConditionExpression: aws.String("GSI1PK = :pk AND GSI1SK = :sk"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk": &types.AttributeValueMemberS{Value: "USER#" + username},
			":sk": &types.AttributeValueMemberS{Value: "META"},
		},
		Limit: aws.Int32(1),
	})
	if err != nil {
		return nil, fmt.Errorf("find user by username %q: %w", username, err)
	}
	if len(output.Items) == 0 {
		return nil, nil
	}
	var item userItem
	if err := attributevalue.UnmarshalMap(output.Items[0], &item); err != nil {
		return nil, fmt.Errorf("unmarshal user by username %q: %w", username, err)
	}
	user, err := fromItem(item)
	if err != nil {
		return nil, fmt.Errorf("decode user by username %q: %w", username, err)
	}
	return user, nil
}

func (r Repository) Save(ctx context.Context, user *users.User) error {
	item, err := attributevalue.MarshalMap(toItem(user))
	if err != nil {
		return fmt.Errorf("marshal user %q: %w", user.ID, err)
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
		return fmt.Errorf("save user %q: %w", user.ID, err)
	}
	return nil
}
