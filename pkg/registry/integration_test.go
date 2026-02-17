package registry

import (
	"testing"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegistryIntegration(t *testing.T) {
	registry := unit.NewRegistry()
	err := RegisterAll(registry)
	require.NoError(t, err, "RegisterAll should not return an error")

	testCases := []struct {
		name     string
		unitName string
		unitType string
	}{
		{"model.create command", "model.create", "command"},
		{"model.delete command", "model.delete", "command"},
		{"model.pull command", "model.pull", "command"},
		{"model.import command", "model.import", "command"},
		{"model.verify command", "model.verify", "command"},
		{"model.get query", "model.get", "query"},
		{"model.list query", "model.list", "query"},
		{"model.search query", "model.search", "query"},
		{"model.estimate_resources query", "model.estimate_resources", "query"},

		{"device.detect command", "device.detect", "command"},
		{"device.set_power_limit command", "device.set_power_limit", "command"},
		{"device.info query", "device.info", "query"},
		{"device.metrics query", "device.metrics", "query"},
		{"device.health query", "device.health", "query"},

		{"engine.start command", "engine.start", "command"},
		{"engine.stop command", "engine.stop", "command"},
		{"engine.restart command", "engine.restart", "command"},
		{"engine.install command", "engine.install", "command"},
		{"engine.get query", "engine.get", "query"},
		{"engine.list query", "engine.list", "query"},
		{"engine.features query", "engine.features", "query"},

		{"inference.chat command", "inference.chat", "command"},
		{"inference.complete command", "inference.complete", "command"},
		{"inference.embed command", "inference.embed", "command"},
		{"inference.transcribe command", "inference.transcribe", "command"},
		{"inference.synthesize command", "inference.synthesize", "command"},
		{"inference.generate_image command", "inference.generate_image", "command"},
		{"inference.generate_video command", "inference.generate_video", "command"},
		{"inference.rerank command", "inference.rerank", "command"},
		{"inference.detect command", "inference.detect", "command"},
		{"inference.models query", "inference.models", "query"},
		{"inference.voices query", "inference.voices", "query"},

		{"resource.allocate command", "resource.allocate", "command"},
		{"resource.release command", "resource.release", "command"},
		{"resource.update_slot command", "resource.update_slot", "command"},
		{"resource.status query", "resource.status", "query"},
		{"resource.budget query", "resource.budget", "query"},
		{"resource.allocations query", "resource.allocations", "query"},

		{"service.create command", "service.create", "command"},
		{"service.delete command", "service.delete", "command"},
		{"service.scale command", "service.scale", "command"},
		{"service.start command", "service.start", "command"},
		{"service.stop command", "service.stop", "command"},
		{"service.get query", "service.get", "query"},
		{"service.list query", "service.list", "query"},

		{"app.install command", "app.install", "command"},
		{"app.uninstall command", "app.uninstall", "command"},
		{"app.start command", "app.start", "command"},
		{"app.stop command", "app.stop", "command"},
		{"app.get query", "app.get", "query"},
		{"app.list query", "app.list", "query"},
		{"app.logs query", "app.logs", "query"},

		{"pipeline.create command", "pipeline.create", "command"},
		{"pipeline.delete command", "pipeline.delete", "command"},
		{"pipeline.run command", "pipeline.run", "command"},
		{"pipeline.cancel command", "pipeline.cancel", "command"},
		{"pipeline.get query", "pipeline.get", "query"},
		{"pipeline.list query", "pipeline.list", "query"},
		{"pipeline.status query", "pipeline.status", "query"},
		{"pipeline.validate query", "pipeline.validate", "query"},

		{"alert.create_rule command", "alert.create_rule", "command"},
		{"alert.update_rule command", "alert.update_rule", "command"},
		{"alert.delete_rule command", "alert.delete_rule", "command"},
		{"alert.acknowledge command", "alert.acknowledge", "command"},
		{"alert.resolve command", "alert.resolve", "command"},
		{"alert.list_rules query", "alert.list_rules", "query"},
		{"alert.history query", "alert.history", "query"},
		{"alert.active query", "alert.active", "query"},

		{"remote.enable command", "remote.enable", "command"},
		{"remote.disable command", "remote.disable", "command"},
		{"remote.exec command", "remote.exec", "command"},
		{"remote.status query", "remote.status", "query"},
		{"remote.audit query", "remote.audit", "query"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.unitType == "command" {
				cmd := registry.GetCommand(tc.unitName)
				assert.NotNil(t, cmd, "command %s should be registered", tc.unitName)
				if cmd != nil {
					assert.Equal(t, tc.unitName, cmd.Name())
					assert.NotEmpty(t, cmd.Domain())
					assert.NotEmpty(t, cmd.Description())
				}
			} else {
				q := registry.GetQuery(tc.unitName)
				assert.NotNil(t, q, "query %s should be registered", tc.unitName)
				if q != nil {
					assert.Equal(t, tc.unitName, q.Name())
					assert.NotEmpty(t, q.Domain())
					assert.NotEmpty(t, q.Description())
				}
			}
		})
	}
}

func TestRegistryIntegrationCounts(t *testing.T) {
	registry := unit.NewRegistry()
	err := RegisterAll(registry)
	require.NoError(t, err)

	assert.Greater(t, registry.CommandCount(), 0, "should have at least one command registered")
	assert.Greater(t, registry.QueryCount(), 0, "should have at least one query registered")

	t.Logf("Registered %d commands, %d queries, %d resources",
		registry.CommandCount(), registry.QueryCount(), registry.ResourceCount())
}

func TestRegistryIntegrationAllDomains(t *testing.T) {
	registry := unit.NewRegistry()
	err := RegisterAll(registry)
	require.NoError(t, err)

	expectedDomains := []string{
		"model",
		"device",
		"engine",
		"inference",
		"resource",
		"service",
		"app",
		"pipeline",
		"alert",
		"remote",
	}

	registeredDomains := make(map[string]bool)
	for _, cmd := range registry.ListCommands() {
		registeredDomains[cmd.Domain()] = true
	}
	for _, q := range registry.ListQueries() {
		registeredDomains[q.Domain()] = true
	}

	for _, domain := range expectedDomains {
		assert.True(t, registeredDomains[domain], "domain %s should have at least one unit registered", domain)
	}
}

func TestRegistryIntegrationSchemaValid(t *testing.T) {
	registry := unit.NewRegistry()
	err := RegisterAll(registry)
	require.NoError(t, err)

	for _, cmd := range registry.ListCommands() {
		t.Run("command/"+cmd.Name(), func(t *testing.T) {
			inputSchema := cmd.InputSchema()
			assert.NotEmpty(t, inputSchema.Type, "input schema should have a type")

			outputSchema := cmd.OutputSchema()
			assert.NotEmpty(t, outputSchema.Type, "output schema should have a type")
		})
	}

	for _, q := range registry.ListQueries() {
		t.Run("query/"+q.Name(), func(t *testing.T) {
			inputSchema := q.InputSchema()
			assert.NotEmpty(t, inputSchema.Type, "input schema should have a type")

			outputSchema := q.OutputSchema()
			assert.NotEmpty(t, outputSchema.Type, "output schema should have a type")
		})
	}
}

func TestRegistryIntegrationDuplicateRegistration(t *testing.T) {
	registry := unit.NewRegistry()
	err := RegisterAll(registry)
	require.NoError(t, err)

	initialCmdCount := registry.CommandCount()
	initialQueryCount := registry.QueryCount()

	err = RegisterAll(registry)
	assert.Error(t, err, "second RegisterAll should return error for duplicate registration")

	assert.Equal(t, initialCmdCount, registry.CommandCount(), "command count should not change after failed re-registration")
	assert.Equal(t, initialQueryCount, registry.QueryCount(), "query count should not change after failed re-registration")
}
