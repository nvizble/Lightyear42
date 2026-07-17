package cache

import (
	"path/filepath"
	"testing"
	"time"
)

func openTestStore(t *testing.T) *Store {
	t.Helper()
	store, err := Open(filepath.Join(t.TempDir(), "sub", "cache.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { store.Close() })
	return store
}

func TestStore_SetGet(t *testing.T) {
	t.Parallel()
	store := openTestStore(t)

	if err := store.Set("k", []byte("v1"), time.Minute); err != nil {
		t.Fatalf("Set: %v", err)
	}

	value, hit, err := store.Get("k")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !hit {
		t.Fatal("expected cache hit")
	}
	if string(value) != "v1" {
		t.Errorf("value = %q, want v1", value)
	}

	// Overwrite must replace the value.
	if err := store.Set("k", []byte("v2"), time.Minute); err != nil {
		t.Fatalf("Set (overwrite): %v", err)
	}
	value, _, _ = store.Get("k")
	if string(value) != "v2" {
		t.Errorf("value = %q, want v2", value)
	}
}

func TestStore_Miss(t *testing.T) {
	t.Parallel()
	store := openTestStore(t)

	_, hit, err := store.Get("absent")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if hit {
		t.Fatal("expected miss for absent key")
	}
}

func TestStore_Expiry(t *testing.T) {
	t.Parallel()
	store := openTestStore(t)

	if err := store.Set("k", []byte("v"), -time.Second); err != nil {
		t.Fatalf("Set: %v", err)
	}

	_, hit, err := store.Get("k")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if hit {
		t.Fatal("expected miss for expired entry")
	}
}

func TestStore_Clear(t *testing.T) {
	t.Parallel()
	store := openTestStore(t)

	if err := store.Set("a", []byte("1"), time.Minute); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if err := store.Set("b", []byte("2"), time.Minute); err != nil {
		t.Fatalf("Set: %v", err)
	}

	if err := store.Clear(); err != nil {
		t.Fatalf("Clear: %v", err)
	}

	for _, key := range []string{"a", "b"} {
		if _, hit, _ := store.Get(key); hit {
			t.Errorf("key %q still present after Clear", key)
		}
	}
}
