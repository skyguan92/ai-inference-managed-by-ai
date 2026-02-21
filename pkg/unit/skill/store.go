package skill

import (
	"bytes"
	"context"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

// SkillStore is the storage interface for skills.
type SkillStore interface {
	Add(ctx context.Context, skill *Skill) error
	Get(ctx context.Context, id string) (*Skill, error)
	List(ctx context.Context, filter SkillFilter) ([]Skill, int, error)
	Remove(ctx context.Context, id string) error
	Update(ctx context.Context, skill *Skill) error
	Search(ctx context.Context, query string, category string) ([]Skill, error)
}

// MemoryStore is an in-memory implementation of SkillStore.
type MemoryStore struct {
	skills map[string]*Skill
	mu     sync.RWMutex
}

// NewMemoryStore creates a new MemoryStore.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		skills: make(map[string]*Skill),
	}
}

func (s *MemoryStore) Add(ctx context.Context, skill *Skill) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.skills[skill.ID]; exists {
		return ErrSkillAlreadyExists
	}

	s.skills[skill.ID] = skill
	return nil
}

func (s *MemoryStore) Get(ctx context.Context, id string) (*Skill, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	skill, exists := s.skills[id]
	if !exists {
		return nil, ErrSkillNotFound
	}
	return skill, nil
}

func (s *MemoryStore) List(ctx context.Context, filter SkillFilter) ([]Skill, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []Skill
	for _, sk := range s.skills {
		if filter.Category != "" && sk.Category != filter.Category {
			continue
		}
		if filter.Source != "" && sk.Source != filter.Source {
			continue
		}
		if filter.EnabledOnly && !sk.Enabled {
			continue
		}
		result = append(result, *sk)
	}

	total := len(result)

	offset := filter.Offset
	if offset > len(result) {
		offset = len(result)
	}

	end := len(result)
	if filter.Limit > 0 {
		end = offset + filter.Limit
		if end > len(result) {
			end = len(result)
		}
	}

	return result[offset:end], total, nil
}

func (s *MemoryStore) Remove(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	sk, exists := s.skills[id]
	if !exists {
		return ErrSkillNotFound
	}
	if sk.Source == SourceBuiltin {
		return ErrBuiltinSkillImmutable
	}

	delete(s.skills, id)
	return nil
}

func (s *MemoryStore) Update(ctx context.Context, skill *Skill) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.skills[skill.ID]; !exists {
		return ErrSkillNotFound
	}

	s.skills[skill.ID] = skill
	return nil
}

func (s *MemoryStore) Search(ctx context.Context, query string, category string) ([]Skill, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	q := strings.ToLower(query)
	var result []Skill
	for _, sk := range s.skills {
		if !sk.Enabled {
			continue
		}
		if category != "" && sk.Category != category {
			continue
		}
		if strings.Contains(strings.ToLower(sk.Name), q) ||
			strings.Contains(strings.ToLower(sk.Description), q) ||
			strings.Contains(strings.ToLower(sk.Content), q) {
			result = append(result, *sk)
		}
	}
	return result, nil
}

// skillFrontMatter is used to parse YAML front-matter from a skill file.
type skillFrontMatter struct {
	ID          string       `yaml:"id"`
	Name        string       `yaml:"name"`
	Category    string       `yaml:"category"`
	Description string       `yaml:"description"`
	Trigger     SkillTrigger `yaml:"trigger"`
	Priority    int          `yaml:"priority"`
	Enabled     bool         `yaml:"enabled"`
	Source      string       `yaml:"source"`
}

// ParseSkillFile parses a Markdown file with YAML front-matter into a Skill.
// The format is:
//
//	---
//	id: skill-id
//	name: "Skill Name"
//	...
//	---
//
//	Markdown body...
func ParseSkillFile(content string) (*Skill, error) {
	if !strings.HasPrefix(content, "---") {
		return nil, ErrSkillInvalid
	}

	// Find end of front-matter
	rest := content[3:]
	endIdx := strings.Index(rest, "\n---")
	if endIdx < 0 {
		return nil, ErrSkillInvalid
	}

	frontMatterStr := rest[:endIdx]
	body := strings.TrimPrefix(rest[endIdx+4:], "\n")

	var fm skillFrontMatter
	dec := yaml.NewDecoder(bytes.NewBufferString(frontMatterStr))
	if err := dec.Decode(&fm); err != nil {
		return nil, ErrSkillInvalid
	}

	if fm.ID == "" || fm.Name == "" {
		return nil, ErrSkillInvalid
	}

	return &Skill{
		ID:          fm.ID,
		Name:        fm.Name,
		Category:    fm.Category,
		Description: fm.Description,
		Trigger:     fm.Trigger,
		Content:     body,
		Priority:    fm.Priority,
		Enabled:     fm.Enabled,
		Source:      fm.Source,
	}, nil
}
