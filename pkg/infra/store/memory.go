package store

import (
	"fmt"
	"strconv"
	"sync"
)

// MemoryStore 基于内存的存储实现
type MemoryStore struct {
	mu       sync.RWMutex
	data     map[string]map[string]any // table -> key -> value
	counters map[string]int            // 自增ID计数器
}

// NewMemoryStore 创建新的内存存储实例
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		data:     make(map[string]map[string]any),
		counters: make(map[string]int),
	}
}

// Create 在指定表中创建新记录
func (s *MemoryStore) Create(table string, key string, value any) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.data[table]; !exists {
		s.data[table] = make(map[string]any)
	}

	if _, exists := s.data[table][key]; exists {
		return fmt.Errorf("record with key %q already exists in table %q", key, table)
	}

	s.data[table][key] = value
	return nil
}

// Get 从指定表中获取记录
func (s *MemoryStore) Get(table string, key string) (any, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tableData, exists := s.data[table]
	if !exists {
		return nil, false
	}

	value, exists := tableData[key]
	return value, exists
}

// List 列出指定表中的所有记录
func (s *MemoryStore) List(table string) []any {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tableData, exists := s.data[table]
	if !exists {
		return []any{}
	}

	result := make([]any, 0, len(tableData))
	for _, value := range tableData {
		result = append(result, value)
	}

	return result
}

// Update 更新指定表中的记录
func (s *MemoryStore) Update(table string, key string, value any) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	tableData, exists := s.data[table]
	if !exists {
		return fmt.Errorf("table %q does not exist", table)
	}

	if _, exists := tableData[key]; !exists {
		return fmt.Errorf("record with key %q not found in table %q", key, table)
	}

	tableData[key] = value
	return nil
}

// Delete 从指定表中删除记录
func (s *MemoryStore) Delete(table string, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	tableData, exists := s.data[table]
	if !exists {
		return fmt.Errorf("table %q does not exist", table)
	}

	if _, exists := tableData[key]; !exists {
		return fmt.Errorf("record with key %q not found in table %q", key, table)
	}

	delete(tableData, key)
	return nil
}

// NextID 获取指定表的下一个自增ID
func (s *MemoryStore) NextID(table string) string {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.counters[table]++
	return strconv.Itoa(s.counters[table])
}

// Clear 清空所有数据
func (s *MemoryStore) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.data = make(map[string]map[string]any)
	s.counters = make(map[string]int)
}
