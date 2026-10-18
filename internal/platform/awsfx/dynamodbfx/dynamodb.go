package dynamodbfx

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/cronos/keno4min/internal/config"
)

func NewDynamodbClient(awsCfg aws.Config, cfg *config.Config) *dynamodb.Client {
	return dynamodb.NewFromConfig(awsCfg, func(o *dynamodb.Options) {
		if cfg.Dynamodb.Endpoint != "" {
			o.BaseEndpoint = aws.String(cfg.Dynamodb.Endpoint)
		}
	})
}
