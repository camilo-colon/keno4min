package main

import (
	"slices"
	"strings"

	"github.com/cronos/keno4min/internal/common"
)

type QueueConfig struct {
	Name         string
	RulePatterns []string
}

type RuleConfig struct {
	Name    string
	Pattern string
	Prefix  bool
}

type TargetConfig struct {
	RuleName   string
	QueueNames []string
}

func extractQueueConfigs(subscribers []common.DomainEventSubscriber) []QueueConfig {
	queues := make([]QueueConfig, 0, len(subscribers))
	for _, subscriber := range subscribers {
		prototypes := subscriber.SubscribeTo()
		patterns := make([]string, 0, len(prototypes))
		for _, prototype := range prototypes {
			patterns = append(patterns, prototype.Type())
		}
		slices.Sort(patterns)
		queues = append(queues, QueueConfig{
			Name:         strings.ReplaceAll(subscriber.Name(), ".", "-"),
			RulePatterns: slices.Compact(patterns),
		})
	}
	slices.SortFunc(queues, func(a, b QueueConfig) int { return strings.Compare(a.Name, b.Name) })
	return queues
}

func extractRulesFromQueues(queues []QueueConfig) []RuleConfig {
	var rules []RuleConfig
	seen := make(map[string]bool)
	for _, queue := range queues {
		for _, pattern := range queue.RulePatterns {
			if seen[pattern] {
				continue
			}
			seen[pattern] = true
			rules = append(rules, RuleConfig{
				Name:    formatRuleName(pattern),
				Pattern: strings.TrimSuffix(pattern, "*"),
				Prefix:  strings.HasSuffix(pattern, "*"),
			})
		}
	}
	slices.SortFunc(rules, func(a, b RuleConfig) int { return strings.Compare(a.Name, b.Name) })
	return rules
}

func extractTargetsFromQueues(queues []QueueConfig) []TargetConfig {
	queuesByRule := make(map[string][]string)
	for _, queue := range queues {
		for _, pattern := range queue.RulePatterns {
			name := formatRuleName(pattern)
			queuesByRule[name] = append(queuesByRule[name], queue.Name)
		}
	}
	var targets []TargetConfig
	for rule, names := range queuesByRule {
		slices.Sort(names)
		targets = append(targets, TargetConfig{RuleName: rule, QueueNames: slices.Compact(names)})
	}
	slices.SortFunc(targets, func(a, b TargetConfig) int { return strings.Compare(a.RuleName, b.RuleName) })
	return targets
}

func formatRuleName(pattern string) string {
	return "rule-" + strings.NewReplacer(".", "-", "*", "all").Replace(pattern)
}
