import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { ThemePicker } from '@/components/theme-picker'
import { Palette, Sparkles } from 'lucide-react'

export function ThemesPage() {
  return (
    <div className="space-y-6">
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Palette className="h-5 w-5 text-primary" /> Темы
          </CardTitle>
          <CardDescription>
            Все цвета приложения построены на CSS-переменных HSL — переключение мгновенное и применяется ко всему UI.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <ThemePicker />
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Sparkles className="h-5 w-5 text-primary" /> Демо элементов
          </CardTitle>
          <CardDescription>Превью основных компонентов в выбранной теме.</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex flex-wrap gap-2">
            <div className="rounded-md bg-primary px-4 py-2 text-sm text-primary-foreground glow-primary">primary</div>
            <div className="rounded-md bg-secondary px-4 py-2 text-sm text-secondary-foreground">secondary</div>
            <div className="rounded-md bg-accent px-4 py-2 text-sm text-accent-foreground">accent</div>
            <div className="rounded-md bg-muted px-4 py-2 text-sm text-muted-foreground">muted</div>
            <div className="rounded-md bg-destructive px-4 py-2 text-sm text-destructive-foreground">destructive</div>
          </div>
          <div className="rounded-lg border border-primary/40 bg-primary/5 p-4 text-glow">
            Текст с свечением (text-glow) — заметно в неоновых темах.
          </div>
          <div className="neon-border rounded-lg p-4 text-sm">
            Блок с неоновой обводкой и тенью под цвет primary.
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
