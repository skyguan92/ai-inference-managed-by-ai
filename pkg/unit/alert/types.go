package alert

import "time"

type AlertSeverity string

const (
	AlertSeverityInfo     AlertSeverity = "info"
	AlertSeverityWarning  AlertSeverity = "warning"
	AlertSeverityCritical AlertSeverity = "critical"
)

type AlertStatus string

const (
	AlertStatusFiring       AlertStatus = "firing"
	AlertStatusAcknowledged AlertStatus = "acknowledged"
	AlertStatusResolved     AlertStatus = "resolved"
)

type AlertRule struct {
	ID        string        `json:"id"`
	Name      string        `json:"name"`
	Condition string        `json:"condition"`
	Severity  AlertSeverity `json:"severity"`
	Channels  []string      `json:"channels,omitempty"`
	Cooldown  int           `json:"cooldown,omitempty"`
	Enabled   bool          `json:"enabled"`
	CreatedAt time.Time     `json:"created_at"`
	UpdatedAt time.Time     `json:"updated_at"`
}

type Alert struct {
	ID             string         `json:"id"`
	RuleID         string         `json:"rule_id"`
	RuleName       string         `json:"rule_name"`
	Severity       AlertSeverity  `json:"severity"`
	Status         AlertStatus    `json:"status"`
	Message        string         `json:"message"`
	Metrics        map[string]any `json:"metrics,omitempty"`
	TriggeredAt    time.Time      `json:"triggered_at"`
	AcknowledgedAt *time.Time     `json:"acknowledged_at,omitempty"`
	ResolvedAt     *time.Time     `json:"resolved_at,omitempty"`
}

type RuleFilter struct {
	EnabledOnly bool
}

type AlertFilter struct {
	RuleID   string
	Status   AlertStatus
	Severity AlertSeverity
	Limit    int
	Offset   int
}

type AlertHistoryFilter struct {
	RuleID   string
	Status   AlertStatus
	Severity AlertSeverity
	Limit    int
}
