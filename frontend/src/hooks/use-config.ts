import { useEffect, useState, useCallback } from 'react'
import { GetConfig, SaveProfile, DeleteProfile, SetActiveProfile, SaveChain, DeleteChain } from '../../wailsjs/go/main/App'
import type { macro } from '../../wailsjs/go/models'

export function useConfig() {
  const [cfg, setCfg] = useState<macro.Config | null>(null)

  const reload = useCallback(async () => {
    const c = await GetConfig()
    setCfg(c)
  }, [])

  useEffect(() => {
    reload()
  }, [reload])

  return {
    cfg,
    reload,
    saveProfile: async (idx: number, p: macro.SimpleConfig) => {
      const newIdx = await SaveProfile(idx, p)
      await reload()
      return newIdx
    },
    deleteProfile: async (idx: number) => {
      await DeleteProfile(idx)
      await reload()
    },
    setActiveProfile: async (idx: number) => {
      await SetActiveProfile(idx)
      await reload()
    },
    saveChain: async (idx: number, ch: macro.Chain) => {
      await SaveChain(idx, ch)
      await reload()
    },
    deleteChain: async (idx: number) => {
      await DeleteChain(idx)
      await reload()
    },
  }
}
