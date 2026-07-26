package terminal

import (
	"fmt"
	"sync"

	"github.com/google/uuid"
)

type Entry struct {
	ID           string
	ConnectionID string
	Transport    string
	Session      *Session
	Close        func() error
	Resize       func(width, height int) error
}

type Registry struct {
	mu      sync.RWMutex
	entries map[string]*Entry
}

func NewRegistry() *Registry {
	return &Registry{entries: make(map[string]*Entry)}
}

func (r *Registry) Register(connectionID, transport string, session *Session, closeFn func() error, resizeFn func(int, int) error) *Entry {
	entry := &Entry{
		ID:           "t-" + uuid.NewString()[:8],
		ConnectionID: connectionID,
		Transport:    transport,
		Session:      session,
		Close:        closeFn,
		Resize:       resizeFn,
	}
	r.mu.Lock()
	r.entries[entry.ID] = entry
	r.mu.Unlock()
	return entry
}

func (r *Registry) Get(id string) (*Entry, error) {
	r.mu.RLock()
	entry := r.entries[id]
	r.mu.RUnlock()
	if entry == nil {
		return nil, fmt.Errorf("terminal session %q not found", id)
	}
	return entry, nil
}

func (r *Registry) FindByConnection(connectionID string) *Entry {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, entry := range r.entries {
		if entry.ConnectionID == connectionID {
			return entry
		}
	}
	return nil
}

func (r *Registry) Close(id string) error {
	r.mu.Lock()
	entry := r.entries[id]
	if entry != nil {
		delete(r.entries, id)
	}
	r.mu.Unlock()
	if entry == nil {
		return fmt.Errorf("terminal session %q not found", id)
	}
	entry.Session.Close()
	if entry.Close != nil {
		return entry.Close()
	}
	return nil
}

func (r *Registry) RemoveByConnection(connectionID string) {
	r.mu.Lock()
	for id, entry := range r.entries {
		if entry.ConnectionID == connectionID {
			entry.Session.Close()
			delete(r.entries, id)
		}
	}
	r.mu.Unlock()
}
