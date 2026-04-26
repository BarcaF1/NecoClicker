//go:build windows

// Package hotkey — глобальные хоткеи через низкоуровневые хуки:
//   - WH_KEYBOARD_LL для клавиатуры
//   - WH_MOUSE_LL    для XButton1/XButton2 (Mouse4/Mouse5)
// Оба хука висят на одной OS-thread с собственным message-loop.
//
// Поддерживаемые форматы строк:
//   - "F6", "Ctrl+Shift+F1", "Alt+Q"
//   - "Mouse4", "Mouse5", "Ctrl+Mouse4"
package hotkey

import (
	"errors"
	"fmt"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
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
	whMouseLL    = 14

	wmKeyDown    = 0x0100
	wmSysKeyDown = 0x0104
	wmQuit       = 0x0012

	// Mouse messages в WPARAM low-level mouse hook'а
	wmXButtonDown = 0x020B

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

// Виртуальные "VK" коды для мыши (выходят за обычный VK-диапазон 0x00-0xFF):
//   - 0xE000+1 = Mouse4 (XButton1)
//   - 0xE000+2 = Mouse5 (XButton2)
const (
	vkMouseBase = 0xE000
	VKMouse4    = vkMouseBase + 1
	VKMouse5    = vkMouseBase + 2
)

type kbdllhookstruct struct {
	VkCode      uint32
	ScanCode    uint32
	Flags       uint32
	Time        uint32
	DwExtraInfo uintptr
}

type msllhookstruct struct {
	Pt          [2]int32 // x, y
	MouseData   uint32   // HIWORD = XButton id
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
	key  uint32 // VK или VKMouse4/5
	cb   func()
}

type Manager struct {
	mu       sync.RWMutex
	binds    []binding
	threadID uint32
	hKbd     uintptr
	hMouse   uintptr
	running  atomic.Bool
	stopped  chan struct{}

	recordMu sync.Mutex
	recordCh chan string // nil = не записываем
}

func NewManager() *Manager {
	return &Manager{stopped: make(chan struct{})}
}

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

	// Запись хоткея, если активна
	m.recordMu.Lock()
	ch := m.recordCh
	m.recordMu.Unlock()
	if ch != nil {
		select {
		case ch <- formatCombo(mods, vk):
		default:
		}
	}

	return len(hit) > 0
}

func (m *Manager) keyboardHookProc(nCode int32, wParam, lParam uintptr) uintptr {
	if nCode == 0 && (wParam == wmKeyDown || wParam == wmSysKeyDown) {
		k := (*kbdllhookstruct)(unsafe.Pointer(lParam))
		// Игнорируем сами модификаторы как primary — иначе при удержании
		// Ctrl мы постоянно будем регистрировать "хоткей Ctrl"
		switch k.VkCode {
		case vkShift, vkControl, vkMenu, vkLWin, vkRWin:
		default:
			m.dispatch(k.VkCode)
		}
	}
	ret, _, _ := procCallNextHookEx.Call(0, uintptr(nCode), wParam, lParam)
	return ret
}

func (m *Manager) mouseHookProc(nCode int32, wParam, lParam uintptr) uintptr {
	if nCode == 0 && wParam == wmXButtonDown {
		mi := (*msllhookstruct)(unsafe.Pointer(lParam))
		// HIWORD(mouseData) = 1 → XButton1, 2 → XButton2
		xb := (mi.MouseData >> 16) & 0xFFFF
		var vk uint32
		switch xb {
		case 1:
			vk = VKMouse4
		case 2:
			vk = VKMouse5
		default:
			vk = 0
		}
		if vk != 0 {
			m.dispatch(vk)
		}
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
		tid, _, _ := procGetCurrentThreadId.Call()
		atomic.StoreUint32(&m.threadID, uint32(tid))

		modH, _, _ := procGetModuleHandleW.Call(0)

		kbdCb := syscall.NewCallback(m.keyboardHookProc)
		hKbd, _, callErr := procSetWindowsHookExW.Call(uintptr(whKeyboardLL), kbdCb, modH, 0)
		if hKbd == 0 {
			started <- fmt.Errorf("SetWindowsHookEx (keyboard) failed: %v", callErr)
			close(m.stopped)
			return
		}
		m.hKbd = hKbd

		mouseCb := syscall.NewCallback(m.mouseHookProc)
		hMouse, _, callErr := procSetWindowsHookExW.Call(uintptr(whMouseLL), mouseCb, modH, 0)
		if hMouse == 0 {
			procUnhookWindowsHookEx.Call(m.hKbd)
			started <- fmt.Errorf("SetWindowsHookEx (mouse) failed: %v", callErr)
			close(m.stopped)
			return
		}
		m.hMouse = hMouse

		started <- nil

		var msg winMsg
		for {
			r, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
			if int32(r) <= 0 {
				break
			}
		}
		procUnhookWindowsHookEx.Call(m.hMouse)
		procUnhookWindowsHookEx.Call(m.hKbd)
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

// RecordOnce ждёт первое нажатие любой клавиши/Mouse4/5 и возвращает строку
// в каноническом формате (например "Ctrl+Shift+F1"). Возвращает ErrTimeout
// если за timeout ничего не нажато.
var ErrRecordTimeout = errors.New("hotkey recording timeout")
var ErrAlreadyRecording = errors.New("already recording another hotkey")

func (m *Manager) RecordOnce(timeout time.Duration) (string, error) {
	m.recordMu.Lock()
	if m.recordCh != nil {
		m.recordMu.Unlock()
		return "", ErrAlreadyRecording
	}
	ch := make(chan string, 1)
	m.recordCh = ch
	m.recordMu.Unlock()

	defer func() {
		m.recordMu.Lock()
		m.recordCh = nil
		m.recordMu.Unlock()
	}()

	select {
	case s := <-ch:
		return s, nil
	case <-time.After(timeout):
		return "", ErrRecordTimeout
	}
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

// ---- parser ---------------------------------------------------------------

// Parse разбирает строку вида "Ctrl+Shift+F1", "Mouse4", "Ctrl+Mouse5" → (mods, vk, err).
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
	switch s {
	case "MOUSE4", "M4", "XBUTTON1":
		return VKMouse4, nil
	case "MOUSE5", "M5", "XBUTTON2":
		return VKMouse5, nil
	}
	if len(s) > 1 && s[0] == 'F' {
		var n int
		if _, err := fmt.Sscanf(s, "F%d", &n); err == nil && n >= 1 && n <= 24 {
			return uint32(0x6F + n), nil
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

// formatCombo превращает (mods, vk) обратно в строку.
func formatCombo(mods, vk uint32) string {
	var parts []string
	if mods&ModCtrl != 0 {
		parts = append(parts, "Ctrl")
	}
	if mods&ModAlt != 0 {
		parts = append(parts, "Alt")
	}
	if mods&ModShift != 0 {
		parts = append(parts, "Shift")
	}
	if mods&ModWin != 0 {
		parts = append(parts, "Win")
	}
	parts = append(parts, formatVK(vk))
	return strings.Join(parts, "+")
}

func formatVK(vk uint32) string {
	switch vk {
	case VKMouse4:
		return "Mouse4"
	case VKMouse5:
		return "Mouse5"
	case 0x20:
		return "Space"
	case 0x0D:
		return "Enter"
	case 0x09:
		return "Tab"
	case 0x1B:
		return "Esc"
	case 0x2D:
		return "Insert"
	case 0x2E:
		return "Delete"
	case 0x24:
		return "Home"
	case 0x23:
		return "End"
	case 0x21:
		return "PgUp"
	case 0x22:
		return "PgDn"
	case 0x26:
		return "Up"
	case 0x28:
		return "Down"
	case 0x25:
		return "Left"
	case 0x27:
		return "Right"
	}
	if vk >= 0x70 && vk <= 0x87 {
		return fmt.Sprintf("F%d", vk-0x6F)
	}
	if (vk >= 'A' && vk <= 'Z') || (vk >= '0' && vk <= '9') {
		return string(rune(vk))
	}
	return fmt.Sprintf("VK_%X", vk)
}
