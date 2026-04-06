import { create } from 'zustand'
import { persist } from 'zustand/middleware'

interface EditorState {
  language: string
  theme: 'vs-dark' | 'light'
  fontSize: number
  tabSize: number
  wordWrap: 'on' | 'off'

  setLanguage: (lang: string) => void
  setTheme: (theme: 'vs-dark' | 'light') => void
  setFontSize: (size: number) => void
  setTabSize: (size: number) => void
  setWordWrap: (wrap: 'on' | 'off') => void
}

export const useEditorStore = create<EditorState>()(
  persist(
    (set) => ({
      language: 'cpp',
      theme: 'vs-dark',
      fontSize: 14,
      tabSize: 4,
      wordWrap: 'on',

      setLanguage: (language) => set({ language }),
      setTheme: (theme) => set({ theme }),
      setFontSize: (fontSize) => set({ fontSize }),
      setTabSize: (tabSize) => set({ tabSize }),
      setWordWrap: (wordWrap) => set({ wordWrap }),
    }),
    {
      name: 'editor-settings',
    }
  )
)