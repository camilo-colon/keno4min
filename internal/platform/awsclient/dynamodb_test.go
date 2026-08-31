package awsclient

import (
	"context"
	"errors"
	"strings"
	"testing"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	sdkconfig "github.com/aws/aws-sdk-go-v2/config"
	appconfig "github.com/cronos/keno4min/internal/config"
)

func TestNewDynamoDBConfiguresRegionAndEndpoint(t *testing.T) {
	ctx := context.Background()
	loader := func(receivedCtx context.Context, options ...func(*sdkconfig.LoadOptions) error) (awssdk.Config, error) {
		if receivedCtx != ctx {
			t.Fatal("configuration loader received a different context")
		}

		loadOptions := sdkconfig.LoadOptions{}
		for _, option := range options {
			if err := option(&loadOptions); err != nil {
				t.Fatalf("apply load option: %v", err)
			}
		}
		return awssdk.Config{Region: loadOptions.Region}, nil
	}

	client, err := newDynamoDB(
		ctx,
		appconfig.AWSConfig{Region: "us-west-2"},
		appconfig.DynamoDBConfig{EndpointURL: "http://localhost:8000"},
		loader,
	)
	if err != nil {
		t.Fatalf("new DynamoDB client: %v", err)
	}

	options := client.Options()
	if options.Region != "us-west-2" {
		t.Fatalf("unexpected region: %q", options.Region)
	}
	if options.BaseEndpoint == nil || *options.BaseEndpoint != "http://localhost:8000" {
		t.Fatalf("unexpected base endpoint: %v", options.BaseEndpoint)
	}
}

func TestNewDynamoDBLeavesAWSDefaultEndpointUnset(t *testing.T) {
	loader := func(_ context.Context, _ ...func(*sdkconfig.LoadOptions) error) (awssdk.Config, error) {
		return awssdk.Config{Region: "us-east-1"}, nil
	}

	client, err := newDynamoDB(
		context.Background(),
		appconfig.AWSConfig{Region: "us-east-1"},
		appconfig.DynamoDBConfig{},
		loader,
	)
	if err != nil {
		t.Fatalf("new DynamoDB client: %v", err)
	}
	if endpoint := client.Options().BaseEndpoint; endpoint != nil {
		t.Fatalf("expected AWS endpoint resolution, got %q", *endpoint)
	}
}

func TestNewDynamoDBWrapsConfigurationErrors(t *testing.T) {
	wantErr := errors.New("shared configuration is invalid")
	loader := func(_ context.Context, _ ...func(*sdkconfig.LoadOptions) error) (awssdk.Config, error) {
		return awssdk.Config{}, wantErr
	}

	client, err := newDynamoDB(
		context.Background(),
		appconfig.AWSConfig{Region: "us-east-1"},
		appconfig.DynamoDBConfig{},
		loader,
	)
	if client != nil {
		t.Fatal("expected no client when AWS configuration fails")
	}
	if !errors.Is(err, wantErr) || !strings.Contains(err.Error(), "load AWS SDK configuration") {
		t.Fatalf("unexpected error: %v", err)
	}
}
