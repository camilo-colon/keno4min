package workerfx

import (
	"context"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/cronos/keno4min/internal/common"
)

type Handler func(context.Context, common.DomainEvent) error

type ConsumerSQS struct {
	queueURL  string
	handler   Handler
	sqsClient *sqs.Client
}

func NewConsumerSQS(queueUrl string, handler Handler) *ConsumerSQS {
	return &ConsumerSQS{}
}

func (c ConsumerSQS) Run(ctx context.Context) error {
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		messages, err := c.GetMessage(ctx)
		if err != nil {
			return err
		}
		for _, message := range messages {
			attributes := message.Attributes
			fmt.Println(attributes)
		}
	}
}

func (c ConsumerSQS) GetMessage(ctx context.Context) ([]types.Message, error) {
	var messages []types.Message
	result, err := c.sqsClient.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{})
	if err != nil {
		log.Printf("Couldn't get messages from queue %v. Here's why: %v\n", c.queueURL, err)
	} else {
		messages = result.Messages
	}
	return messages, err
}
