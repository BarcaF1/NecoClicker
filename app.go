package main

import (
	"context"
	"fmt"
	"log"

	"NecoClicker/internal/engine"
	"NecoClicker/internal/hotkey"
	"NecoClicker/internal/macro"
	"NecoClicker/internal/winmouse"

	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx     context.Context
	cfg     *macro.Config
	engine  *engine.Engine
	hotkeys *hotkey.Manager
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
	})
	if err := a.hotkeys.Start(); err != nil {
		log.Printf("hotkey start: %v", err)
	}
	a.rebindHotkeys()
	// CPS-репортер на всё время жизни приложения
	a.engine.StartCPSReporter(a.ctx, func(r engine.CPSReport) {
		wruntime.EventsEmit(a.ctx, "engine:cps", r)
	})
}

func (a *App) shutdown(ctx context.Context) {
	a.engine.StopCPSReporter()
	a.engine.Stop()
	a.hotkeys.Stop()
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

// ---------------- Профили простого кликера (multi-preset) ----------------

// ListProfiles возвращает все простые-кликер профили.
func (a *App) ListProfiles() []macro.SimpleConfig { return a.cfg.Profiles }

// ActiveProfileIndex — индекс активного профиля (по нему запускается StartSimple
// и работает глобальный хоткей).
func (a *App) ActiveProfileIndex() int { return a.cfg.Active }

// SetActiveProfile меняет активный профиль и перепривязывает хоткеи.
func (a *App) SetActiveProfile(idx int) error {
	if idx < 0 || idx >= len(a.cfg.Profiles) {
		return fmt.Errorf("profile index %d out of range", idx)
	}
	a.cfg.Active = idx
	a.rebindHotkeys()
	return macro.Save(a.cfg)
}

// SaveProfile создаёт (idx=-1) или обновляет существующий профиль.
// Возвращает индекс получившейся записи.
func (a *App) SaveProfile(idx int, p macro.SimpleConfig) (int, error) {
	if p.Name == "" {
		p.Name = fmt.Sprintf("Profile %d", len(a.cfg.Profiles)+1)
	}
	// 0 = max speed (tight loop). Минимум — 0, максимум — 600000ms (10 минут).
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

// DeleteProfile удаляет профиль; всегда оставляет хотя бы один.
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
	a.rebindHotkeys()
	return macro.Save(a.cfg)
}

// ---------------- Управление движком ----------------

func (a *App) IsRunning() bool { return a.engine.IsRunning() }

// StartSimple запускает активный профиль.
func (a *App) StartSimple() {
	a.engine.SetDryRun(false)
	a.engine.RunSimple(a.cfg.ActiveProfile())
}

// StartProfile запускает заданный по индексу профиль (без переключения активного).
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

// StartSimpleDry — запуск активного профиля в dry-run режиме (для CPS-замеров без реальных кликов).
func (a *App) StartSimpleDry() {
	a.engine.SetDryRun(true)
	a.engine.RunSimple(a.cfg.ActiveProfile())
}

// StartProfileDry — запуск конкретного профиля в dry-run.
func (a *App) StartProfileDry(idx int) error {
	if idx < 0 || idx >= len(a.cfg.Profiles) {
		return fmt.Errorf("profile index %d out of range", idx)
	}
	a.engine.SetDryRun(true)
	a.engine.RunSimple(a.cfg.Profiles[idx])
	return nil
}

func (a *App) Stop() { a.engine.Stop() }

// ---------------- CPS / счётчик ----------------

func (a *App) ResetClicks() { a.engine.ResetClicks() }

func (a *App) TotalClicks() uint64 { return a.engine.TotalClicks() }

// ---------------- Утилиты ----------------

func (a *App) CursorPos() [2]int {
	x, y := winmouse.GetCursor()
	return [2]int{x, y}
}

func (a *App) ConfigPath() string {
	p, _ := macro.ConfigPath()
	return p
}

// rebindHotkeys: активный профиль + все цепочки с непустым хоткеем.
func (a *App) rebindHotkeys() {
	binds := []hotkey.Bind{}

	if len(a.cfg.Profiles) > 0 {
		// Хоткей АКТИВНОГО профиля toggleит запуск
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
