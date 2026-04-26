// Package engine выполняет кликер и цепочки в фоновых goroutine'ах
// с безопасной отменой через context, опциональными лимитами
// (длительность / число кликов) и рандомизацией интервала.
package engine

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"NecoClicker/internal/macro"
	"NecoClicker/internal/winmouse"
)

type Logger func(string)

type CPSReport struct {
	CPS   float64 `json:"cps"`
	Total uint64  `json:"total"`
}

type CPSCallback func(CPSReport)

type Engine struct {
	mu        sync.Mutex
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	running   bool
	dryRun    bool
	log       Logger
	listeners []func(running bool)

	clickCount atomic.Uint64
	cpsCancel  context.CancelFunc
}

func New(log Logger) *Engine {
	if log == nil {
		log = func(string) {}
	}
	return &Engine{log: log}
}

func (e *Engine) IsRunning() bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.running
}

func (e *Engine) IsDryRun() bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.dryRun
}

func (e *Engine) SetDryRun(v bool) {
	e.mu.Lock()
	e.dryRun = v
	e.mu.Unlock()
}

func (e *Engine) OnStateChange(fn func(bool)) {
	e.mu.Lock()
	e.listeners = append(e.listeners, fn)
	e.mu.Unlock()
}

func (e *Engine) emit(running bool) {
	e.mu.Lock()
	e.running = running
	ls := append([]func(bool){}, e.listeners...)
	e.mu.Unlock()
	for _, fn := range ls {
		fn(running)
	}
}

func (e *Engine) Stop() {
	e.mu.Lock()
	c := e.cancel
	e.cancel = nil
	e.mu.Unlock()
	if c != nil {
		c()
	}
	e.wg.Wait()
}

func (e *Engine) Toggle(start func()) {
	if e.IsRunning() {
		e.Stop()
		return
	}
	start()
}

func (e *Engine) start() context.Context {
	e.Stop()
	ctx, cancel := context.WithCancel(context.Background())
	e.mu.Lock()
	e.cancel = cancel
	e.mu.Unlock()
	e.emit(true)
	e.wg.Add(1)
	return ctx
}

func (e *Engine) finish() {
	e.wg.Done()
	e.emit(false)
}

// ---- click counting ---------------------------------------------------------

func (e *Engine) ResetClicks()      { e.clickCount.Store(0) }
func (e *Engine) TotalClicks() uint64 { return e.clickCount.Load() }

func (e *Engine) StartCPSReporter(parent context.Context, cb CPSCallback) {
	ctx, cancel := context.WithCancel(parent)
	e.mu.Lock()
	if e.cpsCancel != nil {
		e.cpsCancel()
	}
	e.cpsCancel = cancel
	e.mu.Unlock()

	go func() {
		const tick = 250 * time.Millisecond
		const window = 4
		t := time.NewTicker(tick)
		defer t.Stop()

		samples := make([]uint64, 0, window)
		last := e.clickCount.Load()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				cur := e.clickCount.Load()
				delta := cur - last
				last = cur
				samples = append(samples, delta)
				if len(samples) > window {
					samples = samples[len(samples)-window:]
				}
				var sum uint64
				for _, s := range samples {
					sum += s
				}
				secs := float64(len(samples)) * tick.Seconds()
				if secs <= 0 {
					secs = tick.Seconds()
				}
				cb(CPSReport{
					CPS:   float64(sum) / secs,
					Total: cur,
				})
			}
		}
	}()
}

func (e *Engine) StopCPSReporter() {
	e.mu.Lock()
	c := e.cpsCancel
	e.cpsCancel = nil
	e.mu.Unlock()
	if c != nil {
		c()
	}
}

// doClick — единая точка отправки клика: инкрементит счётчик и в нужном
// режиме либо реально кликает, либо только пишет лог.
func (e *Engine) doClick(button string, x, y int, useCurrent bool) {
	e.clickCount.Add(1)
	if e.IsDryRun() {
		return
	}
	winmouse.Click(button, x, y, useCurrent)
}

// ---- runners ----------------------------------------------------------------

func (e *Engine) RunSimple(cfg macro.SimpleConfig) {
	e.RunSimpleLimited(cfg, macro.RunLimits{})
}

// RunSimpleLimited — кликер с опциональными лимитами и jitter'ом.
func (e *Engine) RunSimpleLimited(cfg macro.SimpleConfig, lim macro.RunLimits) {
	ctx := e.start()
	go func() {
		defer e.finish()
		btn := string(cfg.Button)
		if btn == "" {
			btn = "left"
		}
		ms := cfg.IntervalMs
		if ms < 0 {
			ms = 0
		}
		desc := fmt.Sprintf("Simple started: btn=%s interval=%gms", btn, ms)
		if lim.JitterMs > 0 {
			desc += fmt.Sprintf(" jitter=±%gms", lim.JitterMs/2)
		}
		if lim.DurationSec > 0 {
			desc += fmt.Sprintf(" duration=%ds", lim.DurationSec)
		}
		if lim.MaxClicks > 0 {
			desc += fmt.Sprintf(" max=%d", lim.MaxClicks)
		}
		e.log(desc)

		// Лимит по времени реализуем как отдельный таймер,
		// который отменяет ctx; счётчик кликов проверяем после каждого fire'а.
		var deadlineCancel context.CancelFunc
		if lim.DurationSec > 0 {
			ctx, deadlineCancel = context.WithTimeout(ctx, time.Duration(lim.DurationSec)*time.Second)
			defer deadlineCancel()
		}

		startTotal := e.clickCount.Load()
		rng := rand.New(rand.NewSource(time.Now().UnixNano()))

		fire := func() {
			if e.IsDryRun() {
				x, y := winmouse.GetCursor()
				if !cfg.UseCurrent {
					x, y = cfg.X, cfg.Y
				}
				e.log(fmt.Sprintf("[dry] click %s at (%d,%d)", btn, x, y))
			}
			e.doClick(btn, cfg.X, cfg.Y, cfg.UseCurrent)
		}

		// Лимит по кликам относительно текущего значения счётчика
		clicksDone := func() bool {
			if lim.MaxClicks == 0 {
				return false
			}
			done := e.clickCount.Load() - startTotal
			return done >= lim.MaxClicks
		}

		nextDelay := func() time.Duration {
			d := ms
			if lim.JitterMs > 0 {
				// рандомное смещение в [-J/2, +J/2]
				d += (rng.Float64() - 0.5) * lim.JitterMs
				if d < 0 {
					d = 0
				}
			}
			return time.Duration(d * float64(time.Millisecond))
		}

		// Hot path: ms == 0 без джиттера → tight loop
		if ms == 0 && lim.JitterMs == 0 {
			for {
				if ctx.Err() != nil || clicksDone() {
					e.log("Simple stopped")
					return
				}
				fire()
			}
		}

		// Обычный путь: select+After (резолюция ~OS scheduler tick)
		fire()
		if clicksDone() {
			e.log("Simple stopped (max clicks)")
			return
		}
		for {
			d := nextDelay()
			select {
			case <-ctx.Done():
				e.log("Simple stopped")
				return
			case <-time.After(d):
				fire()
				if clicksDone() {
					e.log("Simple stopped (max clicks)")
					return
				}
			}
		}
	}()
}

func (e *Engine) RunChain(chain macro.Chain) {
	ctx := e.start()
	go func() {
		defer e.finish()
		e.log(fmt.Sprintf("Chain %q started (loops=%d, steps=%d)", chain.Name, chain.Loops, len(chain.Actions)))
		infinite := chain.Loops <= 0
		for i := 0; infinite || i < chain.Loops; i++ {
			for idx, a := range chain.Actions {
				if ctx.Err() != nil {
					e.log("Chain stopped")
					return
				}
				if !e.executeAction(ctx, a, idx) {
					return
				}
			}
		}
		e.log(fmt.Sprintf("Chain %q done", chain.Name))
	}()
}

func (e *Engine) executeAction(ctx context.Context, a macro.Action, idx int) bool {
	dry := e.IsDryRun()

	switch a.Type {
	case macro.ActionClick:
		x, y := a.X, a.Y
		if a.Relative {
			cx, cy := winmouse.GetCursor()
			x, y = cx+a.X, cy+a.Y
		} else if a.UseCurrent {
			x, y = winmouse.GetCursor()
		}
		btn := string(a.Button)
		if btn == "" {
			btn = "left"
		}
		if dry {
			e.log(fmt.Sprintf("[%d][dry] click %s (%d,%d)", idx, btn, x, y))
		} else {
			e.log(fmt.Sprintf("[%d] click %s (%d,%d)", idx, btn, x, y))
		}
		e.doClick(btn, x, y, a.UseCurrent && !a.Relative)

	case macro.ActionMove:
		x, y := a.X, a.Y
		if a.Relative {
			cx, cy := winmouse.GetCursor()
			x, y = cx+a.X, cy+a.Y
		}
		if dry {
			e.log(fmt.Sprintf("[%d][dry] move (%d,%d)", idx, x, y))
		} else {
			winmouse.SetCursor(x, y)
			e.log(fmt.Sprintf("[%d] move (%d,%d)", idx, x, y))
		}

	case macro.ActionDelay:
		d := time.Duration(a.DelayMs) * time.Millisecond
		select {
		case <-ctx.Done():
			return false
		case <-time.After(d):
		}
	}
	return true
}
