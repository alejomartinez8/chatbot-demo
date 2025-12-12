package session

import (
	"context"
	"fmt"

	"google.golang.org/adk/session"
)

// Manager manages agent sessions
type Manager struct {
	service session.Service
}

// NewManager creates a new session manager
func NewManager() *Manager {
	return &Manager{
		service: session.InMemoryService(),
	}
}

// Create creates a new session
func (m *Manager) Create(ctx context.Context, appName, userID string) (session.Session, error) {
	sessResp, err := m.service.Create(ctx, &session.CreateRequest{
		AppName: appName,
		UserID:  userID,
	})
	if err != nil {
		var zeroSess session.Session
		return zeroSess, fmt.Errorf("failed to create session: %w", err)
	}

	return sessResp.Session, nil
}

// GetOrCreate gets an existing session by ID or creates a new one
// This allows reusing sessions for the same threadID
func (m *Manager) GetOrCreate(ctx context.Context, appName, userID, sessionID string) (session.Session, error) {
	// Try to get existing session first
	if sessionID != "" {
		getResp, err := m.service.Get(ctx, &session.GetRequest{
			SessionID: sessionID,
		})
		if err == nil && getResp != nil {
			return getResp.Session, nil
		}
		// If get fails, fall through to create a new session
	}

	// Create a new session if we don't have one or couldn't get it
	return m.Create(ctx, appName, userID)
}

// Service returns the underlying session service
func (m *Manager) Service() session.Service {
	return m.service
}
