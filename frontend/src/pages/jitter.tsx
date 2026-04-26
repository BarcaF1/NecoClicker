import { useState } from 'react'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Shuffle, Play, Square, ShieldCheck, Sparkles } from 'lucide-react'
import { useConfig } from '@/hooks/use-config'
import { useEngine } from '@/components/engine-provider'
import { StartProfileLimited, Stop, ResetClicks } from '../../wailsjs/go/main/App'
import { macro } from '../../wailsjs/go/models'
import { cn } from '@/lib/utils'

export function JitterPage() {
  const { cfg } = useConfig()
  const { running, cps } = useEngine()
  const profiles = cfg?.profiles ?? []

  const [profileIdx, setProfileIdx] = useState('0')
  const [jitter, setJitter] = useState('30')
  const [dryRun, setDryRun] = useState(false)
  const [withTimer, setWithTimer] = useState(false)
  const [duration, setDuration] = useState('60')

  const start = async () => {
    const idx = parseInt(profileIdx)
    if (isNaN(idx) || !profiles[idx]) return
    await ResetClicks()
    const lim = new macro.RunLimits({
      jitter_ms: Math.max(0, parseFloat(jitter) || 0),
      duration_sec: withTimer ? Math.max(1, parseInt(duration) || 0) : 0,
      max_clicks: 0,
    })
    await StartProfileLimited(idx, lim, dryRun)
  }

  const cur = profiles[parseInt(profileIdx)]
  const baseMs = cur?.interval_ms ?? 0
  const jit = parseFloat(jitter) || 0
  const minMs = Math.max(0, baseMs - jit / 2)
  const maxMs = baseMs + jit / 2

  return (
    <div className="grid gap-6 lg:grid-cols-3">
      <Card className="lg:col-span-2">
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Shuffle className="h-5 w-5 text-primary" /> Хаотичный кликер (jitter)
          </CardTitle>
          <CardDescription>
            Каждый клик получает случайное смещение интервала ±N мс — клики не выглядят "идеально равномерными".
            Полезно против античитов, которые палят ботов по правильному ритму.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-5">
          <div className="space-y-1.5">
            <Label>Профиль (база)</Label>
            <Select value={profileIdx} onValueChange={setProfileIdx}>
              <SelectTrigger><SelectValue /></SelectTrigger>
              <SelectContent>
                {profiles.map((p, i) => (
                  <SelectItem key={i} value={String(i)}>{p.name || `Profile ${i + 1}`}</SelectItem>
                ))}
              </SelectContent>
            </Select>
            {cur && (
              <p className="text-[11px] text-muted-foreground">
                {cur.button} · интервал {cur.interval_ms} мс
              </p>
            )}
          </div>

          <div className="space-y-1.5">
            <Label htmlFor="jit">Размах jitter, мс (±N/2)</Label>
            <Input
              id="jit"
              type="number"
              min="0"
              step="1"
              value={jitter}
              onChange={(e) => setJitter(e.target.value)}
            />
            <div className="rounded-md border bg-muted/20 p-3 text-xs">
              <div className="mb-1 flex items-center gap-1.5 text-muted-foreground">
                <Sparkles className="h-3.5 w-3.5" /> Превью разброса
              </div>
              <div className="grid grid-cols-3 gap-2 font-mono">
                <Range label="мин" value={minMs.toFixed(1) + ' мс'} />
                <Range label="база" value={baseMs.toFixed(1) + ' мс'} accent />
                <Range label="макс" value={maxMs.toFixed(1) + ' мс'} />
              </div>
            </div>
          </div>

          <div className="space-y-2">
            <label className={cn(
              'flex items-center justify-between gap-3 rounded-md border px-3 py-2.5 text-xs transition-colors',
              withTimer ? 'border-primary/40 bg-primary/5' : 'bg-muted/30',
            )}>
              <div>
                <div className="font-medium text-foreground">Автостоп по времени</div>
                <div className="text-muted-foreground">Остановиться через N секунд</div>
              </div>
              <Switch checked={withTimer} onCheckedChange={setWithTimer} />
            </label>
            {withTimer && (
              <Input
                type="number"
                min="1"
                value={duration}
                onChange={(e) => setDuration(e.target.value)}
                placeholder="секунд"
              />
            )}

            <label className={cn(
              'flex items-center justify-between gap-3 rounded-md border px-3 py-2.5 text-xs transition-colors',
              dryRun ? 'border-primary/40 bg-primary/5' : 'bg-muted/30',
            )}>
              <div className="flex items-start gap-2">
                <ShieldCheck className={cn('mt-0.5 h-4 w-4 shrink-0', dryRun ? 'text-primary' : 'text-muted-foreground')} />
                <div>
                  <div className="font-medium text-foreground">Безопасный режим</div>
                  <div className="text-muted-foreground">Сухой прогон без отправки кликов</div>
                </div>
              </div>
              <Switch checked={dryRun} onCheckedChange={setDryRun} />
            </label>
          </div>
        </CardContent>
      </Card>

      <Card className="self-start">
        <CardHeader>
          <CardTitle>Запуск</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          <div className="grid grid-cols-2 gap-2">
            <SmallStat label="CPS" value={cps.cps.toFixed(1)} />
            <SmallStat label="Всего" value={Number(cps.total).toLocaleString()} />
          </div>

          {running ? (
            <Button variant="destructive" className="w-full" onClick={Stop}>
              <Square className="h-4 w-4" /> Остановить
            </Button>
          ) : (
            <Button variant="neon" className="w-full" onClick={start} disabled={profiles.length === 0}>
              <Play className="h-4 w-4" /> Запустить с jitter'ом
            </Button>
          )}

          <div className="rounded-md border border-dashed border-primary/30 bg-primary/5 p-3 text-[11px]">
            <div className="mb-1 font-semibold text-primary">Как это работает</div>
            <div className="text-muted-foreground">
              Перед каждым кликом движок берёт случайное число из <span className="font-mono">[-J/2; +J/2]</span> мс
              и прибавляет к базовому интервалу профиля. Распределение равномерное.
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}

function Range({ label, value, accent }: { label: string; value: string; accent?: boolean }) {
  return (
    <div className={cn('rounded border bg-card/50 px-2 py-1.5 text-center', accent && 'border-primary/40')}>
      <div className="text-[9px] uppercase text-muted-foreground">{label}</div>
      <div className={cn('font-mono text-xs font-bold', accent && 'text-primary')}>{value}</div>
    </div>
  )
}

function SmallStat({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-md border bg-card/50 p-2 text-center">
      <div className="text-[9px] font-semibold uppercase tracking-wider text-muted-foreground">{label}</div>
      <div className="mt-0.5 font-mono text-base font-bold tabular-nums">{value}</div>
    </div>
  )
}
