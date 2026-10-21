package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/cronos/keno4min/internal/common"
)

type testSubscriber struct {
	name     string
	patterns []string
}

type testEvent struct {
	common.BaseEvent
	pattern string
}

func (e testEvent) Type() string { return e.pattern }

func (s testSubscriber) Name() string { return s.name }
func (s testSubscriber) SubscribeTo() []common.DomainEvent {
	events := make([]common.DomainEvent, 0, len(s.patterns))
	for _, pattern := range s.patterns {
		events = append(events, testEvent{pattern: pattern})
	}
	return events
}
func (s testSubscriber) On(context.Context, common.DomainEvent) error {
	panic("generation must not invoke event handlers")
}

func TestGenerateTerraform(t *testing.T) {
	subscribers := []common.DomainEventSubscriber{
		testSubscriber{"audit", []string{"core.*", "core.created", "core.created"}},
		testSubscriber{"nodes", []string{"core.created"}},
	}
	queues := extractQueueConfigs(subscribers)
	if got := extractRulesFromQueues(queues); !reflect.DeepEqual(got, []RuleConfig{
		{Name: "rule-core-all", Pattern: "core.", Prefix: true},
		{Name: "rule-core-created", Pattern: "core.created"},
	}) {
		t.Fatalf("unexpected rules: %#v", got)
	}
	if got := extractTargetsFromQueues(queues); !reflect.DeepEqual(got, []TargetConfig{
		{RuleName: "rule-core-all", QueueNames: []string{"audit"}},
		{RuleName: "rule-core-created", QueueNames: []string{"audit", "nodes"}},
	}) {
		t.Fatalf("unexpected targets: %#v", got)
	}
	directory := t.TempDir()
	if err := newScript(subscribers).generateTerraform(directory); err != nil {
		t.Fatal(err)
	}
	checks := map[string]string{
		"variables.tf":                      `variable "event_bus_name"`,
		"versions.tf":                       `source  = "hashicorp/aws"`,
		"sqs_queue_audit.tf":                `deadLetterTargetArn = aws_sqs_queue.audit-dl.arn`,
		"sqs_queue_nodes.tf":                `aws_cloudwatch_event_rule.rule-core-created.arn`,
		"event_rule_rule-core-all.tf":       `"detail-type" = [{ prefix = "core." }]`,
		"event_rule_rule-core-created.tf":   `"detail-type" = ["core.created"]`,
		"event_target_rule-core-all.tf":     `arn            = aws_sqs_queue.audit.arn`,
		"event_target_rule-core-created.tf": `arn            = aws_sqs_queue.nodes.arn`,
	}
	entries, err := os.ReadDir(directory)
	if err != nil || len(entries) != len(checks) {
		t.Fatalf("unexpected generated files: %v, %v", entries, err)
	}
	for name, expected := range checks {
		content, err := os.ReadFile(filepath.Join(directory, name))
		if err != nil || !strings.Contains(string(content), expected) {
			t.Errorf("%s: expected %s; error: %v", name, expected, err)
		}
	}
	if terraform, err := exec.LookPath("terraform"); err == nil {
		if output, err := exec.Command(terraform, "fmt", "-check", "-diff", directory).CombinedOutput(); err != nil {
			t.Fatalf("terraform fmt: %v\n%s", err, output)
		}
	}
}

func TestRunReplacesGeneratedSubscriptionsAndPreservesProject(t *testing.T) {
	t.Chdir(t.TempDir())
	root := filepath.Join("infra", "eventbridge")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	preserved := []string{"providers.tf", "terraform.tfvars", "terraform.tfstate", ".terraform.lock.hcl"}
	for _, name := range preserved {
		if err := os.WriteFile(filepath.Join(root, name), []byte("keep"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	script := newScript([]common.DomainEventSubscriber{
		testSubscriber{"audit", []string{"core.*", "core.created"}},
		testSubscriber{"nodes", []string{"core.created"}},
	})
	if err := script.run(); err != nil {
		t.Fatal(err)
	}
	script.subscribers = script.subscribers[1:]
	if err := script.run(); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"sqs_queue_audit.tf", "event_rule_rule-core-all.tf", "event_target_rule-core-all.tf"} {
		if _, err := os.Stat(filepath.Join(root, "generated", name)); !os.IsNotExist(err) {
			t.Errorf("obsolete file %s still exists: %v", name, err)
		}
	}
	for _, name := range []string{"sqs_queue_nodes.tf", "event_rule_rule-core-created.tf", "event_target_rule-core-created.tf"} {
		content, err := os.ReadFile(filepath.Join(root, "generated", name))
		if err != nil || strings.Contains(string(content), "audit") {
			t.Errorf("remaining subscription %s is incorrect: %s, %v", name, content, err)
		}
	}
	// Removing the last subscriber must also remove all resources.
	script.subscribers = nil
	if err := script.run(); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(filepath.Join(root, "generated"))
	if err != nil || len(entries) != 2 {
		t.Fatalf("empty subscriptions should leave only module declarations: %v, %v", entries, err)
	}
	for _, name := range preserved {
		content, err := os.ReadFile(filepath.Join(root, name))
		if err != nil || string(content) != "keep" {
			t.Errorf("project file %s changed: %s, %v", name, content, err)
		}
	}
}
