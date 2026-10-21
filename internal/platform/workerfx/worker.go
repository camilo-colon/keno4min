package workerfx

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/cronos/keno4min/internal/common"
	"github.com/cronos/keno4min/internal/config"
	"go.uber.org/fx"
)

type Worker struct {
	consumers []*ConsumerSQS
}

func NewWorker(subscribers []common.DomainEventSubscriber, sqsClient *sqs.Client, config *config.Config) (*Worker, error) {
	consumers := make([]*ConsumerSQS, 0, len(subscribers))
	for _, subscriber := range subscribers {
		deserializer, err := NewDomainEventJSONDeserializer(subscriber.SubscribeTo())
		if err != nil {
			return nil, fmt.Errorf("subscriber %q: %w", subscriber.Name(), err)
		}
		consumers = append(consumers, NewConsumerSQS(subscriber.Name(), subscriber.On, deserializer, sqsClient, config))
	}
	return &Worker{consumers: consumers}, nil
}

func (w Worker) Start(ctx context.Context) {
	var wg sync.WaitGroup
	for _, consumer := range w.consumers {
		wg.Go(func() {
			if err := consumer.Run(ctx); err != nil && ctx.Err() == nil {
				log.Printf("Consumer %s stopped: %v", consumer.queueName, err)
			}
		})
	}
	wg.Wait()
}

func (w Worker) Run(lc fx.Lifecycle) {
	ctxCancel, cancel := context.WithCancel(context.Background())
	finished := make(chan struct{})
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go func() {
				defer close(finished)
				w.Start(ctxCancel)
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			cancel()
			select {
			case <-finished:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		},
	})
}
