package agent

import (
	"sync"
	"time"

	"github.com/google/uuid"
	agentllm "github.com/jguan/ai-inference-managed-by-ai/pkg/agent/llm"
)

const (
	// maxConversationMessages caps the message history to avoid unbounded memory growth.
	maxConversationMessages = 100
	// conversationTTL is the idle duration after which a conversation is eligible for cleanup.
	conversationTTL = 2 * time.Hour
)

// Conversation holds a single conversation thread.
type Conversation struct {
	ID        string              `json:"id"`
	Messages  []agentllm.Message  `json:"messages"`
	CreatedAt time.Time           `json:"created_at"`
	UpdatedAt time.Time           `json:"updated_at"`
}

// addMessage appends a message and trims history to maxConversationMessages.
func (c *Conversation) addMessage(m agentllm.Message) {
	c.Messages = append(c.Messages, m)
	if len(c.Messages) > maxConversationMessages {
		// Keep the newest messages; drop the oldest non-system ones.
		c.Messages = c.Messages[len(c.Messages)-maxConversationMessages:]
	}
	c.UpdatedAt = time.Now()
}

// ConversationStore manages active conversations in memory with TTL-based cleanup.
type ConversationStore struct {
	mu            sync.RWMutex
	conversations map[string]*Conversation
}

// NewConversationStore creates a new ConversationStore.
func NewConversationStore() *ConversationStore {
	s := &ConversationStore{
		conversations: make(map[string]*Conversation),
	}
	go s.cleanupLoop()
	return s
}

// GetOrCreate returns the existing conversation for id, or creates a new one.
func (s *ConversationStore) GetOrCreate(id string) *Conversation {
	s.mu.Lock()
	defer s.mu.Unlock()

	if conv, ok := s.conversations[id]; ok {
		return conv
	}

	if id == "" {
		id = "conv-" + uuid.New().String()[:8]
	}

	conv := &Conversation{
		ID:        id,
		Messages:  []agentllm.Message{},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	s.conversations[id] = conv
	return conv
}

// Get returns the conversation by id, or nil if not found.
func (s *ConversationStore) Get(id string) *Conversation {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.conversations[id]
}

// Inject inserts or replaces a conversation in the store (used for loading from disk).
func (s *ConversationStore) Inject(conv *Conversation) {
	if conv == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.conversations[conv.ID] = conv
}

// Delete removes a conversation.
func (s *ConversationStore) Delete(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, exists := s.conversations[id]
	delete(s.conversations, id)
	return exists
}

// List returns a snapshot of all conversations.
func (s *ConversationStore) List() []*Conversation {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*Conversation, 0, len(s.conversations))
	for _, c := range s.conversations {
		result = append(result, c)
	}
	return result
}

// Count returns the number of active conversations.
func (s *ConversationStore) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.conversations)
}

// cleanupLoop periodically removes conversations that have been idle past TTL.
func (s *ConversationStore) cleanupLoop() {
	ticker := time.NewTicker(15 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		s.cleanup()
	}
}

func (s *ConversationStore) cleanup() {
	s.mu.Lock()
	defer s.mu.Unlock()
	cutoff := time.Now().Add(-conversationTTL)
	for id, conv := range s.conversations {
		if conv.UpdatedAt.Before(cutoff) {
			delete(s.conversations, id)
		}
	}
}
