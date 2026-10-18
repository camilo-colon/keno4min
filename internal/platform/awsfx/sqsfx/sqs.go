package sqsfx

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

func NewSqsClient(awsCfg aws.Config) *sqs.Client {
	return sqs.NewFromConfig(awsCfg)
}
