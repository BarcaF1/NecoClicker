import * as React from 'react'
import {
  GetConfig, SaveProfile, DeleteProfile, SetActiveProfile,
  SaveChain, DeleteChain,
} from '../../wailsjs/go/main/App'
import type { macro } from '../../wailsjs/go/models'

type Ctx = {
  cfg: macro.Config | null
  reload: () => Promise<void>
  saveProfile: (idx: number, p: macro.SimpleConfig) => Promise<number>
  deleteProfile: (idx: number) => Promise<void>
  setActiveProfile: (idx: number) => Promise<void>
  saveChain: (idx: number, ch: macro.Chain) => Promise<void>
  deleteChain: (idx: number) => Promise<void>
}

const ConfigCtx = React.createContext<Ctx>({
  cfg: null,
  reload: async () => {},
  saveProfile: async () => 0,
  deleteProfile: async () => {},
  setActiveProfile: async () => {},
  saveChain: async () => {},
  deleteChain: async () => {},
})

/**
 * ConfigProvider — единственный источник конфига для всего приложения.
 * Заменяет per-component useConfig, иначе изменения (например toggle
 * always_on_top) не пробрасывались между компонентами и приходилось
 * перезапускать приложение.
 */
export function ConfigProvider({ children }: { children: React.ReactNode }) {
  const [cfg, setCfg] = React.useState<macro.Config | null>(null)

  const reload = React.useCallback(async () => {
    const c = await GetConfig()
    setCfg(c)
  }, [])

  React.useEffect(() => { reload() }, [reload])

  const value = React.useMemo<Ctx>(() => ({
    cfg,
    reload,
    saveProfile: async (idx, p) => {
      const newIdx = await SaveProfile(idx, p)
      await reload()
      return newIdx
    },
    deleteProfile: async (idx) => {
      await DeleteProfile(idx)
      await reload()
    },
    setActiveProfile: async (idx) => {
      await SetActiveProfile(idx)
      await reload()
    },
    saveChain: async (idx, ch) => {
      await SaveChain(idx, ch)
      await reload()
    },
    deleteChain: async (idx) => {
      await DeleteChain(idx)
      await reload()
    },
  }), [cfg, reload])

  return <ConfigCtx.Provider value={value}>{children}</ConfigCtx.Provider>
}

export function useConfig() {
  return React.useContext(ConfigCtx)
}
