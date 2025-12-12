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

// Service returns the underlying session service
func (m *Manager) Service() session.Service {
	return m.service
}
