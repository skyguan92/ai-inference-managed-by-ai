package agent

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	agentllm "github.com/jguan/ai-inference-managed-by-ai/pkg/agent/llm"
)

// ---------------------------------------------------------------------------
// Conversation.addMessage tests
// ---------------------------------------------------------------------------

func TestConversation_AddMessage_Basic(t *testing.T) {
	conv := &Conversation{
		ID:       "c1",
		Messages: []agentllm.Message{},
	}
	conv.addMessage(agentllm.Message{Role: "user", Content: "hello"})
	require.Len(t, conv.Messages, 1)
	assert.Equal(t, "user", conv.Messages[0].Role)
	assert.Equal(t, "hello", conv.Messages[0].Content)
}

func TestConversationStore_MessageTrimming(t *testing.T) {
	tests := []struct {
		name        string
		messageCnt  int
		wantMaxLen  int
		wantContent string // content of last message after trimming
	}{
		{
			name:       "exactly at limit — no trim",
			messageCnt: maxConversationMessages,
			wantMaxLen: maxConversationMessages,
		},
		{
			name:        "one over limit — oldest trimmed",
			messageCnt:  maxConversationMessages + 1,
			wantMaxLen:  maxConversationMessages,
			wantContent: fmt.Sprintf("msg %d", maxConversationMessages),
		},
		{
			name:        "ten over limit — ten oldest trimmed",
			messageCnt:  maxConversationMessages + 10,
			wantMaxLen:  maxConversationMessages,
			wantContent: fmt.Sprintf("msg %d", maxConversationMessages+9),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			conv := &Conversation{
				ID:       "trim-test",
				Messages: []agentllm.Message{},
			}
			for i := 0; i < tc.messageCnt; i++ {
				conv.addMessage(agentllm.Message{
					Role:    "user",
					Content: fmt.Sprintf("msg %d", i),
				})
			}
			assert.LessOrEqual(t, len(conv.Messages), tc.wantMaxLen)
			if tc.wantContent != "" {
				assert.Equal(t, tc.wantContent, conv.Messages[len(conv.Messages)-1].Content)
			}
		})
	}
}

func TestConversation_AddMessage_UpdatesUpdatedAt(t *testing.T) {
	conv := &Conversation{
		ID:        "ts-test",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	before := conv.UpdatedAt
	time.Sleep(time.Millisecond)
	conv.addMessage(agentllm.Message{Role: "assistant", Content: "reply"})
	assert.True(t, conv.UpdatedAt.After(before))
}

// ---------------------------------------------------------------------------
// ConversationStore — Create & Get
// ---------------------------------------------------------------------------

func TestConversationStore_CreateAndGet(t *testing.T) {
	store := &ConversationStore{conversations: make(map[string]*Conversation)}

	t.Run("GetOrCreate with explicit ID", func(t *testing.T) {
		conv := store.GetOrCreate("explicit-id")
		require.NotNil(t, conv)
		assert.Equal(t, "explicit-id", conv.ID)
		assert.Empty(t, conv.Messages)
		assert.False(t, conv.CreatedAt.IsZero())
		assert.False(t, conv.UpdatedAt.IsZero())
	})

	t.Run("GetOrCreate returns same conversation for same ID", func(t *testing.T) {
		c1 := store.GetOrCreate("shared")
		c2 := store.GetOrCreate("shared")
		assert.Same(t, c1, c2)
	})

	t.Run("GetOrCreate with empty ID generates one", func(t *testing.T) {
		conv := store.GetOrCreate("")
		assert.NotEmpty(t, conv.ID)
		assert.Contains(t, conv.ID, "conv-")
	})

	t.Run("Get returns existing conversation", func(t *testing.T) {
		store.GetOrCreate("findme")
		found := store.Get("findme")
		require.NotNil(t, found)
		assert.Equal(t, "findme", found.ID)
	})

	t.Run("Get returns nil for unknown ID", func(t *testing.T) {
		assert.Nil(t, store.Get("does-not-exist"))
	})
}

// ---------------------------------------------------------------------------
// ConversationStore — Delete
// ---------------------------------------------------------------------------

func TestConversationStore_Delete_Comprehensive(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(s *ConversationStore)
		deleteID   string
		wantOk     bool
		wantExists bool
	}{
		{
			name: "delete existing conversation returns true",
			setup: func(s *ConversationStore) {
				s.GetOrCreate("del-me")
			},
			deleteID:   "del-me",
			wantOk:     true,
			wantExists: false,
		},
		{
			name:       "delete nonexistent conversation returns false",
			setup:      func(s *ConversationStore) {},
			deleteID:   "ghost",
			wantOk:     false,
			wantExists: false,
		},
		{
			name: "double delete returns false on second call",
			setup: func(s *ConversationStore) {
				s.GetOrCreate("double-del")
				s.Delete("double-del")
			},
			deleteID:   "double-del",
			wantOk:     false,
			wantExists: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			store := &ConversationStore{conversations: make(map[string]*Conversation)}
			tc.setup(store)
			ok := store.Delete(tc.deleteID)
			assert.Equal(t, tc.wantOk, ok)
			if tc.wantExists {
				assert.NotNil(t, store.Get(tc.deleteID))
			} else {
				assert.Nil(t, store.Get(tc.deleteID))
			}
		})
	}
}

// ---------------------------------------------------------------------------
// ConversationStore — ListActive (List)
// ---------------------------------------------------------------------------

func TestConversationStore_ListActive(t *testing.T) {
	t.Run("empty store returns empty slice", func(t *testing.T) {
		store := &ConversationStore{conversations: make(map[string]*Conversation)}
		list := store.List()
		assert.NotNil(t, list)
		assert.Empty(t, list)
	})

	t.Run("returns all active conversations", func(t *testing.T) {
		store := &ConversationStore{conversations: make(map[string]*Conversation)}
		ids := []string{"a", "b", "c"}
		for _, id := range ids {
			store.GetOrCreate(id)
		}
		list := store.List()
		assert.Len(t, list, len(ids))

		// verify all IDs are present
		listedIDs := make(map[string]bool)
		for _, c := range list {
			listedIDs[c.ID] = true
		}
		for _, id := range ids {
			assert.True(t, listedIDs[id], "expected ID %q in list", id)
		}
	})

	t.Run("deleted conversations not included in list", func(t *testing.T) {
		store := &ConversationStore{conversations: make(map[string]*Conversation)}
		store.GetOrCreate("keep")
		store.GetOrCreate("remove")
		store.Delete("remove")
		list := store.List()
		assert.Len(t, list, 1)
		assert.Equal(t, "keep", list[0].ID)
	})
}

// ---------------------------------------------------------------------------
// ConversationStore — TTL Cleanup
// ---------------------------------------------------------------------------

func TestConversationStore_TTLCleanup(t *testing.T) {
	store := &ConversationStore{conversations: make(map[string]*Conversation)}

	// Create a conversation and manually set its UpdatedAt to before the TTL cutoff.
	staleConv := store.GetOrCreate("stale")
	staleConv.UpdatedAt = time.Now().Add(-(conversationTTL + time.Minute))

	// Create a fresh conversation.
	store.GetOrCreate("fresh")

	assert.Equal(t, 2, store.Count(), "should start with 2 conversations")

	// Call cleanup directly (not via the 15-minute ticker).
	store.cleanup()

	assert.Equal(t, 1, store.Count(), "stale conversation should be removed")
	assert.Nil(t, store.Get("stale"), "stale conversation should not be retrievable")
	assert.NotNil(t, store.Get("fresh"), "fresh conversation should survive cleanup")
}

func TestConversationStore_TTLCleanup_NothingExpired(t *testing.T) {
	store := &ConversationStore{conversations: make(map[string]*Conversation)}
	store.GetOrCreate("c1")
	store.GetOrCreate("c2")

	store.cleanup()

	assert.Equal(t, 2, store.Count(), "no conversations should be removed if none expired")
}

func TestConversationStore_TTLCleanup_AllExpired(t *testing.T) {
	store := &ConversationStore{conversations: make(map[string]*Conversation)}

	ids := []string{"a", "b", "c"}
	for _, id := range ids {
		conv := store.GetOrCreate(id)
		conv.UpdatedAt = time.Now().Add(-(conversationTTL + time.Second))
	}

	store.cleanup()

	assert.Equal(t, 0, store.Count(), "all expired conversations should be removed")
}

func TestConversationStore_TTLCleanup_ExactCutoff(t *testing.T) {
	store := &ConversationStore{conversations: make(map[string]*Conversation)}

	// Exactly at cutoff boundary should be removed (Before check is strict less-than).
	cutoffConv := store.GetOrCreate("at-cutoff")
	cutoffConv.UpdatedAt = time.Now().Add(-conversationTTL - time.Millisecond)

	freshConv := store.GetOrCreate("just-fresh")
	freshConv.UpdatedAt = time.Now().Add(-conversationTTL + time.Minute)

	store.cleanup()

	assert.Nil(t, store.Get("at-cutoff"), "at-cutoff conversation should be removed")
	assert.NotNil(t, store.Get("just-fresh"), "just-fresh conversation should survive")
}

// ---------------------------------------------------------------------------
// ConversationStore — Concurrent Access
// ---------------------------------------------------------------------------

func TestConversationStore_ConcurrentAccess(t *testing.T) {
	// Use a store without the background goroutine to avoid races with cleanup.
	store := &ConversationStore{conversations: make(map[string]*Conversation)}

	const goroutines = 50
	const opsPerGoroutine = 20

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			id := fmt.Sprintf("conv-%d", idx%5) // 5 shared IDs to force contention
			for j := 0; j < opsPerGoroutine; j++ {
				switch j % 4 {
				case 0:
					store.GetOrCreate(id)
				case 1:
					store.Get(id)
				case 2:
					store.List()
				case 3:
					store.Count()
				}
			}
		}(i)
	}

	wg.Wait()
	// If we reach here without a data race or panic, the test passes.
	assert.GreaterOrEqual(t, store.Count(), 0)
}

func TestConversationStore_ConcurrentGetOrCreate(t *testing.T) {
	store := &ConversationStore{conversations: make(map[string]*Conversation)}

	const goroutines = 100
	var wg sync.WaitGroup
	wg.Add(goroutines)

	// All goroutines race to create the same conversation.
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			store.GetOrCreate("race-id")
		}()
	}

	wg.Wait()
	// Exactly one conversation should exist.
	assert.Equal(t, 1, store.Count())
	conv := store.Get("race-id")
	require.NotNil(t, conv)
	assert.Equal(t, "race-id", conv.ID)
}

func TestConversationStore_ConcurrentAddMessage(t *testing.T) {
	store := &ConversationStore{conversations: make(map[string]*Conversation)}
	conv := store.GetOrCreate("msg-race")

	const goroutines = 50
	var wg sync.WaitGroup
	wg.Add(goroutines)

	// Note: addMessage is not goroutine-safe on its own (it's the caller's
	// responsibility to lock). This test verifies that the store-level operations
	// (GetOrCreate, Get, List, Count) are safe under concurrent access.
	// We serialise addMessage via our own mutex here.
	var mu sync.Mutex
	for i := 0; i < goroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			msg := agentllm.Message{Role: "user", Content: fmt.Sprintf("concurrent msg %d", idx)}
			mu.Lock()
			conv.addMessage(msg)
			mu.Unlock()
			// Concurrent store reads alongside the writes.
			store.Get("msg-race")
			store.Count()
		}(i)
	}

	wg.Wait()
	assert.LessOrEqual(t, len(conv.Messages), maxConversationMessages)
}

func TestConversationStore_ConcurrentDeleteAndRead(t *testing.T) {
	store := &ConversationStore{conversations: make(map[string]*Conversation)}

	const goroutines = 40
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			id := fmt.Sprintf("del-race-%d", idx%10)
			store.GetOrCreate(id)
			store.Get(id)
			store.Delete(id)
			store.Get(id) // should be nil — must not panic
			store.List()
		}(i)
	}

	wg.Wait()
	// Final count: any value 0–10 is valid depending on timing.
	count := store.Count()
	assert.GreaterOrEqual(t, count, 0)
	assert.LessOrEqual(t, count, 10)
}
