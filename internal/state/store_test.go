package state

import (
	"path/filepath"
	"sync"
	"testing"
)

func TestStorePersistsProfilesAndHistoryAcrossInstances(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.db")
	first, err := Open(path)
	if err != nil {
		t.Fatalf("open first store: %v", err)
	}
	defer first.Close()
	if err := first.CreateProfile(ConnectionProfile{
		ID: "prod-web", Description: "Production web server", Host: "10.0.0.4", Port: 22,
		Username: "deploy", Password: "secret",
	}); err != nil {
		t.Fatalf("create profile: %v", err)
	}
	if err := first.RecordHistory(HistoryEntry{
		ConnectionID: "prod-web", DescriptionSnapshot: "Production web server", Kind: "exec",
		Input: "uname -a", Output: "Linux", State: "success",
	}); err != nil {
		t.Fatalf("record history: %v", err)
	}

	second, err := Open(path)
	if err != nil {
		t.Fatalf("open second store: %v", err)
	}
	defer second.Close()
	profile, err := second.GetProfile("prod-web")
	if err != nil {
		t.Fatalf("get profile from second store: %v", err)
	}
	if profile.Host != "10.0.0.4" || profile.Password != "secret" {
		t.Fatalf("unexpected persisted profile: %+v", profile)
	}
	history, err := second.ListHistory("prod-web", 10)
	if err != nil {
		t.Fatalf("list history from second store: %v", err)
	}
	if len(history) != 1 || history[0].Input != "uname -a" || history[0].Output != "Linux" {
		t.Fatalf("unexpected history: %+v", history)
	}
}

func TestStoreHandlesConcurrentReadersAndWriters(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()
	if err := store.CreateProfile(ConnectionProfile{ID: "lab", Description: "Lab host", Host: "127.0.0.1", Port: 22, Username: "user", Password: "secret"}); err != nil {
		t.Fatalf("create profile: %v", err)
	}

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				if err := store.RecordHistory(HistoryEntry{ConnectionID: "lab", DescriptionSnapshot: "Lab host", Kind: "exec", Input: "true", State: "success"}); err != nil {
					t.Errorf("record history: %v", err)
					return
				}
			}
		}()
	}
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := store.ListProfiles(); err != nil {
				t.Errorf("list profiles: %v", err)
			}
			if _, err := store.ListHistory("lab", 200); err != nil {
				t.Errorf("list history: %v", err)
			}
		}()
	}
	wg.Wait()
	history, err := store.ListHistory("lab", 200)
	if err != nil {
		t.Fatalf("list final history: %v", err)
	}
	if len(history) != 80 {
		t.Fatalf("history count = %d, want 80", len(history))
	}
}

func TestSeparateStoresCoordinateConcurrentWrites(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.db")
	first, err := Open(path)
	if err != nil {
		t.Fatalf("open first store: %v", err)
	}
	defer first.Close()
	second, err := Open(path)
	if err != nil {
		t.Fatalf("open second store: %v", err)
	}
	defer second.Close()
	if err := first.CreateProfile(ConnectionProfile{ID: "shared", Description: "Shared connection", Host: "127.0.0.1", Port: 22, Username: "user", Password: "secret"}); err != nil {
		t.Fatalf("create profile: %v", err)
	}

	var wg sync.WaitGroup
	for _, store := range []*Store{first, second} {
		store := store
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 20; i++ {
				if err := store.RecordHistory(HistoryEntry{ConnectionID: "shared", DescriptionSnapshot: "Shared connection", Kind: "exec", Input: "true", State: "success"}); err != nil {
					t.Errorf("record shared history: %v", err)
					return
				}
			}
		}()
	}
	wg.Wait()
	history, err := first.ListHistory("shared", 100)
	if err != nil {
		t.Fatalf("list shared history: %v", err)
	}
	if len(history) != 40 {
		t.Fatalf("history count = %d, want 40", len(history))
	}
}
