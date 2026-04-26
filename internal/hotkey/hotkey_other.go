//go:build !windows

package hotkey

type Manager struct{}
type Bind struct {
	Hotkey string
	Cb     func()
}

func NewManager() *Manager        { return &Manager{} }
func (m *Manager) SetAll([]Bind) error { return nil }
func (m *Manager) Start() error   { return nil }
func (m *Manager) Stop()          {}
