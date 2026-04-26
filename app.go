package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync/atomic"
	"time"

	"NecoClicker/internal/engine"
	"NecoClicker/internal/hotkey"
	"NecoClicker/internal/macro"
	"NecoClicker/internal/winmouse"

	"fyne.io/systray"
	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed build/windows/icon.ico
var trayIconIco []byte

type App struct {
	ctx     context.Context
	cfg     *macro.Config
	engine  *engine.Engine
	hotkeys *hotkey.Manager

	trayToggleItem atomic.Value // *systray.MenuItem
	trayPinItem    atomic.Value // *systray.MenuItem
}

func NewApp() *App {
	cfg, err := macro.Load()
	if err != nil {
		log.Printf("config load: %v", err)
		cfg = macro.DefaultConfig()
	}
	a := &App{cfg: cfg, hotkeys: hotkey.NewManager()}
	a.engine = engine.New(a.logEvent)
	return a
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.engine.OnStateChange(func(running bool) {
		wruntime.EventsEmit(a.ctx, "engine:state", running)
		if it, ok := a.trayToggleItem.Load().(*systray.MenuItem); ok && it != nil {
			if running {
				it.SetTitle("Остановить кликер")
			} else {
				it.SetTitle("Запустить активный профиль")
			}
		}
	})
	if err := a.hotkeys.Start(); err != nil {
		log.Printf("hotkey start: %v", err)
	}
	a.rebindHotkeys()
	a.engine.StartCPSReporter(a.ctx, func(r engine.CPSReport) {
		wruntime.EventsEmit(a.ctx, "engine:cps", r)
	})

	if a.cfg.AlwaysOnTop {
		wruntime.WindowSetAlwaysOnTop(a.ctx, true)
	}

	go a.runTray()
}

func (a *App) shutdown(ctx context.Context) {
	a.engine.StopCPSReporter()
	a.engine.Stop()
	a.hotkeys.Stop()
	systray.Quit()
}

func (a *App) logEvent(line string) {
	if a.ctx != nil {
		wruntime.EventsEmit(a.ctx, "engine:log", line)
	}
}

// ---------------- Конфиг ----------------

func (a *App) GetConfig() *macro.Config { return a.cfg }

func (a *App) SetTheme(name string) error {
	a.cfg.Theme = name
	return macro.Save(a.cfg)
}

func (a *App) SetAlwaysOnTop(v bool) error {
	a.cfg.AlwaysOnTop = v
	if a.ctx != nil {
		wruntime.WindowSetAlwaysOnTop(a.ctx, v)
	}
	if it, ok := a.trayPinItem.Load().(*systray.MenuItem); ok && it != nil {
		if v {
			it.Check()
		} else {
			it.Uncheck()
		}
	}
	return macro.Save(a.cfg)
}

// ImportConfig — заменяет текущий конфиг данными из JSON-строки.
func (a *App) ImportConfig(data string) error {
	cfg := &macro.Config{}
	if err := json.Unmarshal([]byte(data), cfg); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}
	// валидируем через ту же миграцию
	cfg.Migrate()
	a.cfg = cfg
	a.rebindHotkeys()
	if a.ctx != nil {
		wruntime.WindowSetAlwaysOnTop(a.ctx, a.cfg.AlwaysOnTop)
	}
	return macro.Save(a.cfg)
}

// ExportConfig — отдаёт текущий конфиг как JSON-строку.
func (a *App) ExportConfig() (string, error) {
	b, err := json.MarshalIndent(a.cfg, "", "  ")
	return string(b), err
}

// ImportConfigFromFile — выбор файла через системный диалог.
func (a *App) ImportConfigFromFile() error {
	if a.ctx == nil {
		return fmt.Errorf("no app context")
	}
	path, err := wruntime.OpenFileDialog(a.ctx, wruntime.OpenDialogOptions{
		Title: "Импорт конфига NecoClicker",
		Filters: []wruntime.FileFilter{
			{DisplayName: "NecoClicker config", Pattern: "*.necoclicker.json;*.json"},
		},
	})
	if err != nil || path == "" {
		return err
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return a.ImportConfig(string(b))
}

// ExportConfigToFile — диалог сохранения.
func (a *App) ExportConfigToFile() error {
	if a.ctx == nil {
		return fmt.Errorf("no app context")
	}
	ts := time.Now().Format("20060102-150405")
	path, err := wruntime.SaveFileDialog(a.ctx, wruntime.SaveDialogOptions{
		Title:           "Экспорт конфига NecoClicker",
		DefaultFilename: "necoclicker-" + ts + ".necoclicker.json",
		Filters: []wruntime.FileFilter{
			{DisplayName: "NecoClicker config", Pattern: "*.necoclicker.json;*.json"},
		},
	})
	if err != nil || path == "" {
		return err
	}
	data, err := a.ExportConfig()
	if err != nil {
		return err
	}
	return os.WriteFile(path, []byte(data), 0o644)
}

// ---------------- Профили ----------------

func (a *App) ListProfiles() []macro.SimpleConfig { return a.cfg.Profiles }
func (a *App) ActiveProfileIndex() int            { return a.cfg.Active }

func (a *App) SetActiveProfile(idx int) error {
	if idx < 0 || idx >= len(a.cfg.Profiles) {
		return fmt.Errorf("profile index %d out of range", idx)
	}
	a.cfg.Active = idx
	a.rebindHotkeys()
	return macro.Save(a.cfg)
}

func (a *App) SaveProfile(idx int, p macro.SimpleConfig) (int, error) {
	if p.Name == "" {
		p.Name = fmt.Sprintf("Profile %d", len(a.cfg.Profiles)+1)
	}
	if p.IntervalMs < 0 {
		p.IntervalMs = 0
	}
	if p.IntervalMs > 600000 {
		p.IntervalMs = 600000
	}
	if p.Button == "" {
		p.Button = macro.BtnLeft
	}
	if idx < 0 || idx >= len(a.cfg.Profiles) {
		a.cfg.Profiles = append(a.cfg.Profiles, p)
		idx = len(a.cfg.Profiles) - 1
	} else {
		a.cfg.Profiles[idx] = p
	}
	a.rebindHotkeys()
	return idx, macro.Save(a.cfg)
}

func (a *App) DeleteProfile(idx int) error {
	if idx < 0 || idx >= len(a.cfg.Profiles) {
		return nil
	}
	a.cfg.Profiles = append(a.cfg.Profiles[:idx], a.cfg.Profiles[idx+1:]...)
	if len(a.cfg.Profiles) == 0 {
		a.cfg.Profiles = []macro.SimpleConfig{macro.DefaultProfile()}
	}
	if a.cfg.Active >= len(a.cfg.Profiles) {
		a.cfg.Active = len(a.cfg.Profiles) - 1
	}
	a.rebindHotkeys()
	return macro.Save(a.cfg)
}

// ---------------- Цепочки ----------------

func (a *App) SaveChain(idx int, ch macro.Chain) error {
	if idx < 0 || idx >= len(a.cfg.Chains) {
		a.cfg.Chains = append(a.cfg.Chains, ch)
	} else {
		a.cfg.Chains[idx] = ch
	}
	a.rebindHotkeys()
	return macro.Save(a.cfg)
}

func (a *App) DeleteChain(idx int) error {
	if idx < 0 || idx >= len(a.cfg.Chains) {
		return nil
	}
	a.cfg.Chains = append(a.cfg.Chains[:idx], a.cfg.Chains[idx+1:]...)
	if a.cfg.ActiveChain >= len(a.cfg.Chains) {
		a.cfg.ActiveChain = len(a.cfg.Chains) - 1
		if a.cfg.ActiveChain < 0 {
			a.cfg.ActiveChain = 0
		}
	}
	a.rebindHotkeys()
	return macro.Save(a.cfg)
}

func (a *App) SetActiveChain(idx int) error {
	if idx < 0 || idx >= len(a.cfg.Chains) {
		return fmt.Errorf("chain index %d out of range", idx)
	}
	a.cfg.ActiveChain = idx
	return macro.Save(a.cfg)
}

func (a *App) ActiveChainIndex() int { return a.cfg.ActiveChain }

// ---------------- Управление движком ----------------

func (a *App) IsRunning() bool { return a.engine.IsRunning() }

func (a *App) StartSimple() {
	a.engine.SetDryRun(false)
	a.engine.RunSimple(a.cfg.ActiveProfile())
}

func (a *App) StartProfile(idx int) error {
	if idx < 0 || idx >= len(a.cfg.Profiles) {
		return fmt.Errorf("profile index %d out of range", idx)
	}
	a.engine.SetDryRun(false)
	a.engine.RunSimple(a.cfg.Profiles[idx])
	return nil
}

func (a *App) StartChain(idx int) {
	if idx < 0 || idx >= len(a.cfg.Chains) {
		return
	}
	a.engine.SetDryRun(false)
	a.engine.RunChain(a.cfg.Chains[idx])
}

func (a *App) StartChainDry(idx int) {
	if idx < 0 || idx >= len(a.cfg.Chains) {
		return
	}
	a.engine.SetDryRun(true)
	a.engine.RunChain(a.cfg.Chains[idx])
}

func (a *App) StartSimpleDry() {
	a.engine.SetDryRun(true)
	a.engine.RunSimple(a.cfg.ActiveProfile())
}

func (a *App) StartProfileDry(idx int) error {
	if idx < 0 || idx >= len(a.cfg.Profiles) {
		return fmt.Errorf("profile index %d out of range", idx)
	}
	a.engine.SetDryRun(true)
	a.engine.RunSimple(a.cfg.Profiles[idx])
	return nil
}

// StartProfileLimited — запуск профиля с ограничениями (для Таймера / Jitter).
func (a *App) StartProfileLimited(idx int, lim macro.RunLimits, dry bool) error {
	if idx < 0 || idx >= len(a.cfg.Profiles) {
		return fmt.Errorf("profile index %d out of range", idx)
	}
	a.engine.SetDryRun(dry)
	a.engine.RunSimpleLimited(a.cfg.Profiles[idx], lim)
	return nil
}

func (a *App) Stop() { a.engine.Stop() }

// ---------------- CPS ----------------

func (a *App) ResetClicks()        { a.engine.ResetClicks() }
func (a *App) TotalClicks() uint64 { return a.engine.TotalClicks() }

// ---------------- Hotkey recorder ----------------

// RecordHotkey ждёт первое глобальное нажатие (любая клавиша/Mouse4/Mouse5)
// и возвращает строку, готовую для сохранения в конфиг.
func (a *App) RecordHotkey(timeoutMs int) (string, error) {
	if timeoutMs <= 0 {
		timeoutMs = 8000
	}
	return a.hotkeys.RecordOnce(time.Duration(timeoutMs) * time.Millisecond)
}

// ---------------- Утилиты ----------------

func (a *App) CursorPos() [2]int {
	x, y := winmouse.GetCursor()
	return [2]int{x, y}
}

func (a *App) ConfigPath() string {
	p, _ := macro.ConfigPath()
	return p
}

func (a *App) ShowWindow() {
	if a.ctx != nil {
		wruntime.WindowShow(a.ctx)
	}
}

func (a *App) HideWindow() {
	if a.ctx != nil {
		wruntime.WindowHide(a.ctx)
	}
}

// rebindHotkeys: активный профиль + все цепочки с непустым хоткеем.
func (a *App) rebindHotkeys() {
	binds := []hotkey.Bind{}

	if len(a.cfg.Profiles) > 0 {
		ap := a.cfg.ActiveProfile()
		if ap.Hotkey != "" {
			binds = append(binds, hotkey.Bind{
				Hotkey: ap.Hotkey,
				Cb: func() {
					a.engine.Toggle(func() {
						a.engine.SetDryRun(false)
						a.engine.RunSimple(a.cfg.ActiveProfile())
					})
				},
			})
		}
	}

	for i := range a.cfg.Chains {
		idx := i
		ch := a.cfg.Chains[i]
		if ch.Hotkey == "" {
			continue
		}
		binds = append(binds, hotkey.Bind{
			Hotkey: ch.Hotkey,
			Cb: func() {
				a.engine.Toggle(func() {
					if idx >= len(a.cfg.Chains) {
						return
					}
					a.engine.SetDryRun(false)
					a.engine.RunChain(a.cfg.Chains[idx])
				})
			},
		})
	}
	if err := a.hotkeys.SetAll(binds); err != nil {
		log.Printf("hotkey rebind: %v", err)
	}
}

// ---------------- Tray ----------------

func (a *App) runTray() {
	systray.Run(func() {
		systray.SetIcon(trayIconIco)
		systray.SetTitle("NecoClicker")
		systray.SetTooltip("NecoClicker — кликер")

		showItem := systray.AddMenuItem("Показать окно", "")
		toggleItem := systray.AddMenuItem("Запустить активный профиль", "Toggle active simple profile")
		a.trayToggleItem.Store(toggleItem)
		pinItem := systray.AddMenuItemCheckbox("Поверх всех окон", "Always-on-top", a.cfg.AlwaysOnTop)
		a.trayPinItem.Store(pinItem)
		systray.AddSeparator()
		quitItem := systray.AddMenuItem("Выйти", "Quit application")

		go func() {
			for {
				select {
				case <-showItem.ClickedCh:
					a.ShowWindow()
				case <-toggleItem.ClickedCh:
					a.engine.Toggle(func() {
						a.engine.SetDryRun(false)
						a.engine.RunSimple(a.cfg.ActiveProfile())
					})
				case <-pinItem.ClickedCh:
					_ = a.SetAlwaysOnTop(!a.cfg.AlwaysOnTop)
				case <-quitItem.ClickedCh:
					if a.ctx != nil {
						wruntime.Quit(a.ctx)
					}
					return
				}
			}
		}()
	}, nil)
}
