package registry

import (
	"testing"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
)

func TestRegisterAll(t *testing.T) {
	registry := unit.NewRegistry()

	err := RegisterAll(registry)
	if err != nil {
		t.Fatalf("RegisterAll() error = %v", err)
	}

	// Verify commands were registered
	cmds := registry.ListCommands()
	if len(cmds) == 0 {
		t.Error("Expected commands to be registered, got 0")
	}

	// Verify queries were registered
	queries := registry.ListQueries()
	if len(queries) == 0 {
		t.Error("Expected queries to be registered, got 0")
	}
}

func TestRegisterAllWithDefaults(t *testing.T) {
	registry := unit.NewRegistry()

	err := RegisterAllWithDefaults(registry)
	if err != nil {
		t.Fatalf("RegisterAllWithDefaults() error = %v", err)
	}

	// Check specific domains are registered
	tests := []struct {
		name     string
		unitName string
		wantCmd  bool
		wantQry  bool
	}{
		{"model.create command", "model.create", true, false},
		{"model.list query", "model.list", false, true},
		{"device.detect command", "device.detect", true, false},
		{"device.info query", "device.info", false, true},
		{"engine.start command", "engine.start", true, false},
		{"engine.list query", "engine.list", false, true},
		{"inference.chat command", "inference.chat", true, false},
		{"inference.models query", "inference.models", false, true},
		{"resource.allocate command", "resource.allocate", true, false},
		{"resource.status query", "resource.status", false, true},
		{"service.create command", "service.create", true, false},
		{"service.list query", "service.list", false, true},
		{"app.install command", "app.install", true, false},
		{"app.list query", "app.list", false, true},
		{"pipeline.create command", "pipeline.create", true, false},
		{"pipeline.list query", "pipeline.list", false, true},
		{"alert.create_rule command", "alert.create_rule", true, false},
		{"alert.list_rules query", "alert.list_rules", false, true},
		{"remote.enable command", "remote.enable", true, false},
		{"remote.status query", "remote.status", false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.wantCmd {
				cmd := registry.GetCommand(tt.unitName)
				if cmd == nil {
					t.Errorf("Expected command %s to be registered", tt.unitName)
				}
			}
			if tt.wantQry {
				qry := registry.GetQuery(tt.unitName)
				if qry == nil {
					t.Errorf("Expected query %s to be registered", tt.unitName)
				}
			}
		})
	}
}

func TestRegisterAllWithStores(t *testing.T) {
	registry := unit.NewRegistry()

	// Test with explicit nil stores (should create defaults)
	err := RegisterAll(registry, WithStores(Stores{}))
	if err != nil {
		t.Fatalf("RegisterAll() with empty stores error = %v", err)
	}

	// Verify units still registered
	if registry.CommandCount() == 0 {
		t.Error("Expected commands to be registered with empty stores")
	}
}

func TestCommandCounts(t *testing.T) {
	registry := unit.NewRegistry()

	err := RegisterAll(registry)
	if err != nil {
		t.Fatalf("RegisterAll() error = %v", err)
	}

	// Model domain: 5 commands (create, delete, pull, import, verify)
	// Device domain: 2 commands (detect, set_power_limit)
	// Engine domain: 4 commands (start, stop, restart, install)
	// Inference domain: 9 commands (chat, complete, embed, transcribe, synthesize, generate_image, generate_video, rerank, detect)
	// Resource domain: 3 commands (allocate, release, update_slot)
	// Service domain: 5 commands (create, delete, scale, start, stop)
	// App domain: 4 commands (install, uninstall, start, stop)
	// Pipeline domain: 4 commands (create, delete, run, cancel)
	// Alert domain: 5 commands (create_rule, update_rule, delete_rule, acknowledge, resolve)
	// Remote domain: 3 commands (enable, disable, exec)
	// Total: 44 commands

	cmdCount := registry.CommandCount()
	if cmdCount < 40 {
		t.Errorf("Expected at least 40 commands, got %d", cmdCount)
	}

	// Verify query counts
	qryCount := registry.QueryCount()
	if qryCount < 20 {
		t.Errorf("Expected at least 20 queries, got %d", qryCount)
	}
}
