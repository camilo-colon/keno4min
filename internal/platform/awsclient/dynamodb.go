// Package awsclient builds AWS service clients shared by the process.
package awsclient

import (
	"context"
	"fmt"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	sdkconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	appconfig "github.com/cronos/keno4min/internal/config"
)

type configLoader func(context.Context, ...func(*sdkconfig.LoadOptions) error) (awssdk.Config, error)

// NewDynamoDB loads the standard AWS configuration chain and builds a DynamoDB
// client. Client construction is local and does not make a request to AWS.
func NewDynamoDB(
	ctx context.Context,
	awsConfig appconfig.AWSConfig,
	dynamoDBConfig appconfig.DynamoDBConfig,
) (*dynamodb.Client, error) {
	return newDynamoDB(ctx, awsConfig, dynamoDBConfig, sdkconfig.LoadDefaultConfig)
}

func newDynamoDB(
	ctx context.Context,
	awsConfig appconfig.AWSConfig,
	dynamoDBConfig appconfig.DynamoDBConfig,
	load configLoader,
) (*dynamodb.Client, error) {
	sdkConfig, err := load(ctx, sdkconfig.WithRegion(awsConfig.Region))
	if err != nil {
		return nil, fmt.Errorf("load AWS SDK configuration: %w", err)
	}

	return dynamodb.NewFromConfig(sdkConfig, func(options *dynamodb.Options) {
		if dynamoDBConfig.EndpointURL != "" {
			options.BaseEndpoint = &dynamoDBConfig.EndpointURL
		}
	}), nil
}
