package workerfx

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/cronos/keno4min/internal/common"
	"github.com/cronos/keno4min/internal/config"
)

type Handler func(context.Context, common.DomainEvent) error

type ConsumerSQS struct {
	queueName    string
	handler      Handler
	deserializer *DomainEventJSONDeserializer
	config       *config.Config
	sqsClient    *sqs.Client
	queueURL     string
}

func NewConsumerSQS(queueName string, handler Handler, deserializer *DomainEventJSONDeserializer, sqsClient *sqs.Client, config *config.Config) *ConsumerSQS {
	return &ConsumerSQS{
		queueName:    queueName,
		handler:      handler,
		deserializer: deserializer,
		sqsClient:    sqsClient,
		config:       config,
	}
}

// Resolve looks up the queue URL. It must succeed before Run is called.
func (c *ConsumerSQS) Resolve(ctx context.Context) error {
	queueURL, err := c.GetQueueUrl(ctx)
	if err != nil {
		return err
	}
	c.queueURL = queueURL
	return nil
}

// Run polls the queue resolved by Resolve until ctx is cancelled or receiving fails.
func (c *ConsumerSQS) Run(ctx context.Context) error {
	queueURL := c.queueURL
	if queueURL == "" {
		return fmt.Errorf("queue %q was not resolved", c.queueName)
	}
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		messages, err := c.receiveMessages(ctx, queueURL)
		if err != nil {
			return err
		}
		for _, message := range messages {
			if err := ctx.Err(); err != nil {
				return err
			}
			if err := c.processMessage(ctx, queueURL, message); err != nil {
				// Leave failed messages for SQS redelivery and the configured DLQ.
				log.Printf("Couldn't process message in queue %s: %v", c.queueName, err)
			}
		}
	}
}

func (c ConsumerSQS) processMessage(ctx context.Context, queueURL string, message types.Message) error {
	if message.Body == nil || message.ReceiptHandle == nil || *message.ReceiptHandle == "" {
		return fmt.Errorf("message is missing body or receipt handle")
	}
	event, err := c.deserializer.Deserialize([]byte(*message.Body))
	if err != nil {
		return err
	}
	if err := c.handler(ctx, event); err != nil {
		return fmt.Errorf("handle event %q: %w", event.Type(), err)
	}
	if _, err := c.sqsClient.DeleteMessage(ctx, &sqs.DeleteMessageInput{
		QueueUrl:      &queueURL,
		ReceiptHandle: message.ReceiptHandle,
	}); err != nil {
		return fmt.Errorf("delete processed message: %w", err)
	}
	return nil
}

func (c ConsumerSQS) GetQueueName() string {
	queueName := strings.ReplaceAll(c.queueName, ".", "-")
	queueName = strings.ReplaceAll(queueName, "cronos-", fmt.Sprintf("cronos-%s-", c.config.Environment))
	return queueName
}

func (c ConsumerSQS) GetQueueUrl(ctx context.Context) (string, error) {
	queueName := c.GetQueueName()
	queueUrl, err := c.sqsClient.GetQueueUrl(ctx, &sqs.GetQueueUrlInput{
		QueueName: &queueName,
	})
	if err != nil {
		return "", fmt.Errorf("get queue %q: %w", queueName, err)
	}
	if queueUrl.QueueUrl == nil || *queueUrl.QueueUrl == "" {
		return "", fmt.Errorf("queue %q has no URL", queueName)
	}
	return *queueUrl.QueueUrl, nil
}

func (c ConsumerSQS) GetMessage(ctx context.Context) ([]types.Message, error) {
	queueUrl, err := c.GetQueueUrl(ctx)
	if err != nil {
		return nil, err
	}
	return c.receiveMessages(ctx, queueUrl)
}

func (c ConsumerSQS) receiveMessages(ctx context.Context, queueURL string) ([]types.Message, error) {
	result, err := c.sqsClient.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
		QueueUrl:            &queueURL,
		WaitTimeSeconds:     20,
		MaxNumberOfMessages: 10,
	})
	if err != nil {
		return nil, fmt.Errorf("receive messages from queue %q: %w", c.queueName, err)
	}
	return result.Messages, nil
}
