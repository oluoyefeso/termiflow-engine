package taste

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

var ctx = context.Background()

// --- InMemoryProfileStore tests ---

func TestInMemoryStore_SaveAndLoad(t *testing.T) {
	store := NewInMemoryProfileStore()
	profile := &TasteProfile{
		Version:    1,
		UserID:     "user1",
		Affinities: map[string]float64{"go": 0.8, "rust": -0.3},
		SignalCount: 10,
		LastUpdated: time.Now(),
	}

	if err := store.Save(ctx, profile); err != nil {
		t.Fatalf("save: %v", err)
	}

	loaded, err := store.Load(ctx, "user1")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if loaded == nil {
		t.Fatal("loaded profile is nil")
	}
	if loaded.UserID != "user1" {
		t.Errorf("UserID = %q, want %q", loaded.UserID, "user1")
	}
	if loaded.Affinities["go"] != 0.8 {
		t.Errorf("affinity[go] = %.2f, want 0.8", loaded.Affinities["go"])
	}
	if loaded.SignalCount != 10 {
		t.Errorf("SignalCount = %d, want 10", loaded.SignalCount)
	}
}

func TestInMemoryStore_LoadNonExistent(t *testing.T) {
	store := NewInMemoryProfileStore()
	profile, err := store.Load(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if profile != nil {
		t.Errorf("expected nil profile, got %+v", profile)
	}
}

func TestInMemoryStore_Delete(t *testing.T) {
	store := NewInMemoryProfileStore()
	profile := &TasteProfile{Version: 1, UserID: "user1", Affinities: map[string]float64{"go": 0.5}}

	store.Save(ctx, profile)
	store.Delete(ctx, "user1")

	loaded, err := store.Load(ctx, "user1")
	if err != nil {
		t.Fatalf("load after delete: %v", err)
	}
	if loaded != nil {
		t.Error("expected nil after delete")
	}
}

func TestInMemoryStore_DeleteNonExistent(t *testing.T) {
	store := NewInMemoryProfileStore()
	err := store.Delete(ctx, "nonexistent")
	if err != nil {
		t.Errorf("delete nonexistent: %v", err)
	}
}

func TestInMemoryStore_Isolation(t *testing.T) {
	store := NewInMemoryProfileStore()
	profile := &TasteProfile{
		Version:    1,
		UserID:     "user1",
		Affinities: map[string]float64{"go": 0.5},
	}

	store.Save(ctx, profile)

	// Mutate the original after saving.
	profile.Affinities["go"] = 0.99
	profile.Affinities["new"] = 0.1

	// Loaded profile should have the saved values, not the mutated ones.
	loaded, _ := store.Load(ctx, "user1")
	if loaded.Affinities["go"] != 0.5 {
		t.Errorf("affinity[go] = %.2f, want 0.5 (deep copy on save)", loaded.Affinities["go"])
	}
	if _, ok := loaded.Affinities["new"]; ok {
		t.Error("loaded profile has 'new' key, expected deep copy isolation")
	}

	// Mutate the loaded profile, verify store is unaffected.
	loaded.Affinities["go"] = 0.0
	loaded2, _ := store.Load(ctx, "user1")
	if loaded2.Affinities["go"] != 0.5 {
		t.Errorf("affinity[go] = %.2f after mutating load result, want 0.5 (deep copy on load)", loaded2.Affinities["go"])
	}
}

// --- FileProfileStore tests ---

func TestFileStore_SaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	store := NewFileProfileStore(dir)
	profile := &TasteProfile{
		Version:    1,
		UserID:     "user1",
		Affinities: map[string]float64{"go": 0.8, "rust": -0.3},
		SignalCount: 10,
		LastUpdated: time.Now().Truncate(time.Second),
	}

	if err := store.Save(ctx, profile); err != nil {
		t.Fatalf("save: %v", err)
	}

	loaded, err := store.Load(ctx, "user1")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if loaded == nil {
		t.Fatal("loaded profile is nil")
	}
	if loaded.Affinities["go"] != 0.8 {
		t.Errorf("affinity[go] = %.2f, want 0.8", loaded.Affinities["go"])
	}
	if loaded.SignalCount != 10 {
		t.Errorf("SignalCount = %d, want 10", loaded.SignalCount)
	}
}

func TestFileStore_LoadNonExistent(t *testing.T) {
	dir := t.TempDir()
	store := NewFileProfileStore(dir)
	profile, err := store.Load(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if profile != nil {
		t.Errorf("expected nil profile, got %+v", profile)
	}
}

func TestFileStore_Delete(t *testing.T) {
	dir := t.TempDir()
	store := NewFileProfileStore(dir)
	profile := &TasteProfile{Version: 1, UserID: "user1", Affinities: map[string]float64{"go": 0.5}}

	store.Save(ctx, profile)
	store.Delete(ctx, "user1")

	loaded, _ := store.Load(ctx, "user1")
	if loaded != nil {
		t.Error("expected nil after delete")
	}
}

func TestFileStore_DeleteNonExistent(t *testing.T) {
	dir := t.TempDir()
	store := NewFileProfileStore(dir)
	err := store.Delete(ctx, "nonexistent")
	if err != nil {
		t.Errorf("delete nonexistent: %v", err)
	}
}

func TestFileStore_JSONReadable(t *testing.T) {
	dir := t.TempDir()
	store := NewFileProfileStore(dir)
	profile := &TasteProfile{
		Version:    1,
		UserID:     "user1",
		Affinities: map[string]float64{"go": 0.8},
		SignalCount: 5,
	}

	store.Save(ctx, profile)

	// Read raw file and verify it's valid, pretty-printed JSON.
	data, err := os.ReadFile(filepath.Join(dir, "user1.json"))
	if err != nil {
		t.Fatalf("read file: %v", err)
	}

	if !json.Valid(data) {
		t.Fatal("file is not valid JSON")
	}

	// Verify pretty-printed (contains newlines and indentation).
	content := string(data)
	if len(content) < 10 {
		t.Fatal("file too short to be pretty-printed")
	}
	if content[0] != '{' {
		t.Errorf("expected JSON object, got %q", content[:1])
	}

	// Verify it can be parsed.
	var parsed TasteProfile
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if parsed.Affinities["go"] != 0.8 {
		t.Errorf("parsed affinity[go] = %.2f, want 0.8", parsed.Affinities["go"])
	}
}

func TestFileStore_InvalidUserID(t *testing.T) {
	dir := t.TempDir()
	store := NewFileProfileStore(dir)

	// Path traversal.
	_, err := store.Load(ctx, "../../../etc/passwd")
	if err == nil {
		t.Error("expected error for path traversal userID")
	}

	// Path separator.
	_, err = store.Load(ctx, "user/evil")
	if err == nil {
		t.Error("expected error for userID with path separator")
	}

	// Empty.
	_, err = store.Load(ctx, "")
	if err == nil {
		t.Error("expected error for empty userID")
	}
}

func TestFileStore_CorruptedJSON(t *testing.T) {
	dir := t.TempDir()
	store := NewFileProfileStore(dir)

	// Write corrupted JSON.
	path := filepath.Join(dir, "baduser.json")
	os.WriteFile(path, []byte("{not json at all"), 0644)

	_, err := store.Load(ctx, "baduser")
	if err == nil {
		t.Error("expected error for corrupted JSON")
	}
}

func TestFileStore_DirectoryCreation(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "deep")
	store := NewFileProfileStore(dir)
	profile := &TasteProfile{Version: 1, UserID: "user1", Affinities: map[string]float64{}}

	if err := store.Save(ctx, profile); err != nil {
		t.Fatalf("save with nested dir: %v", err)
	}

	loaded, err := store.Load(ctx, "user1")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if loaded == nil {
		t.Fatal("expected non-nil profile after save to nested dir")
	}
}
