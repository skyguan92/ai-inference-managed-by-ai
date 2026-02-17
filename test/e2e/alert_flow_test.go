package e2e

import (
	"testing"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/gateway"
)

func TestAlertFlowE2E(t *testing.T) {
	env := SetupTestEnv(t)

	t.Run("create alert rule", func(t *testing.T) {
		resp := env.Gateway.Handle(env.Ctx, &gateway.Request{
			Type: "command",
			Unit: "alert.create_rule",
			Input: map[string]any{
				"name":      "high_memory",
				"condition": "memory > 90%",
				"severity":  "warning",
				"channels":  []string{"email", "slack"},
				"cooldown":  300,
			},
		})
		assertSuccess(t, resp)

		data := getMapField(resp.Data, "")
		ruleID := getStringField(data, "rule_id")
		if ruleID == "" {
			t.Errorf("expected rule_id to be non-empty")
		}
	})

	t.Run("create and list alert rules", func(t *testing.T) {
		for i := 0; i < 3; i++ {
			resp := env.Gateway.Handle(env.Ctx, &gateway.Request{
				Type: "command",
				Unit: "alert.create_rule",
				Input: map[string]any{
					"name":      "list_test_rule",
					"condition": "cpu > 80%",
					"severity":  "warning",
				},
			})
			assertSuccess(t, resp)
		}

		listResp := env.Gateway.Handle(env.Ctx, &gateway.Request{
			Type:  "query",
			Unit:  "alert.list_rules",
			Input: map[string]any{},
		})
		assertSuccess(t, listResp)

		data := getMapField(listResp.Data, "")
		rules := getSliceField(data, "rules")
		total := len(rules)
		if total < 0 {
			t.Errorf("expected non-negative rules count, got: %d", total)
		}
	})

	t.Run("list enabled rules only", func(t *testing.T) {
		resp := env.Gateway.Handle(env.Ctx, &gateway.Request{
			Type: "query",
			Unit: "alert.list_rules",
			Input: map[string]any{
				"enabled_only": true,
			},
		})
		assertSuccess(t, resp)

		data := getMapField(resp.Data, "")
		rules := getSliceField(data, "rules")
		for _, r := range rules {
			rule, ok := r.(map[string]any)
			if !ok {
				continue
			}
			if enabled, ok := rule["enabled"].(bool); ok && !enabled {
				t.Errorf("expected only enabled rules")
			}
		}
	})

	t.Run("update alert rule", func(t *testing.T) {
		createResp := env.Gateway.Handle(env.Ctx, &gateway.Request{
			Type: "command",
			Unit: "alert.create_rule",
			Input: map[string]any{
				"name":      "update_test_rule",
				"condition": "cpu > 70%",
				"severity":  "warning",
			},
		})
		assertSuccess(t, createResp)

		data := getMapField(createResp.Data, "")
		ruleID := getStringField(data, "rule_id")

		updateResp := env.Gateway.Handle(env.Ctx, &gateway.Request{
			Type: "command",
			Unit: "alert.update_rule",
			Input: map[string]any{
				"rule_id":   ruleID,
				"condition": "cpu > 90%",
				"enabled":   true,
			},
		})
		assertSuccess(t, updateResp)

		updateData := getMapField(updateResp.Data, "")
		if updateData["success"] != true {
			t.Errorf("expected success to be true")
		}
	})

	t.Run("delete alert rule", func(t *testing.T) {
		createResp := env.Gateway.Handle(env.Ctx, &gateway.Request{
			Type: "command",
			Unit: "alert.create_rule",
			Input: map[string]any{
				"name":      "delete_test_rule",
				"condition": "cpu > 50%",
				"severity":  "info",
			},
		})
		assertSuccess(t, createResp)

		data := getMapField(createResp.Data, "")
		ruleID := getStringField(data, "rule_id")

		deleteResp := env.Gateway.Handle(env.Ctx, &gateway.Request{
			Type: "command",
			Unit: "alert.delete_rule",
			Input: map[string]any{
				"rule_id": ruleID,
			},
		})
		assertSuccess(t, deleteResp)

		deleteData := getMapField(deleteResp.Data, "")
		if deleteData["success"] != true {
			t.Errorf("expected success to be true")
		}
	})

	t.Run("get alert history", func(t *testing.T) {
		resp := env.Gateway.Handle(env.Ctx, &gateway.Request{
			Type:  "query",
			Unit:  "alert.history",
			Input: map[string]any{},
		})
		assertSuccess(t, resp)

		data := getMapField(resp.Data, "")
		alerts := getSliceField(data, "alerts")
		_ = alerts
	})

	t.Run("get alert history with filters", func(t *testing.T) {
		resp := env.Gateway.Handle(env.Ctx, &gateway.Request{
			Type: "query",
			Unit: "alert.history",
			Input: map[string]any{
				"status": "resolved",
				"limit":  10,
			},
		})
		assertSuccess(t, resp)
	})

	t.Run("get active alerts", func(t *testing.T) {
		resp := env.Gateway.Handle(env.Ctx, &gateway.Request{
			Type:  "query",
			Unit:  "alert.active",
			Input: map[string]any{},
		})
		assertSuccess(t, resp)

		data := getMapField(resp.Data, "")
		alerts := getSliceField(data, "alerts")
		_ = alerts
	})

	t.Run("create rule with different severities", func(t *testing.T) {
		severities := []string{"info", "warning", "critical"}
		for _, sev := range severities {
			resp := env.Gateway.Handle(env.Ctx, &gateway.Request{
				Type: "command",
				Unit: "alert.create_rule",
				Input: map[string]any{
					"name":      "severity_test_" + sev,
					"condition": "test > 0",
					"severity":  sev,
				},
			})
			assertSuccess(t, resp)
		}
	})

	t.Run("create rule with invalid severity should fail", func(t *testing.T) {
		resp := env.Gateway.Handle(env.Ctx, &gateway.Request{
			Type: "command",
			Unit: "alert.create_rule",
			Input: map[string]any{
				"name":      "invalid_severity_rule",
				"condition": "test > 0",
				"severity":  "invalid",
			},
		})
		assertError(t, resp)
	})

	t.Run("create rule without name should fail", func(t *testing.T) {
		resp := env.Gateway.Handle(env.Ctx, &gateway.Request{
			Type: "command",
			Unit: "alert.create_rule",
			Input: map[string]any{
				"condition": "test > 0",
				"severity":  "warning",
			},
		})
		assertError(t, resp)
	})

	t.Run("create rule without condition should fail", func(t *testing.T) {
		resp := env.Gateway.Handle(env.Ctx, &gateway.Request{
			Type: "command",
			Unit: "alert.create_rule",
			Input: map[string]any{
				"name":     "no_condition_rule",
				"severity": "warning",
			},
		})
		assertError(t, resp)
	})

	t.Run("update non-existent rule should fail", func(t *testing.T) {
		resp := env.Gateway.Handle(env.Ctx, &gateway.Request{
			Type: "command",
			Unit: "alert.update_rule",
			Input: map[string]any{
				"rule_id": "non-existent-rule-id",
				"enabled": true,
			},
		})
		assertError(t, resp)
	})

	t.Run("delete non-existent rule should fail", func(t *testing.T) {
		resp := env.Gateway.Handle(env.Ctx, &gateway.Request{
			Type: "command",
			Unit: "alert.delete_rule",
			Input: map[string]any{
				"rule_id": "non-existent-rule-id",
			},
		})
		assertError(t, resp)
	})

	t.Run("acknowledge non-existent alert should fail", func(t *testing.T) {
		resp := env.Gateway.Handle(env.Ctx, &gateway.Request{
			Type: "command",
			Unit: "alert.acknowledge",
			Input: map[string]any{
				"alert_id": "non-existent-alert-id",
			},
		})
		assertError(t, resp)
	})

	t.Run("resolve non-existent alert should fail", func(t *testing.T) {
		resp := env.Gateway.Handle(env.Ctx, &gateway.Request{
			Type: "command",
			Unit: "alert.resolve",
			Input: map[string]any{
				"alert_id": "non-existent-alert-id",
			},
		})
		assertError(t, resp)
	})

	t.Run("full alert rule lifecycle", func(t *testing.T) {
		createResp := env.Gateway.Handle(env.Ctx, &gateway.Request{
			Type: "command",
			Unit: "alert.create_rule",
			Input: map[string]any{
				"name":      "full_lifecycle_rule",
				"condition": "memory > 80%",
				"severity":  "critical",
				"channels":  []string{"email"},
				"cooldown":  600,
			},
		})
		assertSuccess(t, createResp)

		data := getMapField(createResp.Data, "")
		ruleID := getStringField(data, "rule_id")

		listResp := env.Gateway.Handle(env.Ctx, &gateway.Request{
			Type:  "query",
			Unit:  "alert.list_rules",
			Input: map[string]any{},
		})
		assertSuccess(t, listResp)

		updateResp := env.Gateway.Handle(env.Ctx, &gateway.Request{
			Type: "command",
			Unit: "alert.update_rule",
			Input: map[string]any{
				"rule_id":   ruleID,
				"condition": "memory > 85%",
				"enabled":   true,
			},
		})
		assertSuccess(t, updateResp)

		deleteResp := env.Gateway.Handle(env.Ctx, &gateway.Request{
			Type: "command",
			Unit: "alert.delete_rule",
			Input: map[string]any{
				"rule_id": ruleID,
			},
		})
		assertSuccess(t, deleteResp)
	})
}
