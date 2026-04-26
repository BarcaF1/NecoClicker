import * as React from 'react'
import { Button } from '@/components/ui/button'
import { Keyboard, X, Loader2 } from 'lucide-react'
import { RecordHotkey } from '../../wailsjs/go/main/App'
import { cn } from '@/lib/utils'

type Props = {
  value: string
  onChange: (v: string) => void
  placeholder?: string
  className?: string
}

/**
 * HotkeyRecorder — кнопка, которая запускает глобальный захват следующего
 * нажатия (любая клавиша / Mouse4 / Mouse5 + модификаторы Ctrl/Alt/Shift/Win)
 * через backend hotkey.RecordOnce. Без захвата на стороне браузера —
 * это даёт надёжный capture даже когда фокус не в окне.
 */
export function HotkeyRecorder({ value, onChange, placeholder = '— не задан —', className }: Props) {
  const [recording, setRecording] = React.useState(false)
  const [error, setError] = React.useState<string | null>(null)

  const startRecord = async () => {
    setRecording(true)
    setError(null)
    try {
      const got = await RecordHotkey(8000)
      if (got) onChange(got)
    } catch (e: any) {
      setError('Таймаут — нажатие не зафиксировано')
    } finally {
      setRecording(false)
    }
  }

  const clear = () => onChange('')

  return (
    <div className={cn('flex items-center gap-2', className)}>
      <div className={cn(
        'flex h-9 flex-1 items-center gap-2 rounded-md border bg-background px-3 text-sm',
        recording && 'animate-pulse border-primary bg-primary/10 glow-primary',
      )}>
        <Keyboard className={cn('h-3.5 w-3.5 shrink-0', recording ? 'text-primary' : 'text-muted-foreground')} />
        {recording ? (
          <span className="font-mono text-primary">Жми клавишу или Mouse4/Mouse5...</span>
        ) : value ? (
          <span className="font-mono">{value}</span>
        ) : (
          <span className="text-muted-foreground">{placeholder}</span>
        )}
      </div>
      {recording ? (
        <Button size="icon" variant="outline" disabled className="shrink-0">
          <Loader2 className="h-4 w-4 animate-spin" />
        </Button>
      ) : (
        <Button size="sm" variant="outline" onClick={startRecord} className="shrink-0">
          <Keyboard className="h-3.5 w-3.5" /> Записать
        </Button>
      )}
      {value && !recording && (
        <Button size="icon" variant="ghost" onClick={clear} className="h-9 w-9 shrink-0" title="Очистить">
          <X className="h-3.5 w-3.5" />
        </Button>
      )}
      {error && <span className="text-[11px] text-destructive">{error}</span>}
    </div>
  )
}
