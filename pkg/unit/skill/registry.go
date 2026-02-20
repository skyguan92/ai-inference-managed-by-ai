package skill

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

// SkillRegistry provides skill matching and formatting for agent system prompts.
type SkillRegistry struct {
	store SkillStore
}

// NewSkillRegistry creates a new SkillRegistry backed by a SkillStore.
func NewSkillRegistry(store SkillStore) *SkillRegistry {
	return &SkillRegistry{store: store}
}

// MatchSkills returns enabled skills whose keywords match the user message OR
// whose tool_names overlap with the provided active tool names.
// Always-on skills are NOT included here; use GetAlwaysOnSkills() for those.
func (r *SkillRegistry) MatchSkills(ctx context.Context, message string, toolNames []string) ([]Skill, error) {
	skills, _, err := r.store.List(ctx, SkillFilter{EnabledOnly: true})
	if err != nil {
		return nil, fmt.Errorf("list skills: %w", err)
	}

	msgLower := strings.ToLower(message)
	toolSet := make(map[string]struct{}, len(toolNames))
	for _, t := range toolNames {
		toolSet[t] = struct{}{}
	}

	var matched []Skill
	for _, sk := range skills {
		if sk.Trigger.AlwaysOn {
			continue // handled separately
		}

		if keywordsMatch(sk.Trigger.Keywords, msgLower) || toolNamesMatch(sk.Trigger.ToolNames, toolSet) {
			matched = append(matched, sk)
		}
	}

	// Sort by priority descending
	sort.Slice(matched, func(i, j int) bool {
		return matched[i].Priority > matched[j].Priority
	})

	return matched, nil
}

// GetAlwaysOnSkills returns all enabled skills with AlwaysOn=true, sorted by priority.
func (r *SkillRegistry) GetAlwaysOnSkills(ctx context.Context) ([]Skill, error) {
	skills, _, err := r.store.List(ctx, SkillFilter{EnabledOnly: true})
	if err != nil {
		return nil, fmt.Errorf("list skills: %w", err)
	}

	var alwaysOn []Skill
	for _, sk := range skills {
		if sk.Trigger.AlwaysOn {
			alwaysOn = append(alwaysOn, sk)
		}
	}

	sort.Slice(alwaysOn, func(i, j int) bool {
		return alwaysOn[i].Priority > alwaysOn[j].Priority
	})

	return alwaysOn, nil
}

// FormatForSystemPrompt formats a list of skills into a section suitable for
// inclusion in an agent's system prompt.
func (r *SkillRegistry) FormatForSystemPrompt(skills []Skill) string {
	if len(skills) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("## Active Skills\n\n")
	sb.WriteString("The following skills provide guidance for this conversation:\n\n")

	for _, sk := range skills {
		sb.WriteString(fmt.Sprintf("### %s (%s)\n\n", sk.Name, sk.Category))
		sb.WriteString(sk.Content)
		sb.WriteString("\n\n")
	}

	return sb.String()
}

func keywordsMatch(keywords []string, msgLower string) bool {
	for _, kw := range keywords {
		if strings.Contains(msgLower, strings.ToLower(kw)) {
			return true
		}
	}
	return false
}

func toolNamesMatch(triggerTools []string, activeTools map[string]struct{}) bool {
	for _, t := range triggerTools {
		if _, ok := activeTools[t]; ok {
			return true
		}
	}
	return false
}
