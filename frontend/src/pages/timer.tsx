import { useState } from 'react'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { TimerIcon, Play, Square, ShieldCheck, Hash, Hourglass } from 'lucide-react'
import { useConfig } from '@/components/config-provider'
import { useEngine } from '@/components/engine-provider'
import { StartProfileLimited, Stop, ResetClicks } from '../../wailsjs/go/main/App'
import { macro } from '../../wailsjs/go/models'
import { cn } from '@/lib/utils'

type Mode = 'duration' | 'clicks' | 'both'

export function TimerPage() {
  const { cfg } = useConfig()
  const { running, cps } = useEngine()
  const profiles = cfg?.profiles ?? []

  const [profileIdx, setProfileIdx] = useState('0')
  const [mode, setMode] = useState<Mode>('duration')
  const [duration, setDuration] = useState('60')
  const [clicks, setClicks] = useState('100')
  const [dryRun, setDryRun] = useState(false)

  const start = async () => {
    const idx = parseInt(profileIdx)
    if (isNaN(idx) || !profiles[idx]) return
    await ResetClicks()
    const lim = new macro.RunLimits({
      duration_sec: mode === 'duration' || mode === 'both' ? Math.max(1, parseInt(duration) || 0) : 0,
      max_clicks: mode === 'clicks' || mode === 'both' ? Math.max(1, parseInt(clicks) || 0) : 0,
      jitter_ms: 0,
    })
    await StartProfileLimited(idx, lim, dryRun)
  }

  const totalClicks = Number(cps.total)
  const maxClicksGoal = mode !== 'duration' ? Math.max(1, parseInt(clicks) || 0) : 0
  const progress = maxClicksGoal > 0 ? Math.min(100, (totalClicks / maxClicksGoal) * 100) : 0

  return (
    <div className="grid gap-6 lg:grid-cols-3">
      <Card className="lg:col-span-2">
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <TimerIcon className="h-5 w-5 text-primary" /> Кликер с таймером
          </CardTitle>
          <CardDescription>
            Запусти профиль с автостопом — по времени, числу кликов, или обоим.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-5">
          <div className="space-y-1.5">
            <Label>Профиль</Label>
            <Select value={profileIdx} onValueChange={setProfileIdx}>
              <SelectTrigger><SelectValue /></SelectTrigger>
              <SelectContent>
                {profiles.map((p, i) => (
                  <SelectItem key={i} value={String(i)}>{p.name || `Profile ${i + 1}`}</SelectItem>
                ))}
              </SelectContent>
            </Select>
            {profiles[parseInt(profileIdx)] && (
              <p className="text-[11px] text-muted-foreground">
                {profiles[parseInt(profileIdx)].button} · интервал {profiles[parseInt(profileIdx)].interval_ms} мс
              </p>
            )}
          </div>

          <div>
            <Label className="mb-2 block">Условие остановки</Label>
            <div className="grid grid-cols-3 gap-2">
              <ModeCard active={mode === 'duration'} onClick={() => setMode('duration')} icon={Hourglass} label="По времени" />
              <ModeCard active={mode === 'clicks'} onClick={() => setMode('clicks')} icon={Hash} label="По кликам" />
              <ModeCard active={mode === 'both'} onClick={() => setMode('both')} icon={TimerIcon} label="Оба (что раньше)" />
            </div>
          </div>

          <div className="grid gap-4 sm:grid-cols-2">
            {(mode === 'duration' || mode === 'both') && (
              <div className="space-y-1.5">
                <Label htmlFor="dur">Длительность (секунд)</Label>
                <Input id="dur" type="number" min="1" value={duration} onChange={(e) => setDuration(e.target.value)} />
                <p className="text-[11px] text-muted-foreground">
                  ≈ {Math.floor((parseInt(duration) || 0) / 60)}м {(parseInt(duration) || 0) % 60}с
                </p>
              </div>
            )}
            {(mode === 'clicks' || mode === 'both') && (
              <div className="space-y-1.5">
                <Label htmlFor="cl">Кол-во кликов</Label>
                <Input id="cl" type="number" min="1" value={clicks} onChange={(e) => setClicks(e.target.value)} />
              </div>
            )}
          </div>

          <label className={cn(
            'flex items-center justify-between gap-3 rounded-md border px-3 py-2.5 text-xs transition-colors',
            dryRun ? 'border-primary/40 bg-primary/5' : 'bg-muted/30',
          )}>
            <div className="flex items-start gap-2">
              <ShieldCheck className={cn('mt-0.5 h-4 w-4 shrink-0', dryRun ? 'text-primary' : 'text-muted-foreground')} />
              <div>
                <div className="font-medium text-foreground">Безопасный режим</div>
                <div className="text-muted-foreground">Сухой прогон без отправки кликов в систему</div>
              </div>
            </div>
            <Switch checked={dryRun} onCheckedChange={setDryRun} />
          </label>
        </CardContent>
      </Card>

      <Card className="self-start">
        <CardHeader>
          <CardTitle>Статус</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          {(mode === 'clicks' || mode === 'both') && running && (
            <div>
              <div className="mb-1 flex items-center justify-between text-xs">
                <span className="text-muted-foreground">Прогресс кликов</span>
                <span className="font-mono">{totalClicks} / {maxClicksGoal}</span>
              </div>
              <div className="h-2 overflow-hidden rounded-full bg-muted">
                <div
                  className="h-full bg-primary transition-all"
                  style={{ width: `${progress}%` }}
                />
              </div>
            </div>
          )}
          <div className="grid grid-cols-2 gap-2">
            <SmallStat label="CPS" value={cps.cps.toFixed(1)} />
            <SmallStat label="Всего" value={totalClicks.toLocaleString()} />
          </div>

          {running ? (
            <Button variant="destructive" className="w-full" onClick={Stop}>
              <Square className="h-4 w-4" /> Остановить
            </Button>
          ) : (
            <Button variant="neon" className="w-full" onClick={start} disabled={profiles.length === 0}>
              <Play className="h-4 w-4" /> Запустить
            </Button>
          )}

          <div className="rounded-md border border-dashed bg-muted/20 p-3 text-[11px] text-muted-foreground">
            <b className="text-foreground">Таймер</b> следит за условиями и останавливает движок автоматически.
            Можно использовать вместе с глобальными хоткеями активного профиля.
          </div>
        </CardContent>
      </Card>
    </div>
  )
}

function ModeCard({ active, onClick, icon: Icon, label }: { active: boolean; onClick: () => void; icon: any; label: string }) {
  return (
    <button
      onClick={onClick}
      className={cn(
        'flex flex-col items-center gap-1 rounded-lg border p-3 text-xs transition-all',
        active
          ? 'border-primary bg-primary/10 text-primary glow-primary'
          : 'border-border hover:border-primary/40 hover:bg-accent',
      )}
    >
      <Icon className="h-4 w-4" />
      {label}
    </button>
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
