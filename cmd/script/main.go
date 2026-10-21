package main

import (
	"fmt"
	"log"
	"os"

	"github.com/cronos/keno4min/internal/common"
	"github.com/cronos/keno4min/internal/config"
	"github.com/cronos/keno4min/internal/nodes/nodesfx"
	"github.com/cronos/keno4min/internal/platform/awsfx"
	"github.com/cronos/keno4min/internal/platform/awsfx/dynamodbfx"
	"github.com/cronos/keno4min/internal/users/usersfx"
	"go.uber.org/fx"
)

type Script struct {
	subscribers []common.DomainEventSubscriber
}

func newScript(subscribers []common.DomainEventSubscriber) *Script {
	return &Script{subscribers: subscribers}
}

func (s Script) run() error {
	const directory = "infra/eventbridge/generated"

	if err := os.RemoveAll(directory); err != nil {
		return fmt.Errorf("limpiar Terraform generado: %w", err)
	}
	return s.generateTerraform(directory)
}

func (s Script) generateTerraform(directory string) error {
	queues := extractQueueConfigs(s.subscribers)

	if err := writeTerraformFile(directory, "variables", "variables.tf.tmpl", nil); err != nil {
		return err
	}
	if err := writeTerraformFile(directory, "versions", "versions.tf.tmpl", nil); err != nil {
		return err
	}
	for _, queue := range queues {
		if err := writeTerraformFile(directory, "sqs_queue_"+queue.Name, "sqs_queue.tf.tmpl", queue); err != nil {
			return err
		}
	}
	for _, rule := range extractRulesFromQueues(queues) {
		if err := writeTerraformFile(directory, "event_rule_"+rule.Name, "event_rule.tf.tmpl", rule); err != nil {
			return err
		}
	}
	for _, target := range extractTargetsFromQueues(queues) {
		if err := writeTerraformFile(directory, "event_target_"+target.RuleName, "event_target.tf.tmpl", target); err != nil {
			return err
		}
	}
	return nil
}

func main() {
	app := fx.New(
		fx.Provide(config.Load),
		awsfx.Module,
		dynamodbfx.Module,
		nodesfx.WorkerModule,
		usersfx.WorkerModule,
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
