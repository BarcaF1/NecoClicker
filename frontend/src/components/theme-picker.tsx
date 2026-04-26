import { useTheme, THEMES } from './theme-provider'
import { SetTheme } from '../../wailsjs/go/main/App'
import { Check } from 'lucide-react'
import { cn } from '@/lib/utils'

export function ThemePicker() {
  const { theme, setTheme } = useTheme()

  const apply = (id: typeof THEMES[number]['id']) => {
    setTheme(id)
    SetTheme(id).catch(() => {})
  }

  return (
    <div className="grid grid-cols-2 gap-2 sm:grid-cols-3 md:grid-cols-5">
      {THEMES.map((t) => (
        <button
          key={t.id}
          onClick={() => apply(t.id)}
          className={cn(
            'group relative flex items-center gap-3 rounded-lg border bg-card p-3 transition-all hover:border-primary hover:shadow-md',
            theme === t.id && 'border-primary ring-2 ring-primary/40 glow-primary',
          )}
        >
          <div className="relative h-8 w-8 shrink-0 overflow-hidden rounded-md border border-border">
            <div className="absolute inset-0" style={{ background: t.swatch[0] }} />
            <div className="absolute inset-y-0 right-0 w-1/2" style={{ background: t.swatch[1] }} />
            {t.neon && (
              <div className="absolute inset-0 rounded-md ring-1 ring-inset" style={{ boxShadow: `inset 0 0 8px ${t.swatch[1]}` }} />
            )}
          </div>
          <span className="text-xs font-medium">{t.label}</span>
          {theme === t.id && <Check className="absolute right-2 top-2 h-3 w-3 text-primary" />}
        </button>
      ))}
    </div>
  )
}
