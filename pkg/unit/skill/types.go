package skill

// Skill represents a knowledge unit that can be loaded into an Agent's context.
type Skill struct {
	ID          string       `json:"id" yaml:"id"`
	Name        string       `json:"name" yaml:"name"`
	Category    string       `json:"category" yaml:"category"`       // setup, troubleshoot, optimize, manage
	Description string       `json:"description" yaml:"description"`
	Trigger     SkillTrigger `json:"trigger" yaml:"trigger"`
	Content     string       `json:"content" yaml:"content"`         // Markdown body
	Priority    int          `json:"priority" yaml:"priority"`       // higher = more important
	Enabled     bool         `json:"enabled" yaml:"enabled"`
	Source      string       `json:"source" yaml:"source"`           // "builtin", "user", "community"
}

// SkillTrigger defines activation conditions for a Skill.
type SkillTrigger struct {
	Keywords  []string `json:"keywords,omitempty" yaml:"keywords,omitempty"`     // activate when user message contains these words
	ToolNames []string `json:"tool_names,omitempty" yaml:"tool_names,omitempty"` // activate when these tools are involved
	AlwaysOn  bool     `json:"always_on,omitempty" yaml:"always_on,omitempty"`   // always load into system prompt
}

// SkillFilter holds filtering parameters for listing skills.
type SkillFilter struct {
	Category    string
	Source      string
	EnabledOnly bool
	Limit       int
	Offset      int
}

const (
	CategorySetup       = "setup"
	CategoryTroubleshoot = "troubleshoot"
	CategoryOptimize    = "optimize"
	CategoryManage      = "manage"

	SourceBuiltin   = "builtin"
	SourceUser      = "user"
	SourceCommunity = "community"
)
