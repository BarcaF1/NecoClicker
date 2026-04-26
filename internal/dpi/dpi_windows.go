//go:build windows

package dpi

import "syscall"

// EnablePerMonitorV2 включает DPI-Aware режим Per-Monitor V2 на уровне процесса.
// Без этого SendInput/SetCursorPos на HiDPI-мониторах будут работать в "виртуальных"
// координатах и кликать не туда. Должно вызываться ДО создания окон.
func EnablePerMonitorV2() {
	user32 := syscall.NewLazyDLL("user32.dll")
	// Сначала пробуем актуальный API (Win10 1703+)
	if p := user32.NewProc("SetProcessDpiAwarenessContext"); p.Find() == nil {
		// DPI_AWARENESS_CONTEXT_PER_MONITOR_AWARE_V2 = -4
		v := int32(-4)
		p.Call(uintptr(v))
		return
	}
	// Фолбек для более старых систем
	if p := syscall.NewLazyDLL("shcore.dll").NewProc("SetProcessDpiAwareness"); p.Find() == nil {
		// PROCESS_PER_MONITOR_DPI_AWARE = 2
		p.Call(uintptr(2))
		return
	}
	if p := user32.NewProc("SetProcessDPIAware"); p.Find() == nil {
		p.Call()
	}
}
