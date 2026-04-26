import { useEffect, useRef, useState } from 'react'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import { ScrollArea } from '@/components/ui/scroll-area'
import {
  Play, Square, Trash2, FlaskConical, Activity, Gauge, RotateCcw, ShieldCheck,
  Crosshair, ChevronDown, ChevronRight,
} from 'lucide-react'
import { useConfig } from '@/components/config-provider'
import { useEngine } from '@/components/engine-provider'
import {
  StartChainDry, StartProfile, StartProfileDry, StartChain, Stop, ResetClicks,
} from '../../wailsjs/go/main/App'
import { cn } from '@/lib/utils'

type RunTarget = { kind: 'profile' | 'chain'; idx: number }

export function SandboxPage() {
  const { cfg } = useConfig()
  const { running, log, clearLog, cps, history } = useEngine()
  const scrollRef = useRef<HTMLDivElement>(null)

  const [target, setTarget] = useState<string>('profile:0')
  const [dryRun, setDryRun] = useState(true)
  const [peak, setPeak] = useState(0)
  const [logOpen, setLogOpen] = useState(false)

  // визуальный счётчик "клик-тестов" в зону: каждое попадание — пульс
  const [zonePulses, setZonePulses] = useState<{ id: number; x: number; y: number }[]>([])
  const zoneRef = useRef<HTMLDivElement>(null)
  const pulseId = useRef(0)

  useEffect(() => {
    if (running && cps.cps > peak) setPeak(cps.cps)
  }, [cps, running, peak])

  // Когда лог развёрнут — автоскролл вниз
  useEffect(() => {
    if (!logOpen) return
    if (scrollRef.current) {
      scrollRef.current.scrollTop = scrollRef.current.scrollHeight
    }
  }, [log, logOpen])

  // Каждый прирост total → визуальный пульс в зоне (центральная точка)
  const lastTotalRef = useRef(0)
  useEffect(() => {
    if (cps.total === lastTotalRef.current) return
    const delta = Number(cps.total) - lastTotalRef.current
    lastTotalRef.current = Number(cps.total)
    if (!running || delta <= 0 || delta > 30) return // ignore reset/spike
    const id = pulseId.current++
    setZonePulses((prev) => [...prev.slice(-5), { id, x: 50, y: 50 }])
    setTimeout(() => {
      setZonePulses((prev) => prev.filter((p) => p.id !== id))
    }, 700)
  }, [cps.total, running])

  const profiles = cfg?.profiles ?? []
  const chains = cfg?.chains ?? []

  const parseTarget = (s: string): RunTarget => {
    const [kind, idx] = s.split(':')
    return { kind: kind as 'profile' | 'chain', idx: parseInt(idx) || 0 }
  }

  const start = async () => {
    const t = parseTarget(target)
    setPeak(0)
    await ResetClicks()
    if (t.kind === 'profile') {
      if (dryRun) await StartProfileDry(t.idx)
      else await StartProfile(t.idx)
    } else {
      if (dryRun) await StartChainDry(t.idx)
      else await StartChain(t.idx)
    }
  }

  const reset = async () => {
    await ResetClicks()
    setPeak(0)
  }

  return (
    <div className="grid h-[calc(100vh-9rem)] gap-4 lg:grid-cols-[1fr_340px]">
      <div className="flex min-h-0 flex-col gap-4">
        {/* CPS METER */}
        <Card>
          <CardHeader className="pb-3">
            <div className="flex items-start justify-between gap-3">
              <div>
                <CardTitle className="flex items-center gap-2">
                  <Gauge className="h-5 w-5 text-primary" /> CPS-метр
                </CardTitle>
                <CardDescription>Кликов в секунду · сглажено за последнюю секунду</CardDescription>
              </div>
              <Button variant="ghost" size="sm" onClick={reset} className="shrink-0">
                <RotateCcw className="h-3.5 w-3.5" /> Сбросить
              </Button>
            </div>
          </CardHeader>
          <CardContent className="space-y-3">
            <div className="grid grid-cols-3 gap-2">
              <Stat label="Текущий" value={cps.cps.toFixed(1)} accent active={running} />
              <Stat label="Пик" value={peak.toFixed(1)} />
              <Stat label="Всего" value={Number(cps.total).toLocaleString()} />
            </div>
            <BarChart data={history} />
          </CardContent>
        </Card>

        {/* CLICK TARGET ZONE */}
        <Card className="flex min-h-0 flex-1 flex-col overflow-hidden">
          <CardHeader className="pb-2">
            <div className="flex items-center justify-between gap-3">
              <div>
                <CardTitle className="flex items-center gap-2 text-sm">
                  <Crosshair className="h-4 w-4 text-primary" /> Зона замера
                </CardTitle>
                <CardDescription>
                  Поставь курсор внутрь — кликер будет слать события сюда (в реальном режиме). В безопасном — просто визуализация.
                </CardDescription>
              </div>
            </div>
          </CardHeader>
          <CardContent className="flex flex-1 items-center justify-center pb-5">
            <div
              ref={zoneRef}
              className={cn(
                'relative aspect-video w-full max-w-3xl overflow-hidden rounded-xl border-2 border-dashed transition-all',
                running
                  ? 'border-primary/60 bg-primary/5 glow-primary'
                  : 'border-border bg-muted/20',
              )}
            >
              {/* центральная мишень */}
              <div className="pointer-events-none absolute inset-0 flex items-center justify-center">
                <div className={cn(
                  'flex h-32 w-32 items-center justify-center rounded-full border-2 transition-all',
                  running ? 'border-primary text-primary' : 'border-border text-muted-foreground',
                )}>
                  <div className={cn(
                    'flex h-20 w-20 items-center justify-center rounded-full border-2 transition-all',
                    running ? 'border-primary/60' : 'border-border',
                  )}>
                    <div className={cn(
                      'h-2 w-2 rounded-full',
                      running ? 'bg-primary glow-primary animate-pulse' : 'bg-muted-foreground',
                    )} />
                  </div>
                </div>
              </div>

              {/* пульсы при кликах */}
              {zonePulses.map((p) => (
                <span
                  key={p.id}
                  className="pointer-events-none absolute h-3 w-3 -translate-x-1/2 -translate-y-1/2 rounded-full bg-primary"
                  style={{
                    left: `${p.x}%`,
                    top: `${p.y}%`,
                    animation: 'ping 0.7s cubic-bezier(0,0,0.2,1) forwards',
                  }}
                />
              ))}

              {/* подсказка */}
              <div className="pointer-events-none absolute inset-x-0 bottom-3 text-center text-xs text-muted-foreground">
                {running ? (
                  <span className="font-mono text-primary">Идёт замер · {cps.cps.toFixed(1)} CPS</span>
                ) : (
                  <span>Нажми «Запустить» справа · цель — центр круга</span>
                )}
              </div>
            </div>
          </CardContent>
        </Card>

        {/* COLLAPSIBLE LOG */}
        <Card>
          <button
            onClick={() => setLogOpen((v) => !v)}
            className="flex w-full items-center justify-between gap-2 px-5 py-3 text-left transition-colors hover:bg-accent/50"
          >
            <span className="flex items-center gap-2 text-sm font-semibold">
              {logOpen ? <ChevronDown className="h-4 w-4" /> : <ChevronRight className="h-4 w-4" />}
              <Activity className="h-4 w-4 text-primary" />
              Лог выполнения
              {log.length > 0 && (
                <span className="rounded-full bg-muted px-2 py-0.5 text-[10px] font-normal text-muted-foreground">
                  {log.length}
                </span>
              )}
            </span>
            {logOpen && log.length > 0 && (
              <span
                role="button"
                onClick={(e) => { e.stopPropagation(); clearLog() }}
                className="rounded-md p-1 text-muted-foreground hover:bg-destructive/10 hover:text-destructive"
                title="Очистить"
              >
                <Trash2 className="h-3.5 w-3.5" />
              </span>
            )}
          </button>
          {logOpen && (
            <ScrollArea className="h-[180px] border-t">
              <div ref={scrollRef} className="px-5 py-3 font-mono text-[11px] leading-relaxed">
                {log.length === 0 ? (
                  <div className="flex h-32 items-center justify-center text-xs text-muted-foreground">
                    (пусто) — запусти что-нибудь
                  </div>
                ) : (
                  log.map((line, i) => (
                    <div key={i} className="border-b border-border/30 py-0.5 text-muted-foreground last:border-b-0">
                      <span className="mr-2 text-primary/60">{line.slice(0, 12)}</span>
                      <span className="text-foreground/80">{line.slice(14)}</span>
                    </div>
                  ))
                )}
              </div>
            </ScrollArea>
          )}
        </Card>
      </div>

      <Card className="self-start">
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <FlaskConical className="h-5 w-5 text-primary" /> Запуск
          </CardTitle>
          <CardDescription>
            Прогон выбранного профиля или цепочки. CPS считается в обоих режимах.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-3">
          <div className="space-y-1.5">
            <label className="text-xs font-medium text-muted-foreground">Цель</label>
            <Select value={target} onValueChange={setTarget}>
              <SelectTrigger><SelectValue /></SelectTrigger>
              <SelectContent>
                {profiles.length > 0 && (
                  <>
                    <div className="px-2 py-1 text-[10px] font-bold uppercase tracking-wider text-muted-foreground">
                      Простые профили
                    </div>
                    {profiles.map((p, i) => (
                      <SelectItem key={`p${i}`} value={`profile:${i}`}>
                        {p.name || `Profile ${i + 1}`}
                      </SelectItem>
                    ))}
                  </>
                )}
                {chains.length > 0 && (
                  <>
                    <div className="mt-1 px-2 py-1 text-[10px] font-bold uppercase tracking-wider text-muted-foreground">
                      Цепочки
                    </div>
                    {chains.map((c, i) => (
                      <SelectItem key={`c${i}`} value={`chain:${i}`}>
                        {c.name || `Chain ${i + 1}`}
                      </SelectItem>
                    ))}
                  </>
                )}
              </SelectContent>
            </Select>
          </div>

          <label className={cn(
            'flex items-center justify-between gap-3 rounded-md border px-3 py-2.5 text-xs transition-colors',
            dryRun ? 'border-primary/40 bg-primary/5' : 'bg-muted/30',
          )}>
            <div className="flex items-start gap-2">
              <ShieldCheck className={cn('mt-0.5 h-4 w-4 shrink-0', dryRun ? 'text-primary' : 'text-muted-foreground')} />
              <div>
                <div className="font-medium text-foreground">Безопасный режим</div>
                <div className="text-muted-foreground">События не отправляются в систему</div>
              </div>
            </div>
            <Switch checked={dryRun} onCheckedChange={setDryRun} />
          </label>

          {running ? (
            <Button variant="destructive" className="w-full" onClick={Stop}>
              <Square className="h-4 w-4" /> Остановить
            </Button>
          ) : (
            <Button variant="neon" className="w-full" onClick={start}>
              <Play className="h-4 w-4" /> Запустить
            </Button>
          )}

          <div className="rounded-md border border-dashed border-primary/30 bg-primary/5 p-3 text-xs">
            <div className="mb-1 font-semibold text-primary">Замер скорости</div>
            <div className="text-muted-foreground">
              Поставь курсор в зону замера и запусти — увидишь реальный CPS и пульс попаданий.
              В безопасном режиме клики никуда не уходят, всё только в UI.
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}

function Stat({ label, value, accent, active }: { label: string; value: string; accent?: boolean; active?: boolean }) {
  return (
    <div className={cn(
      'rounded-lg border bg-card/50 p-3 transition-all',
      accent && 'border-primary/40',
      accent && active && 'glow-primary',
    )}>
      <div className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground">{label}</div>
      <div className={cn(
        'mt-1 font-mono text-2xl font-bold tabular-nums',
        accent && active && 'text-primary text-glow',
        accent && !active && 'text-primary/70',
      )}>
        {value}
      </div>
    </div>
  )
}

function BarChart({ data }: { data: number[] }) {
  const max = Math.max(1, ...data)
  return (
    <div className="relative flex h-[60px] items-end gap-[2px] overflow-hidden rounded-md border border-border bg-card/30 px-2 py-1.5">
      {data.length === 0 ? (
        <div className="absolute inset-0 flex items-center justify-center text-[10px] text-muted-foreground">
          ожидаю данные...
        </div>
      ) : (
        Array.from({ length: 80 }).map((_, i) => {
          const idx = data.length - 80 + i
          const v = idx >= 0 ? data[idx] : 0
          const h = max > 0 ? (v / max) * 100 : 0
          const recent = i >= 76
          return (
            <div
              key={i}
              className={cn(
                'flex-1 rounded-sm transition-all',
                v === 0 ? 'bg-muted/30' : recent ? 'bg-primary glow-primary' : 'bg-primary/50',
              )}
              style={{ height: `${Math.max(v === 0 ? 6 : 8, h)}%` }}
            />
          )
        })
      )}
    </div>
  )
}
