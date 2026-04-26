import * as React from 'react'

export type ThemeId =
  | 'light'
  | 'dark'
  | 'enemy-dark'
  | 'purple-neon'
  | 'green-neon'
  | 'vampire'
  | 'red-neon'
  | 'cyan-neon'
  | 'pink-neon'
  | 'orange-neon'
  | 'yellow-neon'
  | 'blue-neon'

export const THEMES: { id: ThemeId; label: string; swatch: [string, string]; neon?: boolean }[] = [
  { id: 'light',        label: 'Light',        swatch: ['#ffffff', '#0f172a'] },
  { id: 'dark',         label: 'Dark',         swatch: ['#0b1220', '#f8fafc'] },
  { id: 'enemy-dark',   label: 'Enemy Dark',   swatch: ['#04040a', '#e8eaed'] },
  { id: 'purple-neon',  label: 'Purple neon',  swatch: ['#0e0517', '#bb29ff'], neon: true },
  { id: 'pink-neon',    label: 'Pink neon',    swatch: ['#170614', '#ff4dc7'], neon: true },
  { id: 'red-neon',     label: 'Red neon',     swatch: ['#170404', '#ff2929'], neon: true },
  { id: 'vampire',      label: 'Vampire',      swatch: ['#190509', '#ff1f3a'], neon: true },
  { id: 'orange-neon',  label: 'Orange neon',  swatch: ['#170c03', '#ff7a1f'], neon: true },
  { id: 'yellow-neon',  label: 'Yellow neon',  swatch: ['#16140a', '#fff21f'], neon: true },
  { id: 'green-neon',   label: 'Green neon',   swatch: ['#02110a', '#1bff7a'], neon: true },
  { id: 'cyan-neon',    label: 'Cyan neon',    swatch: ['#03171b', '#1bf0ff'], neon: true },
  { id: 'blue-neon',    label: 'Blue neon',    swatch: ['#04081a', '#3d6bff'], neon: true },
]

export function detectSystemTheme(): ThemeId {
  if (typeof window === 'undefined' || !window.matchMedia) return 'light'
  return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
}

type Ctx = { theme: ThemeId; setTheme: (t: ThemeId) => void }
const ThemeContext = React.createContext<Ctx>({ theme: 'light', setTheme: () => {} })

export function ThemeProvider({ children, initial = 'light' as ThemeId }: { children: React.ReactNode; initial?: ThemeId }) {
  const [theme, setThemeState] = React.useState<ThemeId>(initial)

  React.useEffect(() => {
    const root = document.documentElement
    root.setAttribute('data-theme', theme)
  }, [theme])

  const setTheme = React.useCallback((t: ThemeId) => setThemeState(t), [])
  return <ThemeContext.Provider value={{ theme, setTheme }}>{children}</ThemeContext.Provider>
}

export function useTheme() {
  return React.useContext(ThemeContext)
}
