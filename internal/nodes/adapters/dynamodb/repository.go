package dynamodb

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/cronos/keno4min/internal/config"
	"github.com/cronos/keno4min/internal/nodes"
	"github.com/cronos/keno4min/internal/nodes/ports"
)

var _ ports.Repository = (*DynamodbRepository)(nil)

type DynamodbRepository struct {
	client    *dynamodb.Client
	tableName string
}

func NewDynamodbRepository(client *dynamodb.Client, cfg *config.Config) *DynamodbRepository {
	return &DynamodbRepository{
		client:    client,
		tableName: cfg.Dynamodb.TableName,
	}
}

func (r DynamodbRepository) Save(context context.Context, node *nodes.Node) error {
	return nil
}
