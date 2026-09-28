package state

import "time"

// SessionMeta is the lightweight, summary view of a persisted Session. It is
// what Store.List returns so callers can present or select sessions without
// loading full conversations.
type SessionMeta struct {
	ID           string
	StartedAt    time.Time
	UpdatedAt    time.Time
	WorkspaceDir string
	MessageCount int
}
