package workflow

import (
	"context"
	"sync"
)

type WorkflowStore interface {
	SaveWorkflow(ctx context.Context, def *WorkflowDef) error
	GetWorkflow(ctx context.Context, name string) (*WorkflowDef, error)
	ListWorkflows(ctx context.Context) ([]*WorkflowDef, error)
	DeleteWorkflow(ctx context.Context, name string) error

	SaveExecution(ctx context.Context, result *ExecutionResult) error
	GetExecution(ctx context.Context, runID string) (*ExecutionResult, error)
	ListExecutions(ctx context.Context, workflowName string, limit int) ([]*ExecutionResult, error)
}

type InMemoryWorkflowStore struct {
	workflows   map[string]*WorkflowDef
	executions  map[string]*ExecutionResult
	workflowMu  sync.RWMutex
	executionMu sync.RWMutex
}

func NewInMemoryWorkflowStore() *InMemoryWorkflowStore {
	return &InMemoryWorkflowStore{
		workflows:  make(map[string]*WorkflowDef),
		executions: make(map[string]*ExecutionResult),
	}
}

func (s *InMemoryWorkflowStore) SaveWorkflow(ctx context.Context, def *WorkflowDef) error {
	if def == nil {
		return ErrWorkflowNameEmpty
	}

	s.workflowMu.Lock()
	defer s.workflowMu.Unlock()

	s.workflows[def.Name] = def
	return nil
}

func (s *InMemoryWorkflowStore) GetWorkflow(ctx context.Context, name string) (*WorkflowDef, error) {
	s.workflowMu.RLock()
	defer s.workflowMu.RUnlock()

	def, exists := s.workflows[name]
	if !exists {
		return nil, nil
	}

	return def, nil
}

func (s *InMemoryWorkflowStore) ListWorkflows(ctx context.Context) ([]*WorkflowDef, error) {
	s.workflowMu.RLock()
	defer s.workflowMu.RUnlock()

	result := make([]*WorkflowDef, 0, len(s.workflows))
	for _, def := range s.workflows {
		result = append(result, def)
	}
	return result, nil
}

func (s *InMemoryWorkflowStore) DeleteWorkflow(ctx context.Context, name string) error {
	s.workflowMu.Lock()
	defer s.workflowMu.Unlock()

	delete(s.workflows, name)
	return nil
}

func (s *InMemoryWorkflowStore) SaveExecution(ctx context.Context, result *ExecutionResult) error {
	if result == nil {
		return nil
	}

	s.executionMu.Lock()
	defer s.executionMu.Unlock()

	s.executions[result.RunID] = result
	return nil
}

func (s *InMemoryWorkflowStore) GetExecution(ctx context.Context, runID string) (*ExecutionResult, error) {
	s.executionMu.RLock()
	defer s.executionMu.RUnlock()

	result, exists := s.executions[runID]
	if !exists {
		return nil, nil
	}

	return result, nil
}

func (s *InMemoryWorkflowStore) ListExecutions(ctx context.Context, workflowName string, limit int) ([]*ExecutionResult, error) {
	s.executionMu.RLock()
	defer s.executionMu.RUnlock()

	var result []*ExecutionResult
	for _, exec := range s.executions {
		if workflowName == "" || exec.WorkflowID == workflowName {
			result = append(result, exec)
		}
	}

	if limit > 0 && len(result) > limit {
		result = result[:limit]
	}

	return result, nil
}
