'use client'

import { useState, useEffect } from 'react'
import { useWebSocket } from '@/hooks/useWebSocket'
import { VerdictBadge } from '@/components/ui/Badge'
import type { Verdict } from '@/types'

interface SubmissionStatusProps {
  submissionId: string
  initialStatus?: string
  onComplete?: (verdict: Verdict) => void
}

type Status = 'pending' | 'compiling' | 'running' | 'completed' | 'error'

export function SubmissionStatus({
  submissionId,
  initialStatus = 'pending',
  onComplete,
}: SubmissionStatusProps) {
  const [status, setStatus] = useState<Status>(initialStatus as Status)
  const [progress, setProgress] = useState(0)
  const [verdict, setVerdict] = useState<Verdict | null>(null)
  const [runtime, setRuntime] = useState<number | null>(null)
  const [memory, setMemory] = useState<number | null>(null)

  const { subscribeToSubmission } = useWebSocket()

  useEffect(() => {
    const unsubscribe = subscribeToSubmission(submissionId, (update) => {
      setStatus(update.status as Status)
      setProgress(update.progress)

      if (update.verdict) {
        setVerdict(update.verdict as Verdict)
      }
      if (update.runtime) {
        setRuntime(update.runtime)
      }
      if (update.memory) {
        setMemory(update.memory)
      }

      if (update.status === 'completed' && onComplete) {
        onComplete(update.verdict as Verdict)
      }
    })

    return unsubscribe
  }, [submissionId, subscribeToSubmission, onComplete])

  if (status === 'completed' && verdict) {
    return (
      <div className="flex items-center gap-4 p-4 bg-gray-50 rounded-lg">
        <VerdictBadge verdict={verdict} size="lg" />
        {runtime !== null && (
          <span className="text-sm text-gray-600">Time: {runtime.toFixed(2)}s</span>
        )}
        {memory !== null && (
          <span className="text-sm text-gray-600">
            Memory: {(memory / 1024).toFixed(2)} MB
          </span>
        )}
      </div>
    )
  }

  return (
    <div className="p-4 bg-blue-50 rounded-lg judging-pulse border-2 border-blue-200">
      <div className="flex items-center gap-3">
        <div className="animate-spin rounded-full h-5 w-5 border-b-2 border-blue-600" />
        <div>
          <p className="font-medium text-blue-800 capitalize">{status}...</p>
          {status === 'running' && (
            <p className="text-sm text-blue-600">
              Test case {Math.round(progress / 100)} completed
            </p>
          )}
        </div>
      </div>
      <div className="mt-3 h-2 bg-blue-200 rounded-full overflow-hidden">
        <div
          className="h-full bg-blue-600 transition-all duration-300"
          style={{ width: `${progress}%` }}
        />
      </div>
    </div>
  )
}