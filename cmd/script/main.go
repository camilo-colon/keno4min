package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/cronos/keno4min/internal/common"
	"github.com/cronos/keno4min/internal/config"
	"github.com/cronos/keno4min/internal/nodes/nodesfx"
	"github.com/cronos/keno4min/internal/platform/awsfx"
	"github.com/cronos/keno4min/internal/platform/awsfx/dynamodbfx"
	"github.com/cronos/keno4min/internal/platform/awsfx/sqsfx"
	"go.uber.org/fx"
)

type Script struct {
	sqsClient   *sqs.Client
	subscribers []common.DomainEventSubscriber
}

func newScript(subscribers []common.DomainEventSubscriber, sqsClient *sqs.Client) *Script {
	return &Script{
		sqsClient:   sqsClient,
		subscribers: subscribers,
	}
}

type QueueConfig struct {
	name        string
	rulePattern []string
}

func extractQueueConfigs(subscribers []common.DomainEventSubscriber) []QueueConfig {
	queues := make([]QueueConfig, 0, len(subscribers))
	for _, subscriber := range subscribers {
		patterns := make([]string, 0, len(subscriber.SubscribeTo()))
		for _, item := range subscriber.SubscribeTo() {
			patterns = append(patterns, item)
		}
		queues = append(queues, QueueConfig{
			name:        strings.ReplaceAll(subscriber.Name(), ".", "-"),
			rulePattern: patterns,
		})
	}
	return queues
}

func (s Script) createQueueAndDeadLetter(queueConfig QueueConfig) error {
	ctx := context.Background()
	deadLetterName := fmt.Sprintf("%s-dl", queueConfig.name)
	deadLetterQueue, err := s.sqsClient.CreateQueue(ctx, &sqs.CreateQueueInput{
		QueueName: &deadLetterName,
	})
	if err != nil {
		return fmt.Errorf("crear DLQ %s: %w", deadLetterName, err)
	}
	attributes, err := s.sqsClient.GetQueueAttributes(ctx, &sqs.GetQueueAttributesInput{
		QueueUrl:       deadLetterQueue.QueueUrl,
		AttributeNames: []types.QueueAttributeName{types.QueueAttributeNameQueueArn},
	})
	if err != nil {
		return fmt.Errorf("obtener ARN de DLQ %s: %w", deadLetterName, err)
	}
	deadLetterArn := attributes.Attributes[string(types.QueueAttributeNameQueueArn)]
	if deadLetterArn == "" {
		return fmt.Errorf("SQS no devolvió el ARN de DLQ %s", deadLetterName)
	}
	_, err = s.sqsClient.CreateQueue(ctx, &sqs.CreateQueueInput{
		QueueName: &queueConfig.name,
		Attributes: map[string]string{
			"RedrivePolicy": fmt.Sprintf(`{
	  		  "deadLetterTargetArn": "%s",
              "maxReceiveCount": 3
			}`, deadLetterArn),
		},
	})
	if err != nil {
		return fmt.Errorf("crear cola %s: %w", queueConfig.name, err)
	}
	return nil
}

func (s Script) run() error {
	queuesConfig := extractQueueConfigs(s.subscribers)
	for _, config := range queuesConfig {
		if err := s.createQueueAndDeadLetter(config); err != nil {
			return err
		}
	}
	return nil
}

func main() {
	app := fx.New(
		fx.Provide(config.Load),
		awsfx.Module,
		sqsfx.Module,
		dynamodbfx.Module,
		nodesfx.WorkerModule,
		fx.Provide(
			fx.Annotate(
				newScript,
				fx.ParamTags(`group:"subscribers"`),
			),
		),
		fx.Invoke(func(script *Script) error {
			return script.run()
		}),
	)

	if err := app.Err(); err != nil {
		log.Fatal(err)
	}
}
