package service

import "time"

type BaseEvent struct {
	eventType string
	domain    string
	payload   any
}

func (e *BaseEvent) Type() string          { return e.eventType }
func (e *BaseEvent) Domain() string        { return e.domain }
func (e *BaseEvent) Payload() any          { return e.payload }
func (e *BaseEvent) Timestamp() time.Time  { return time.Now() }
func (e *BaseEvent) CorrelationID() string { return "" }

func getString(m map[string]any, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func getInt(m map[string]any, key string) int {
	if v, ok := m[key]; ok {
		switch val := v.(type) {
		case int:
			return val
		case int32:
			return int(val)
		case int64:
			return int(val)
		case float64:
			return int(val)
		}
	}
	return 0
}

func getInt64(m map[string]any, key string) int64 {
	if v, ok := m[key]; ok {
		switch val := v.(type) {
		case int:
			return int64(val)
		case int32:
			return int64(val)
		case int64:
			return val
		case float64:
			return int64(val)
		}
	}
	return 0
}

func getUint64(m map[string]any, key string) uint64 {
	if v, ok := m[key]; ok {
		switch val := v.(type) {
		case uint64:
			return val
		case int64:
			return uint64(val)
		case float64:
			return uint64(val)
		case int:
			return uint64(val)
		}
	}
	return 0
}

func getFloat64(m map[string]any, key string) float64 {
	if v, ok := m[key]; ok {
		switch val := v.(type) {
		case float64:
			return val
		case float32:
			return float64(val)
		case int:
			return float64(val)
		case int64:
			return float64(val)
		}
	}
	return 0
}

func getBool(m map[string]any, key string) bool {
	if v, ok := m[key].(bool); ok {
		return v
	}
	return false
}

func getStringSlice(m map[string]any, key string) []string {
	if v, ok := m[key]; ok {
		if slice, ok := v.([]string); ok {
			return slice
		}
		if slice, ok := v.([]any); ok {
			result := make([]string, 0, len(slice))
			for _, item := range slice {
				if s, ok := item.(string); ok {
					result = append(result, s)
				}
			}
			return result
		}
	}
	return nil
}
