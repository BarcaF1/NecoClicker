//go:build windows

// Package hotkey — глобальные горячие клавиши через низкоуровневый
// клавиатурный хук (WH_KEYBOARD_LL). В отличие от RegisterHotKey это
// работает даже когда окно свёрнуто и не требует уникальности комбинаций
// в системе. Хук висит на отдельной OS-thread с собственным message-loop.
package hotkey

import (
	"errors"
	"fmt"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"unsafe"
)

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")

	procSetWindowsHookExW   = user32.NewProc("SetWindowsHookExW")
	procCallNextHookEx      = user32.NewProc("CallNextHookEx")
	procUnhookWindowsHookEx = user32.NewProc("UnhookWindowsHookEx")
	procGetMessageW         = user32.NewProc("GetMessageW")
	procPostThreadMessageW  = user32.NewProc("PostThreadMessageW")
	procGetAsyncKeyState    = user32.NewProc("GetAsyncKeyState")
	procGetCurrentThreadId  = kernel32.NewProc("GetCurrentThreadId")
	procGetModuleHandleW    = kernel32.NewProc("GetModuleHandleW")
)

const (
	whKeyboardLL = 13
	wmKeyDown    = 0x0100
	wmSysKeyDown = 0x0104
	wmQuit       = 0x0012

	vkShift   = 0x10
	vkControl = 0x11
	vkMenu    = 0x12
	vkLWin    = 0x5B
	vkRWin    = 0x5C
)

const (
	ModCtrl  uint32 = 1 << 0
	ModAlt   uint32 = 1 << 1
	ModShift uint32 = 1 << 2
	ModWin   uint32 = 1 << 3
)

type kbdllhookstruct struct {
	VkCode      uint32
	ScanCode    uint32
	Flags       uint32
	Time        uint32
	DwExtraInfo uintptr
}

type winMsg struct {
	Hwnd     uintptr
	Message  uint32
	_        uint32
	WParam   uintptr
	LParam   uintptr
	Time     uint32
	X, Y     int32
	LPrivate uint32
}

type binding struct {
	mods uint32
	key  uint32
	cb   func()
}

type Manager struct {
	mu       sync.RWMutex
	binds    []binding
	threadID uint32
	hHook    uintptr
	running  atomic.Bool
	stopped  chan struct{}
}

func NewManager() *Manager {
	return &Manager{stopped: make(chan struct{})}
}

// SetAll заменяет все привязки. Каждая привязка — pre-parsed (mods,key,cb).
type Bind struct {
	Hotkey string
	Cb     func()
}

func (m *Manager) SetAll(bs []Bind) error {
	out := make([]binding, 0, len(bs))
	for _, b := range bs {
		if strings.TrimSpace(b.Hotkey) == "" {
			continue
		}
		mods, key, err := Parse(b.Hotkey)
		if err != nil {
			return fmt.Errorf("hotkey %q: %w", b.Hotkey, err)
		}
		out = append(out, binding{mods: mods, key: key, cb: b.Cb})
	}
	m.mu.Lock()
	m.binds = out
	m.mu.Unlock()
	return nil
}

func (m *Manager) dispatch(vk uint32) bool {
	mods := currentMods()
	m.mu.RLock()
	var hit []func()
	for _, b := range m.binds {
		if b.key == vk && b.mods == mods {
			hit = append(hit, b.cb)
		}
	}
	m.mu.RUnlock()
	for _, cb := range hit {
		go cb()
	}
	return len(hit) > 0
}

func (m *Manager) hookProc(nCode int32, wParam, lParam uintptr) uintptr {
	if nCode == 0 && (wParam == wmKeyDown || wParam == wmSysKeyDown) {
		k := (*kbdllhookstruct)(unsafe.Pointer(lParam))
		// Не глотаем нажатие — пусть фокусированное приложение тоже его получит.
		m.dispatch(k.VkCode)
	}
	ret, _, _ := procCallNextHookEx.Call(0, uintptr(nCode), wParam, lParam)
	return ret
}

func (m *Manager) Start() error {
	if !m.running.CompareAndSwap(false, true) {
		return errors.New("hotkey manager already started")
	}
	started := make(chan error, 1)
	go func() {
		runtime.LockOSThread()
		// thread не размораживаем — он живёт до Stop().

		tid, _, _ := procGetCurrentThreadId.Call()
		atomic.StoreUint32(&m.threadID, uint32(tid))

		modH, _, _ := procGetModuleHandleW.Call(0)
		cb := syscall.NewCallback(m.hookProc)
		h, _, callErr := procSetWindowsHookExW.Call(
			uintptr(whKeyboardLL),
			cb,
			modH,
			0,
		)
		if h == 0 {
			started <- fmt.Errorf("SetWindowsHookEx failed: %v", callErr)
			close(m.stopped)
			return
		}
		m.hHook = h
		started <- nil

		var msg winMsg
		for {
			r, _, _ := procGetMessageW.Call(
				uintptr(unsafe.Pointer(&msg)),
				0, 0, 0,
			)
			if int32(r) <= 0 {
				break
			}
		}
		procUnhookWindowsHookEx.Call(m.hHook)
		close(m.stopped)
	}()
	return <-started
}

func (m *Manager) Stop() {
	if !m.running.CompareAndSwap(true, false) {
		return
	}
	tid := atomic.LoadUint32(&m.threadID)
	procPostThreadMessageW.Call(uintptr(tid), wmQuit, 0, 0)
	<-m.stopped
}

func currentMods() uint32 {
	var m uint32
	if pressed(vkControl) {
		m |= ModCtrl
	}
	if pressed(vkMenu) {
		m |= ModAlt
	}
	if pressed(vkShift) {
		m |= ModShift
	}
	if pressed(vkLWin) || pressed(vkRWin) {
		m |= ModWin
	}
	return m
}

func pressed(vk int) bool {
	r, _, _ := procGetAsyncKeyState.Call(uintptr(vk))
	return r&0x8000 != 0
}

// Parse разбирает строку вида "Ctrl+Shift+F1" → (mods, vk, err).
// Допустимые модификаторы: Ctrl/Control, Alt, Shift, Win.
// Допустимые клавиши: A-Z, 0-9, F1-F24, Space, Enter, Tab, Esc/Escape,
// Insert/Ins, Delete/Del, Home, End, PgUp, PgDn.
func Parse(s string) (mods, vk uint32, err error) {
	parts := strings.Split(strings.ToUpper(strings.ReplaceAll(s, " ", "")), "+")
	if len(parts) == 0 || parts[0] == "" {
		return 0, 0, errors.New("empty hotkey")
	}
	for i, p := range parts {
		if i < len(parts)-1 {
			switch p {
			case "CTRL", "CONTROL":
				mods |= ModCtrl
			case "ALT":
				mods |= ModAlt
			case "SHIFT":
				mods |= ModShift
			case "WIN", "META":
				mods |= ModWin
			default:
				return 0, 0, fmt.Errorf("unknown modifier %q", p)
			}
			continue
		}
		key, e := parseKey(p)
		if e != nil {
			return 0, 0, e
		}
		vk = key
	}
	return mods, vk, nil
}

func parseKey(s string) (uint32, error) {
	if len(s) > 1 && s[0] == 'F' {
		var n int
		if _, err := fmt.Sscanf(s, "F%d", &n); err == nil && n >= 1 && n <= 24 {
			return uint32(0x6F + n), nil // F1=0x70 ... F24=0x87
		}
	}
	if len(s) == 1 {
		c := s[0]
		if (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') {
			return uint32(c), nil
		}
	}
	switch s {
	case "SPACE":
		return 0x20, nil
	case "ENTER", "RETURN":
		return 0x0D, nil
	case "TAB":
		return 0x09, nil
	case "ESC", "ESCAPE":
		return 0x1B, nil
	case "INS", "INSERT":
		return 0x2D, nil
	case "DEL", "DELETE":
		return 0x2E, nil
	case "HOME":
		return 0x24, nil
	case "END":
		return 0x23, nil
	case "PGUP", "PAGEUP":
		return 0x21, nil
	case "PGDN", "PAGEDOWN":
		return 0x22, nil
	case "UP":
		return 0x26, nil
	case "DOWN":
		return 0x28, nil
	case "LEFT":
		return 0x25, nil
	case "RIGHT":
		return 0x27, nil
	}
	return 0, fmt.Errorf("unknown key %q", s)
}
