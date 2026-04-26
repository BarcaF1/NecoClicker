//go:build !windows

package hotkey

import (
	"errors"
	"time"
)

type Manager struct{}
type Bind struct {
	Hotkey string
	Cb     func()
}

func NewManager() *Manager                                  { return &Manager{} }
func (m *Manager) SetAll([]Bind) error                      { return nil }
func (m *Manager) Start() error                             { return nil }
func (m *Manager) Stop()                                    {}
func (m *Manager) RecordOnce(time.Duration) (string, error) { return "", errors.New("not implemented") }
