package service

import (
	"fmt"
	"strings"
)

// ServiceID encapsulates the parsed components of a service identifier.
// Format: "svc-{engineType}-{modelId}" where modelId may contain dashes.
type ServiceID struct {
	EngineType string
	ModelID    string
}

// ParseServiceID parses a service ID string into its components.
func ParseServiceID(id string) (ServiceID, error) {
	prefix := "svc-"
	if !strings.HasPrefix(id, prefix) {
		return ServiceID{}, fmt.Errorf("invalid service ID: %s (must start with 'svc-')", id)
	}
	rest := id[len(prefix):]
	idx := strings.Index(rest, "-")
	if idx == -1 {
		return ServiceID{}, fmt.Errorf("invalid service ID format: %s (expected svc-{engine}-{model})", id)
	}
	return ServiceID{
		EngineType: rest[:idx],
		ModelID:    rest[idx+1:],
	}, nil
}

func (s ServiceID) String() string {
	return fmt.Sprintf("svc-%s-%s", s.EngineType, s.ModelID)
}
