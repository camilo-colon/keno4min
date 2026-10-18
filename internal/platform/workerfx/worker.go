package workerfx

import (
	"context"
	"sync"

	"github.com/cronos/keno4min/internal/common"
	"go.uber.org/fx"
)

type Worker struct {
	subscribers []common.DomainEventSubscriber
}

func NewWorker(subscribers []common.DomainEventSubscriber) *Worker {
	return &Worker{
		subscribers: subscribers,
	}
}

func (w Worker) Start(ctx context.Context) {
	var wg sync.WaitGroup
	for _, subscriber := range w.subscribers {
		queueUrl := subscriber.Name()
		consumer := NewConsumerSQS(queueUrl, subscriber.On)
		wg.Go(func() {
			consumer.Run(ctx)
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
