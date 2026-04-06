import { useEffect, useRef, useCallback } from 'react'
import { useAuthStore } from '@/stores/authStore'

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

type MessageHandler = (data: JudgeProgress) => void

export function useWebSocket() {
  const wsRef = useRef<WebSocket | null>(null)
  const handlersRef = useRef<Map<string, Set<MessageHandler>>>(new Map())
  const { token } = useAuthStore()

  useEffect(() => {
    if (!token) return

    const wsUrl = `${process.env.NEXT_PUBLIC_WS_URL || 'ws://localhost:8080'}/ws?token=${token}`

    wsRef.current = new WebSocket(wsUrl)

    wsRef.current.onopen = () => {
      console.log('WebSocket connected')
    }

    wsRef.current.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data)
        const handlers = handlersRef.current.get(data.type)
        if (handlers) {
          handlers.forEach((handler) => handler(data.payload))
        }
      } catch (err) {
        console.error('WebSocket message parse error:', err)
      }
    }

    wsRef.current.onerror = (error) => {
      console.error('WebSocket error:', error)
    }

    wsRef.current.onclose = () => {
      console.log('WebSocket disconnected')
    }

    return () => {
      wsRef.current?.close()
    }
  }, [token])

  const subscribe = useCallback((eventType: string, handler: MessageHandler) => {
    if (!handlersRef.current.has(eventType)) {
      handlersRef.current.set(eventType, new Set())
    }
    handlersRef.current.get(eventType)!.add(handler)

    return () => {
      handlersRef.current.get(eventType)?.delete(handler)
    }
  }, [])

  const subscribeToSubmission = useCallback(
    (submissionId: string, onUpdate: (progress: JudgeProgress) => void) => {
      return subscribe('judge_result', (data) => {
        if (data.submission_id === submissionId) {
          onUpdate(data)
        }
      })
    },
    [subscribe]
  )

  return {
    subscribe,
    subscribeToSubmission,
    isConnected: wsRef.current?.readyState === WebSocket.OPEN,
  }
}