'use client'

import { useState, useEffect } from 'react'
import { useWebSocket } from '@/hooks/useWebSocket'
import { Badge } from '@/components/ui/badge'
import { VERDICT_CONFIG, type Verdict } from '@/types'

interface SubmissionStatusProps {
  submissionId: string
  initialStatus?: string
  onComplete?: (verdict: Verdict) => void
}

type Status = 'pending' | 'compiling' | 'running' | 'completed' | 'error'

const verdictVariant = (verdict: Verdict): 'default' | 'destructive' => {
  return verdict === 'correct' ? 'default' : 'destructive'
}

const verdictClass = (verdict: Verdict): string => {
  const config = VERDICT_CONFIG[verdict]
  return config?.color ? `${config.color} text-white hover:opacity-90` : ''
}

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
      <div className="flex items-center gap-4 rounded-xl border bg-card p-4">
        <Badge variant={verdictVariant(verdict)} className={verdictClass(verdict)}>
          {VERDICT_CONFIG[verdict]?.label || verdict}
        </Badge>
        {runtime !== null && (
          <span className="text-sm text-muted-foreground">Time: {runtime.toFixed(2)}s</span>
        )}
        {memory !== null && (
          <span className="text-sm text-muted-foreground">
            Memory: {(memory / 1024).toFixed(2)} MB
          </span>
        )}
      </div>
    )
  }

  return (
    <div className="p-4 bg-primary/10 rounded-xl judging-pulse border-2 border-primary/20">
      <div className="flex items-center gap-3">
        <div className="animate-spin rounded-full h-5 w-5 border-b-2 border-primary" />
        <div>
          <p className="font-medium text-primary capitalize">{status}...</p>
          {status === 'running' && (
            <p className="text-sm text-primary">
              Test case {Math.round(progress / 100)} completed
            </p>
          )}
        </div>
      </div>
      <div className="mt-3 h-2 bg-primary/20 rounded-full overflow-hidden">
        <div
          className="h-full bg-primary transition-all duration-300"
          style={{ width: `${progress}%` }}
        />
      </div>
    </div>
  )
}