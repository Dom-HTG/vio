package state

import "time"

// Session is one interactive working context (typically one TUI launch = one
// session). It owns the Conversation for the process lifetime and carries the
// metadata needed to identify, scope, and persist it.
type Session struct {
	ID           string
	StartedAt    time.Time
	UpdatedAt    time.Time
	WorkspaceDir string
	Conversation *Conversation
}

// Store is the persistence port for sessions. It is implemented by the on-disk
// history store; the agent and TUI depend on this interface rather than a
// concrete implementation so persistence stays swappable and testable.
//
// Implementations must contain conversation data only: never credentials or
// API keys.
type Store interface {
	// Save writes the full session, creating or replacing it by ID.
	Save(s *Session) error
	// Load returns the session with the given ID.
	Load(id string) (*Session, error)
	// List returns lightweight metadata for all stored sessions.
	List() ([]SessionMeta, error)
	// Delete removes the session with the given ID.
	Delete(id string) error
}
