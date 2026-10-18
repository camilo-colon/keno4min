package dynamodb

import (
	"context"
	"time"
	"uuid"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/cronos/keno4min/internal/config"
	"github.com/cronos/keno4min/internal/draws/ports"
)

type DynamodbRepository struct {
	client    *dynamodb.Client
	tableName string
}

func NewRepository(client *dynamodb.Client, cfg *config.Config) ports.Repository {
	return &DynamodbRepository{client: client, tableName: cfg.Dynamodb.TableName}
}

func (r *DynamodbRepository) GetRuntime(ctx context.Context) (*ports.Runtime, error) {
	return &ports.Runtime{
		Current: ports.CurrentDraw{
			DrawId:   uuid.NewV7(),
			Tir:      0,
			StartsAt: time.Now(),
			ClosesAt: time.Now(),
		},
		LatestResult: nil,
	}, nil
}
