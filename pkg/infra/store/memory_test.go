package store

import (
	"testing"
)

func TestMemoryStore_Create_Get(t *testing.T) {
	s := NewMemoryStore()

	t.Run("create and get record", func(t *testing.T) {
		value := map[string]string{"name": "test", "value": "123"}
		err := s.Create("test_table", "key1", value)
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}

		got, exists := s.Get("test_table", "key1")
		if !exists {
			t.Fatal("expected record to exist")
		}

		gotMap, ok := got.(map[string]string)
		if !ok {
			t.Fatalf("expected map[string]string, got %T", got)
		}
		if gotMap["name"] != "test" || gotMap["value"] != "123" {
			t.Errorf("got %v, want {name:test, value:123}", gotMap)
		}
	})

	t.Run("create duplicate key fails", func(t *testing.T) {
		value := map[string]string{"name": "test2"}
		err := s.Create("test_table", "key1", value)
		if err == nil {
			t.Fatal("expected error for duplicate key, got nil")
		}
	})

	t.Run("get non-existent key", func(t *testing.T) {
		_, exists := s.Get("test_table", "nonexistent")
		if exists {
			t.Fatal("expected record to not exist")
		}
	})

	t.Run("get from non-existent table", func(t *testing.T) {
		_, exists := s.Get("nonexistent_table", "key1")
		if exists {
			t.Fatal("expected record to not exist")
		}
	})
}

func TestMemoryStore_List(t *testing.T) {
	s := NewMemoryStore()

	t.Run("list empty table", func(t *testing.T) {
		result := s.List("empty_table")
		if len(result) != 0 {
			t.Errorf("expected empty list, got %d items", len(result))
		}
	})

	t.Run("list records", func(t *testing.T) {
		s.Create("items", "item1", "value1")
		s.Create("items", "item2", "value2")
		s.Create("items", "item3", "value3")

		result := s.List("items")
		if len(result) != 3 {
			t.Errorf("expected 3 items, got %d", len(result))
		}

		values := make(map[string]bool)
		for _, v := range result {
			values[v.(string)] = true
		}
		if !values["value1"] || !values["value2"] || !values["value3"] {
			t.Error("list missing expected values")
		}
	})

	t.Run("list is isolated per table", func(t *testing.T) {
		s.Create("table_a", "key1", "a1")
		s.Create("table_b", "key1", "b1")

		aList := s.List("table_a")
		bList := s.List("table_b")

		if len(aList) != 1 || aList[0] != "a1" {
			t.Error("table_a has wrong content")
		}
		if len(bList) != 1 || bList[0] != "b1" {
			t.Error("table_b has wrong content")
		}
	})
}

func TestMemoryStore_Update_Delete(t *testing.T) {
	s := NewMemoryStore()

	t.Run("update existing record", func(t *testing.T) {
		s.Create("updates", "key1", "original")
		err := s.Update("updates", "key1", "updated")
		if err != nil {
			t.Fatalf("Update failed: %v", err)
		}

		got, _ := s.Get("updates", "key1")
		if got != "updated" {
			t.Errorf("got %v, want 'updated'", got)
		}
	})

	t.Run("update non-existent table fails", func(t *testing.T) {
		err := s.Update("nonexistent", "key1", "value")
		if err == nil {
			t.Fatal("expected error for non-existent table")
		}
	})

	t.Run("update non-existent key fails", func(t *testing.T) {
		err := s.Update("updates", "nonexistent", "value")
		if err == nil {
			t.Fatal("expected error for non-existent key")
		}
	})

	t.Run("delete existing record", func(t *testing.T) {
		s.Create("deletes", "key1", "value")
		err := s.Delete("deletes", "key1")
		if err != nil {
			t.Fatalf("Delete failed: %v", err)
		}

		_, exists := s.Get("deletes", "key1")
		if exists {
			t.Error("record should have been deleted")
		}
	})

	t.Run("delete non-existent table fails", func(t *testing.T) {
		err := s.Delete("nonexistent", "key1")
		if err == nil {
			t.Fatal("expected error for non-existent table")
		}
	})

	t.Run("delete non-existent key fails", func(t *testing.T) {
		s.Create("deletes2", "key1", "value")
		err := s.Delete("deletes2", "nonexistent")
		if err == nil {
			t.Fatal("expected error for non-existent key")
		}
	})
}

func TestMemoryStore_NextID(t *testing.T) {
	s := NewMemoryStore()

	t.Run("incrementing IDs", func(t *testing.T) {
		id1 := s.NextID("users")
		id2 := s.NextID("users")
		id3 := s.NextID("users")

		if id1 != "1" || id2 != "2" || id3 != "3" {
			t.Errorf("got IDs %s, %s, %s, want 1, 2, 3", id1, id2, id3)
		}
	})

	t.Run("isolated counters per table", func(t *testing.T) {
		a1 := s.NextID("table_a")
		a2 := s.NextID("table_a")
		b1 := s.NextID("table_b")
		b2 := s.NextID("table_b")

		if a1 != "1" || a2 != "2" {
			t.Errorf("table_a IDs wrong: %s, %s", a1, a2)
		}
		if b1 != "1" || b2 != "2" {
			t.Errorf("table_b IDs wrong: %s, %s", b1, b2)
		}
	})
}

func TestMemoryStore_Clear(t *testing.T) {
	s := NewMemoryStore()

	s.Create("table1", "key1", "value1")
	s.NextID("table1")

	s.Clear()

	result := s.List("table1")
	if len(result) != 0 {
		t.Error("expected empty list after Clear")
	}

	id := s.NextID("table1")
	if id != "1" {
		t.Errorf("counter should reset to 1, got %s", id)
	}
}
