'use client'

import { createContext, useContext, useEffect, useMemo, useState } from 'react'

type RadiusPreset = 'compact' | 'default' | 'large'

const RADIUS_VALUES: Record<RadiusPreset, string> = {
  compact: '0.45rem',
  default: '0.625rem',
  large: '0.85rem',
}

const THEME_STORAGE_KEY = 'theme'
const RADIUS_STORAGE_KEY = 'radiusPreset'

interface AppearanceContextValue {
  darkMode: boolean
  toggleDarkMode: () => void
  setDarkMode: (value: boolean) => void
  radiusPreset: RadiusPreset
  setRadiusPreset: (value: RadiusPreset) => void
}

const AppearanceContext = createContext<AppearanceContextValue | null>(null)

export function AppearanceProvider({ children }: { children: React.ReactNode }) {
  const [darkMode, setDarkMode] = useState(true)
  const [radiusPreset, setRadiusPreset] = useState<RadiusPreset>('default')

  useEffect(() => {
    const savedTheme = localStorage.getItem(THEME_STORAGE_KEY)
    const savedRadius = localStorage.getItem(RADIUS_STORAGE_KEY)

    if (savedTheme === 'dark' || savedTheme === 'light') {
      setDarkMode(savedTheme === 'dark')
    } else {
      setDarkMode(window.matchMedia('(prefers-color-scheme: dark)').matches)
    }

    if (savedRadius === 'compact' || savedRadius === 'default' || savedRadius === 'large') {
      setRadiusPreset(savedRadius)
    }
  }, [])

  useEffect(() => {
    document.documentElement.classList.toggle('dark', darkMode)
    localStorage.setItem(THEME_STORAGE_KEY, darkMode ? 'dark' : 'light')
  }, [darkMode])

  useEffect(() => {
    document.documentElement.style.setProperty('--radius', RADIUS_VALUES[radiusPreset])
    localStorage.setItem(RADIUS_STORAGE_KEY, radiusPreset)
  }, [radiusPreset])

  const value = useMemo<AppearanceContextValue>(() => ({
    darkMode,
    toggleDarkMode: () => setDarkMode((v) => !v),
    setDarkMode,
    radiusPreset,
    setRadiusPreset,
  }), [darkMode, radiusPreset])

  return <AppearanceContext.Provider value={value}>{children}</AppearanceContext.Provider>
}

export function useAppearance() {
  const context = useContext(AppearanceContext)
  if (!context) {
    throw new Error('useAppearance must be used within AppearanceProvider')
  }
  return context
}
