import { useEffect, useState } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { ConfigPath } from '../../wailsjs/go/main/App'
import { BrowserOpenURL } from '../../wailsjs/runtime/runtime'
import { Folder, Keyboard, Info, Github, User, ExternalLink } from 'lucide-react'
import necoUrl from '@/assets/neco.png'

const GITHUB_URL = 'https://github.com/BarcaF1/NecoClicker'
const AUTHOR = 'allbanned'

export function SettingsPage() {
  const [path, setPath] = useState('')
  useEffect(() => { ConfigPath().then(setPath) }, [])

  return (
    <div className="space-y-6">
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Folder className="h-5 w-5 text-primary" /> Конфигурация
          </CardTitle>
          <CardDescription>Все настройки и цепочки хранятся в одном JSON-файле.</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="rounded-md border bg-muted/30 p-3 font-mono text-xs break-all">{path || '...'}</div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Keyboard className="h-5 w-5 text-primary" /> Глобальные хоткеи
          </CardTitle>
          <CardDescription>Работают даже когда окно свёрнуто.</CardDescription>
        </CardHeader>
        <CardContent className="space-y-2 text-sm">
          <div className="grid gap-2 text-xs sm:grid-cols-2">
            <Section title="Модификаторы">Ctrl, Alt, Shift, Win</Section>
            <Section title="Буквы / цифры">A–Z, 0–9</Section>
            <Section title="Функциональные">F1–F24</Section>
            <Section title="Спец.">Space, Enter, Tab, Esc</Section>
            <Section title="Навигация">Insert, Delete, Home, End, PgUp, PgDn</Section>
            <Section title="Стрелки">Up, Down, Left, Right</Section>
          </div>
          <div className="mt-3 rounded-md border border-dashed bg-muted/20 p-3 text-xs text-muted-foreground">
            <b className="text-foreground">Пример:</b> <code className="rounded bg-background px-1 py-0.5 font-mono">Ctrl+Shift+F1</code>,&nbsp;
            <code className="rounded bg-background px-1 py-0.5 font-mono">Alt+Q</code>,&nbsp;
            <code className="rounded bg-background px-1 py-0.5 font-mono">F6</code>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Info className="h-5 w-5 text-primary" /> О приложении
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex items-center gap-4">
            <img src={necoUrl} alt="" className="h-20 w-20 rounded-xl border border-border object-cover" />
            <div className="space-y-1 text-sm">
              <div className="text-base font-semibold">NecoClicker <span className="text-muted-foreground">v1.2</span></div>
              <div className="text-xs text-muted-foreground">
                Лёгкий автокликер с глобальными хоткеями и редактором макросов.<br />
                Go · Wails · React · Tailwind.
              </div>
            </div>
          </div>

          <div className="grid gap-2 sm:grid-cols-2">
            <div className="rounded-md border bg-card/50 p-3">
              <div className="mb-1 flex items-center gap-1.5 text-[10px] font-semibold uppercase tracking-wider text-muted-foreground">
                <User className="h-3 w-3" /> Автор
              </div>
              <div className="font-mono text-sm">{AUTHOR}</div>
              <div className="mt-1 text-[10px] text-muted-foreground italic">соцсети — позже</div>
            </div>

            <button
              onClick={() => BrowserOpenURL(GITHUB_URL)}
              className="group rounded-md border bg-card/50 p-3 text-left transition-colors hover:border-primary/50 hover:bg-primary/5"
            >
              <div className="mb-1 flex items-center gap-1.5 text-[10px] font-semibold uppercase tracking-wider text-muted-foreground">
                <Github className="h-3 w-3" /> GitHub
                <ExternalLink className="ml-auto h-3 w-3 opacity-0 transition-opacity group-hover:opacity-100" />
              </div>
              <div className="break-all font-mono text-xs group-hover:text-primary">{GITHUB_URL.replace('https://', '')}</div>
            </button>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div className="rounded-md border bg-card/50 p-2.5">
      <div className="mb-1 text-[10px] font-semibold uppercase tracking-wider text-muted-foreground">{title}</div>
      <div className="font-mono text-foreground">{children}</div>
    </div>
  )
}
