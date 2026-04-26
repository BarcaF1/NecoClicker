import { useEffect, useMemo, useState } from 'react'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import {
  Crosshair, MousePointerClick, Play, Square, Save, Keyboard,
  Plus, Trash2, Star, StarOff,
} from 'lucide-react'
import { useConfig } from '@/hooks/use-config'
import { useEngine } from '@/components/engine-provider'
import { useConfirm } from '@/components/confirm-dialog'
import { HotkeyRecorder } from '@/components/hotkey-recorder'
import { CursorPos, StartProfile, Stop } from '../../wailsjs/go/main/App'
import { macro } from '../../wailsjs/go/models'
import { cn } from '@/lib/utils'

type ButtonId = 'left' | 'right' | 'middle' | 'x1' | 'x2'
const BUTTON_OPTIONS: { id: ButtonId; label: string }[] = [
  { id: 'left',   label: 'Левая (ЛКМ)' },
  { id: 'right',  label: 'Правая (ПКМ)' },
  { id: 'middle', label: 'Средняя (СКМ)' },
  { id: 'x1',     label: 'X1 — Mouse4 (вперёд)' },
  { id: 'x2',     label: 'X2 — Mouse5 (назад)' },
]

export function HomePage() {
  const { cfg, saveProfile, deleteProfile, setActiveProfile } = useConfig()
  const { running } = useEngine()
  const { ask, alert: showAlert } = useConfirm()

  const profiles = cfg?.profiles ?? []
  const activeIdx = cfg?.active ?? 0
  const [selected, setSelected] = useState(activeIdx)

  // sync selected with active when cfg first loads
  useEffect(() => {
    if (selected >= profiles.length) setSelected(Math.max(0, profiles.length - 1))
  }, [profiles.length])

  const cur = profiles[selected]

  // form state — editing the selected profile
  const [name, setName] = useState('')
  const [button, setButton] = useState<ButtonId>('left')
  const [intervalMs, setIntervalMs] = useState('100')
  const [useCurrent, setUseCurrent] = useState(true)
  const [x, setX] = useState('0')
  const [y, setY] = useState('0')
  const [hotkey, setHotkey] = useState('F6')
  const [dirty, setDirty] = useState(false)

  // when the selected profile changes, repopulate form
  useEffect(() => {
    if (!cur) return
    setName(cur.name ?? '')
    setButton((cur.button as ButtonId) || 'left')
    setIntervalMs(String(cur.interval_ms ?? 100))
    setUseCurrent(!!cur.use_current)
    setX(String(cur.x ?? 0))
    setY(String(cur.y ?? 0))
    setHotkey(cur.hotkey ?? '')
    setDirty(false)
  }, [selected, cfg])

  const buildProfile = (): macro.SimpleConfig => {
    const ms = parseFloat(intervalMs.replace(',', '.'))
    return new macro.SimpleConfig({
      name: name || `Profile ${selected + 1}`,
      button,
      interval_ms: Number.isFinite(ms) && ms >= 0 ? ms : 100,
      use_current: useCurrent,
      x: parseInt(x) || 0,
      y: parseInt(y) || 0,
      hotkey,
    })
  }

  const captureCursor = async () => {
    const [cx, cy] = await CursorPos()
    setX(String(cx))
    setY(String(cy))
    setDirty(true)
  }

  const onSave = async () => {
    await saveProfile(selected, buildProfile())
    setDirty(false)
  }

  const onStart = async () => {
    if (dirty) await onSave()
    await StartProfile(selected)
  }

  const onAdd = async () => {
    const next = new macro.SimpleConfig({
      name: `Profile ${profiles.length + 1}`,
      button: 'left',
      interval_ms: 100.0,
      use_current: true,
      x: 0, y: 0, hotkey: '',
    })
    const idx = await saveProfile(-1, next)
    setSelected(idx)
  }

  const onDelete = async () => {
    if (profiles.length <= 1) {
      await showAlert('Невозможно удалить', 'Должен оставаться хотя бы один профиль.')
      return
    }
    const ok = await ask({
      title: 'Удалить профиль?',
      description: `Профиль "${cur?.name}" будет безвозвратно удалён.`,
      confirmText: 'Удалить',
      destructive: true,
    })
    if (!ok) return
    await deleteProfile(selected)
    setSelected(Math.max(0, selected - 1))
  }

  const onMakeActive = async () => {
    await setActiveProfile(selected)
  }

  const isActive = selected === activeIdx

  if (!cfg) return null

  return (
    <div className="grid gap-6 lg:grid-cols-3">
      <Card className="lg:col-span-2">
        <CardHeader className="space-y-3 pb-3">
          <div className="flex items-center justify-between gap-3">
            <CardTitle className="flex items-center gap-2">
              <MousePointerClick className="h-5 w-5 text-primary" />
              Single click
            </CardTitle>
            <Button size="sm" variant="outline" onClick={onAdd}>
              <Plus className="h-4 w-4" /> Новый профиль
            </Button>
          </div>

          <div className="flex flex-wrap items-center gap-2">
            {profiles.map((p, i) => (
              <button
                key={i}
                onClick={() => setSelected(i)}
                className={cn(
                  'group inline-flex items-center gap-1.5 rounded-full border px-3 py-1 text-xs font-medium transition-all',
                  selected === i
                    ? 'border-primary bg-primary/10 text-primary'
                    : 'border-border hover:border-primary/40 hover:bg-accent',
                )}
                title={p.hotkey ? `Хоткей: ${p.hotkey}` : undefined}
              >
                {i === activeIdx && <Star className="h-3 w-3 fill-current" />}
                <span>{p.name || `Profile ${i + 1}`}</span>
                {p.hotkey && <span className="text-muted-foreground">· {p.hotkey}</span>}
              </button>
            ))}
          </div>
        </CardHeader>

        <CardContent className="space-y-5">
          <div className="grid gap-4 sm:grid-cols-2">
            <div className="space-y-1.5">
              <Label htmlFor="pname">Имя профиля</Label>
              <Input id="pname" value={name} onChange={(e) => { setName(e.target.value); setDirty(true) }} placeholder="например, AFK farm" />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="btn">Кнопка мыши</Label>
              <Select value={button} onValueChange={(v) => { setButton(v as ButtonId); setDirty(true) }}>
                <SelectTrigger id="btn"><SelectValue /></SelectTrigger>
                <SelectContent>
                  {BUTTON_OPTIONS.map((b) => (
                    <SelectItem key={b.id} value={b.id}>{b.label}</SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="ms">Интервал, мс</Label>
              <Input
                id="ms"
                type="number"
                min="0"
                step="0.1"
                value={intervalMs}
                onChange={(e) => { setIntervalMs(e.target.value); setDirty(true) }}
                placeholder="100"
              />
              {(() => {
                const v = parseFloat(intervalMs.replace(',', '.'))
                if (!Number.isFinite(v)) return null
                if (v === 0) return (
                  <p className="text-[11px] text-destructive">⚠ Максимальная скорость: будет грузить 100% одного ядра CPU и слать события без пауз. ОС/игра могут отбрасывать "слишком быстрые" клики.</p>
                )
                if (v > 0 && v < 1) return (
                  <p className="text-[11px] text-primary/80">ℹ Sub-ms интервал ({v}мс ≈ {Math.round(1000 / v)} CPS теоретически). Точность ограничена scheduler-tick'ом ОС (~1мс).</p>
                )
                return null
              })()}
            </div>
            <div className="space-y-1.5">
              <Label className="flex items-center gap-1.5">
                <Keyboard className="h-3.5 w-3.5" /> Хоткей пуск/стоп
              </Label>
              <HotkeyRecorder
                value={hotkey}
                onChange={(v) => { setHotkey(v); setDirty(true) }}
                placeholder="нажми «Записать» — потом клавишу/Mouse4-5"
              />
            </div>
          </div>

          <div className="rounded-lg border border-border bg-muted/30 p-4">
            <div className="mb-3 flex items-center justify-between gap-3">
              <div>
                <div className="text-sm font-medium">По текущей позиции курсора</div>
                <div className="text-xs text-muted-foreground">Игнорировать X/Y и кликать там, где сейчас курсор.</div>
              </div>
              <Switch checked={useCurrent} onCheckedChange={(v) => { setUseCurrent(v); setDirty(true) }} />
            </div>
            <div className={useCurrent ? 'opacity-40 pointer-events-none' : ''}>
              <div className="grid gap-3 sm:grid-cols-3">
                <div className="space-y-1.5">
                  <Label htmlFor="x">X</Label>
                  <Input id="x" type="number" value={x} onChange={(e) => { setX(e.target.value); setDirty(true) }} />
                </div>
                <div className="space-y-1.5">
                  <Label htmlFor="y">Y</Label>
                  <Input id="y" type="number" value={y} onChange={(e) => { setY(e.target.value); setDirty(true) }} />
                </div>
                <div className="flex items-end">
                  <Button variant="outline" onClick={captureCursor} className="w-full">
                    <Crosshair className="h-4 w-4" /> Захватить
                  </Button>
                </div>
              </div>
            </div>
          </div>

          <p className="text-xs text-muted-foreground">
            Хоткей работает только у <b>активного</b> профиля (отмечен ⭐). Чтобы переключить — выбери пресс и нажми "Сделать активным".
            <br />
            Допустимые модификаторы: Ctrl, Alt, Shift, Win. Клавиши: A–Z, 0–9, F1–F24, Space, Enter, Tab, Esc, Insert, Delete, Home, End, PgUp, PgDn, стрелки.
          </p>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Управление</CardTitle>
          <CardDescription>
            {cur ? <>Профиль: <b className="text-foreground">{cur.name}</b>{isActive && ' · ⭐ активный'}</> : 'Нет профиля'}
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-3">
          {running ? (
            <Button variant="destructive" size="lg" className="w-full" onClick={Stop}>
              <Square className="h-4 w-4" /> Остановить
            </Button>
          ) : (
            <Button variant="neon" size="lg" className="w-full" onClick={onStart}>
              <Play className="h-4 w-4" /> Запустить{dirty ? ' и сохранить' : ''}
            </Button>
          )}
          <Button variant="outline" className="w-full" onClick={onSave} disabled={!dirty}>
            <Save className="h-4 w-4" /> {dirty ? 'Сохранить' : 'Сохранено'}
          </Button>

          <div className="grid grid-cols-2 gap-2 pt-1">
            {isActive ? (
              <Button variant="ghost" size="sm" className="col-span-2 text-muted-foreground" disabled>
                <Star className="h-3.5 w-3.5 fill-current" /> Уже активный
              </Button>
            ) : (
              <Button variant="ghost" size="sm" className="col-span-2" onClick={onMakeActive}>
                <StarOff className="h-3.5 w-3.5" /> Сделать активным
              </Button>
            )}
            <Button variant="ghost" size="sm" className="col-span-2 text-destructive hover:bg-destructive/10" onClick={onDelete} disabled={profiles.length <= 1}>
              <Trash2 className="h-3.5 w-3.5" /> Удалить профиль
            </Button>
          </div>

          <div className="rounded-md border border-dashed border-border bg-muted/20 p-3 text-xs text-muted-foreground">
            <div className="mb-1 font-medium text-foreground">Подсказка</div>
            Глобальный хоткей привязан только к <b>активному</b> профилю. Создай несколько пресетов под разные задачи и переключай ⭐.
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
