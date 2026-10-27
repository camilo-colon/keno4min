package main

import (
	"testing"

	"go.uber.org/fx"
)

func TestDependencyGraph(t *testing.T) {
	err := fx.ValidateApp(
		fx.NopLogger,
		apiModule(),
	)
	if err != nil {
		t.Fatalf("invalid API dependency graph: %v", err)
	}
}
