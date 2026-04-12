'use client'

import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { useState } from 'react'
import { AppearanceProvider } from '@/components/providers/AppearanceProvider'

export function Providers({ children }: { children: React.ReactNode }) {
  const [queryClient] = useState(() => new QueryClient({
    defaultOptions: {
      queries: {
        staleTime: 60 * 1000, // 1 minute
        refetchOnWindowFocus: false,
      },
    },
  }))

  return (
    <AppearanceProvider>
      <QueryClientProvider client={queryClient}>
        {children}
      </QueryClientProvider>
    </AppearanceProvider>
  )
}