'use client'

import { useEffect, useRef, useCallback, useState } from 'react'

interface SSEEvent {
  event: string
  data: Record<string, unknown>
}

interface JudgeProgress {
  submission_id: string
  status: 'pending' | 'compiling' | 'running' | 'completed' | 'error'
  progress: number
  current_case: number
  total_cases: number
  verdict?: string
  runtime?: number
  memory?: number
}

interface JudgeRunResult {
  submission_id: string
  test_case_id: string
  rank: number
  verdict: string
  runtime: number
  memory: number
}

interface UseSSEOptions {
  onError?: (error: Error) => void
  onConnect?: () => void
  onDisconnect?: () => void
}

export function useSSE(submissionId: string | null, options?: UseSSEOptions) {
  const eventSourceRef = useRef<EventSource | null>(null)
  const [isConnected, setIsConnected] = useState(false)
  const [status, setStatus] = useState<JudgeProgress | null>(null)
  const [runs, setRuns] = useState<JudgeRunResult[]>([])

  const connect = useCallback(() => {
    if (!submissionId || eventSourceRef.current) return

    const BFF_URL = process.env.NEXT_PUBLIC_BFF_URL || ''
    const url = `${BFF_URL}/api/v1/submissions/${submissionId}/stream`

    const eventSource = new EventSource(url)
    eventSourceRef.current = eventSource

    eventSource.onopen = () => {
      setIsConnected(true)
      options?.onConnect?.()
    }

    eventSource.onerror = (err) => {
      setIsConnected(false)
      options?.onError?.(new Error('SSE connection error'))
      // EventSource will automatically try to reconnect
    }

    // Handle connected event
    eventSource.addEventListener('connected', (event) => {
      setIsConnected(true)
    })

    // Handle status/progress updates
    eventSource.addEventListener('status', (event) => {
      try {
        const data = JSON.parse(event.data) as JudgeProgress
        setStatus(data)
      } catch (e) {
        console.error('Failed to parse status event:', e)
      }
    })

    // Handle individual test case run results
    eventSource.addEventListener('run', (event) => {
      try {
        const data = JSON.parse(event.data) as JudgeRunResult
        setRuns((prev) => {
          // Add or update the run
          const existing = prev.findIndex((r) => r.rank === data.rank)
          if (existing >= 0) {
            const updated = [...prev]
            updated[existing] = data
            return updated
          }
          return [...prev, data].sort((a, b) => a.rank - b.rank)
        })
      } catch (e) {
        console.error('Failed to parse run event:', e)
      }
    })

    // Handle final verdict
    eventSource.addEventListener('verdict', (event) => {
      try {
        const data = JSON.parse(event.data) as JudgeProgress
        setStatus(data)
        setIsConnected(false)
        // Connection will close after verdict
      } catch (e) {
        console.error('Failed to parse verdict event:', e)
      }
    })
  }, [submissionId, options])

  const disconnect = useCallback(() => {
    if (eventSourceRef.current) {
      eventSourceRef.current.close()
      eventSourceRef.current = null
      setIsConnected(false)
      options?.onDisconnect?.()
    }
  }, [options])

  useEffect(() => {
    connect()
    return () => disconnect()
  }, [connect, disconnect])

  const reset = useCallback(() => {
    setStatus(null)
    setRuns([])
  }, [])

  return {
    isConnected,
    status,
    runs,
    connect,
    disconnect,
    reset,
  }
}