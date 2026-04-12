'use client'

import { useParams } from 'next/navigation'
import Link from 'next/link'
import { useState, useEffect } from 'react'
import { useSubmission, useJudgingRuns, useRejudgeSubmission } from '@/hooks/useApi'
import { useSSE } from '@/hooks/useSSE'
import { useAuthStore } from '@/stores/authStore'
import { VERDICT_CONFIG, type Verdict, type JudgingRun } from '@/types'

interface SubmissionResponse {
  submission: {
    id: string
    user_id: string
    problem_id: string
    contest_id: string
    language_id: string
    source_path: string // Contains actual source code
    submit_time: string
  }
  latest_judging: {
    id: string
    submission_id: string
    verdict: string | number
    max_runtime: number
    max_memory: number
    compile_success: boolean
    compile_output_path: string
    start_time: string
    end_time: string
  } | null
}

interface JudgingRunsResponse {
  runs: JudgingRun[]
}

// Verdict icon component
function VerdictIcon({ verdict, size = 'md' }: { verdict: Verdict; size?: 'sm' | 'md' | 'lg' }) {
  const config = VERDICT_CONFIG[verdict] || { color: 'bg-muted', icon: '?', bgColor: 'bg-muted/40' }

  const sizeClasses = {
    sm: 'w-6 h-6 text-xs',
    md: 'w-8 h-8 text-sm',
    lg: 'w-10 h-10 text-base',
  }

  return (
    <div
      className={`${config.color} ${sizeClasses[size]} rounded-full flex items-center justify-center text-white font-bold shadow-sm`}
    >
      {config.icon}
    </div>
  )
}

// Test case result row
function TestCaseResult({
  run,
  isExpanded,
  onToggle,
  problemTimeLimit,
  problemMemoryLimit
}: {
  run: JudgingRun
  isExpanded: boolean
  onToggle: () => void
  problemTimeLimit?: number
  problemMemoryLimit?: number
}) {
  const verdictKey = getVerdictKey(run.verdict)
  const config = VERDICT_CONFIG[verdictKey as Verdict] || { color: 'bg-muted', label: 'Unknown', bgColor: 'bg-muted/40' }

  const isTimeLimitExceeded = problemTimeLimit && run.runtime > problemTimeLimit
  const isMemoryLimitExceeded = problemMemoryLimit && run.memory > problemMemoryLimit * 1024

  return (
    <div className={`border rounded-xl ${config.bgColor} mb-2`}>
      <button
        onClick={onToggle}
        className="w-full flex items-center justify-between p-3 hover:bg-muted/60 transition-colors"
      >
        <div className="flex items-center gap-3">
          <VerdictIcon verdict={verdictKey as Verdict} size="sm" />
          <span className="text-sm font-medium text-foreground">
            Test Case #{run.rank}
          </span>
          {run.testCaseId && (
            <span className="text-xs text-muted-foreground font-mono">
              ({run.testCaseId.slice(0, 8)})
            </span>
          )}
        </div>
        <div className="flex items-center gap-4 text-sm">
          <span className={`${isTimeLimitExceeded ? 'text-red-600 font-semibold' : 'text-muted-foreground'}`}>
            {formatRuntime(run.runtime)}
          </span>
          <span className={`${isMemoryLimitExceeded ? 'text-red-600 font-semibold' : 'text-muted-foreground'}`}>
            {formatMemory(run.memory)}
          </span>
          <span className="text-muted-foreground">
            {isExpanded ? '▼' : '▶'}
          </span>
        </div>
      </button>

      {isExpanded && (
        <div className="px-3 pb-3 border-t border-border">
          {/* Show diff/error output for failed cases */}
          {verdictKey !== 'correct' && (
            <div className="mt-2">
              <div className="text-xs text-muted-foreground mb-1">Details:</div>
              <div className="bg-zinc-950 rounded p-3 text-sm overflow-auto max-h-48">
                {verdictKey === 'wrong-answer' && (
                  <div className="text-yellow-300">
                    <p className="text-muted-foreground mb-2">Your output differs from expected output.</p>
                    <p className="text-red-400">Check your logic for edge cases.</p>
                  </div>
                )}
                {verdictKey === 'run-error' && (
                  <div className="text-red-300">
                    <p>Runtime error occurred during execution.</p>
                    <p className="text-muted-foreground mt-1">Common causes: division by zero, null pointer, array index out of bounds</p>
                  </div>
                )}
                {verdictKey === 'timelimit' && (
                  <div className="text-yellow-300">
                    <p>Your solution took too long.</p>
                    <p className="text-muted-foreground mt-1">Consider optimizing your algorithm or checking for infinite loops.</p>
                  </div>
                )}
                {verdictKey === 'memory-limit' && (
                  <div className="text-orange-300">
                    <p>Memory limit exceeded.</p>
                    <p className="text-muted-foreground mt-1">Reduce data structure sizes or check for memory leaks.</p>
                  </div>
                )}
              </div>
            </div>
          )}

          {/* Runtime/Memory breakdown */}
          <div className="mt-2 grid grid-cols-2 gap-4">
            <div>
              <span className="text-xs text-muted-foreground">Runtime</span>
              <div className="text-sm text-foreground font-medium">
                {formatRuntime(run.runtime)}
                {isTimeLimitExceeded && (
                  <span className="text-xs text-red-600 ml-1">(exceeds {problemTimeLimit}s)</span>
                )}
              </div>
            </div>
            <div>
              <span className="text-xs text-muted-foreground">Memory</span>
              <div className="text-sm text-foreground font-medium">
                {formatMemory(run.memory)}
                {isMemoryLimitExceeded && (
                  <span className="text-xs text-red-600 ml-1">(exceeds {problemMemoryLimit}MB)</span>
                )}
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}

// Real-time judging progress indicator
function JudgingProgressIndicator({
  status,
  runs,
  totalCases
}: {
  status: ReturnType<typeof useSSE>['status']
  runs: ReturnType<typeof useSSE>['runs']
  totalCases?: number
}) {
  if (!status) return null

  const progress = status.progress || 0
  const currentCase = status.current_case || 0
  const estimatedTotal = totalCases || status.total_cases || runs.length || 0

  return (
    <div className="bg-primary/10 rounded-xl p-4 mb-6 border-2 border-primary/20">
      <div className="flex items-center gap-3 mb-3">
        <div className="animate-spin rounded-full h-5 w-5 border-b-2 border-primary" />
        <div>
          <p className="font-medium text-primary capitalize">
            {status.status}...
          </p>
          {status.status === 'running' && (
            <p className="text-sm text-primary">
              Running test case {currentCase} of {estimatedTotal}
            </p>
          )}
        </div>
      </div>

      {/* Progress bar */}
      <div className="h-2 bg-primary/20 rounded-full overflow-hidden">
        <div
          className="h-full bg-primary transition-all duration-300 ease-out"
          style={{ width: `${progress}%` }}
        />
      </div>

      {/* Real-time test case icons */}
      {runs.length > 0 && (
        <div className="mt-3 flex gap-1 overflow-x-auto pb-1">
          {runs.map((run, idx) => (
            <VerdictIcon
              key={idx}
              verdict={getVerdictKey(run.verdict) as Verdict}
              size="sm"
            />
          ))}
          {/* Show placeholder for remaining cases */}
          {estimatedTotal > runs.length && Array.from({ length: estimatedTotal - runs.length }).map((_, idx) => (
            <div
              key={`pending-${idx}`}
              className="w-6 h-6 rounded-full bg-muted/80 flex items-center justify-center"
            >
              <span className="text-xs text-muted-foreground">?</span>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}

export default function SubmissionDetailPage() {
  const params = useParams()
  const submissionId = params.id as string
  const { user } = useAuthStore()
  const isAdmin = user?.role === 'admin'

  const { data, isLoading, error, refetch } = useSubmission(submissionId) as {
    data: SubmissionResponse | undefined
    isLoading: boolean
    error: Error | null
    refetch: () => void
  }

  const { data: runsData } = useJudgingRuns(submissionId) as {
    data: JudgingRunsResponse | undefined
  }

  const rejudgeMutation = useRejudgeSubmission()

  // SSE for real-time updates
  const { isConnected, status: sseStatus, runs: sseRuns, reset: resetSSE } = useSSE(
    submissionId,
    {
      onDisconnect: () => {
        // Refetch data when judging completes
        setTimeout(() => refetch(), 500)
      }
    }
  )

  const [expandedCases, setExpandedCases] = useState<Set<number>>(new Set())
  const [isRejudging, setIsRejudging] = useState(false)

  // Use SSE runs if available, otherwise use API runs
  const displayRuns = sseRuns.length > 0
    ? sseRuns.map(r => ({
        id: '',
        judgingId: '',
        testCaseId: r.test_case_id,
        rank: r.rank,
        runtime: r.runtime,
        memory: r.memory,
        verdict: getVerdictKey(r.verdict) as Verdict
      }))
    : (runsData?.runs || [])

  // Determine if judging is in progress
  const judging = data?.latest_judging
  const verdictKey = judging?.verdict ? getVerdictKey(judging.verdict) : ''
  const isJudging = verdictKey === '' || isConnected || sseStatus?.status === 'running' || sseStatus?.status === 'compiling'

  const handleRejudge = async () => {
    if (!isAdmin) return
    setIsRejudging(true)
    resetSSE()
    try {
      await rejudgeMutation.mutateAsync(submissionId)
    } finally {
      setIsRejudging(false)
    }
  }

  const toggleCaseExpanded = (rank: number) => {
    setExpandedCases(prev => {
      const newSet = new Set(prev)
      if (newSet.has(rank)) {
        newSet.delete(rank)
      } else {
        newSet.add(rank)
      }
      return newSet
    })
  }

  if (error) {
    return (
      <div className="px-4 py-6">
        <h1 className="text-2xl font-bold mb-6 text-foreground">Submission</h1>
        <div className="text-center py-10 text-red-400">
          Error loading submission: {error.message}
        </div>
      </div>
    )
  }

  if (isLoading) {
    return (
      <div className="px-4 py-6">
        <h1 className="text-2xl font-bold mb-6 text-foreground">Submission</h1>
        <div className="text-center py-10 text-muted-foreground">Loading...</div>
      </div>
    )
  }

  if (!data || !data.submission) {
    return (
      <div className="px-4 py-6">
        <h1 className="text-2xl font-bold mb-6 text-foreground">Submission</h1>
        <div className="text-center py-10 text-muted-foreground">Submission not found</div>
      </div>
    )
  }

  const submission = data.submission
  const verdictInfo = verdictKey ? VERDICT_CONFIG[verdictKey as Verdict] : null

  return (
    <div className="px-4 py-6 max-w-4xl mx-auto">
      {/* Header */}
      <div className="flex items-center justify-between mb-6">
        <div className="flex items-center gap-2">
          <Link href="/submissions" className="text-primary hover:underline">
            Submissions
          </Link>
          <span className="text-muted-foreground">/</span>
          <span className="text-foreground font-mono">{submission.id.slice(0, 8)}</span>
        </div>

        {/* Rejudge button for admins */}
        {isAdmin && !isJudging && (
          <button
            onClick={handleRejudge}
            disabled={isRejudging}
            className="px-4 py-2 bg-orange-500 hover:bg-orange-600 disabled:bg-orange-300 text-white rounded-xl text-sm font-medium transition-colors"
          >
            {isRejudging ? 'Rejudging...' : 'Rejudge'}
          </button>
        )}
      </div>

      {/* Real-time judging progress */}
      {isJudging && (
        <JudgingProgressIndicator
          status={sseStatus}
          runs={sseRuns}
          totalCases={sseStatus?.total_cases}
        />
      )}

      {/* Main info card */}
      <div className="bg-card rounded-xl shadow mb-6">
        <div className="p-4 border-b border-border">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-4">
              {/* Large verdict icon */}
              {!isJudging && verdictKey && (
                <VerdictIcon verdict={verdictKey as Verdict} size="lg" />
              )}
              <div>
                <h2 className="text-xl font-semibold text-foreground">
                  {verdictInfo?.label || 'Pending'}
                </h2>
                <div className="text-sm text-muted-foreground">
                  {submission.language_id} • {formatTime(submission.submit_time)}
                </div>
              </div>
            </div>

            {/* Time & Memory summary */}
            {judging && !isJudging && (
              <div className="text-right">
                <div className="text-sm">
                  <span className="text-muted-foreground">Time: </span>
                  <span className="text-foreground font-medium">
                    {formatRuntime(judging.max_runtime)}
                  </span>
                </div>
                <div className="text-sm">
                  <span className="text-muted-foreground">Memory: </span>
                  <span className="text-foreground font-medium">
                    {formatMemory(judging.max_memory)}
                  </span>
                </div>
              </div>
            )}
          </div>
        </div>

        {/* Submission details grid */}
        <div className="p-4 grid grid-cols-2 md:grid-cols-4 gap-4">
          <div>
            <label className="text-muted-foreground text-xs uppercase tracking-wide">ID</label>
            <span className="text-foreground font-mono text-sm block">{submission.id.slice(0, 8)}</span>
          </div>
          <div>
            <label className="text-muted-foreground text-xs uppercase tracking-wide">Problem</label>
            <Link href={`/problems/${submission.problem_id}`} className="text-primary hover:underline text-sm block">
              {submission.problem_id.slice(0, 8)}
            </Link>
          </div>
          <div>
            <label className="text-muted-foreground text-xs uppercase tracking-wide">User</label>
            <span className="text-foreground text-sm block">{submission.user_id.slice(0, 8)}</span>
          </div>
          <div>
            <label className="text-muted-foreground text-xs uppercase tracking-wide">Language</label>
            <span className="text-foreground text-sm block capitalize">{submission.language_id}</span>
          </div>
        </div>
      </div>

      {/* Compilation output */}
      {judging && !judging.compile_success && (
        <div className="bg-card rounded-xl shadow mb-6">
          <div className="p-4 border-b border-border">
            <div className="flex items-center gap-2">
              <VerdictIcon verdict="compiler-error" size="sm" />
              <h3 className="text-lg font-semibold text-foreground">Compilation Error</h3>
            </div>
          </div>
          <div className="p-4">
            <pre className="bg-zinc-950 p-4 rounded text-sm text-red-300 overflow-auto max-h-64 font-mono">
              {judging.compile_output_path || 'Compilation failed. Check your code for syntax errors.'}
            </pre>
          </div>
        </div>
      )}

      {/* Source code display */}
      <div className="bg-card rounded-xl shadow mb-6">
        <div className="p-4 border-b border-border">
          <h3 className="text-lg font-semibold text-foreground">Source Code</h3>
        </div>
        <div className="p-4">
          <pre className="bg-zinc-950 p-4 rounded text-sm text-zinc-300 overflow-auto max-h-[500px] font-mono whitespace-pre-wrap">
            {submission.source_path && submission.source_path !== 'stored-in-db'
              ? submission.source_path
              : 'Source code not available'}
          </pre>
        </div>
      </div>

      {/* Test case results */}
      {displayRuns.length > 0 && (
        <div className="bg-card rounded-xl shadow mb-6">
          <div className="p-4 border-b border-border">
            <div className="flex items-center justify-between">
              <h3 className="text-lg font-semibold text-foreground">
                Test Cases ({displayRuns.length})
              </h3>
              {/* Summary verdict icons */}
              <div className="flex gap-1">
                {displayRuns.map((run, idx) => (
                  <VerdictIcon
                    key={idx}
                    verdict={run.verdict}
                    size="sm"
                  />
                ))}
              </div>
            </div>
          </div>
          <div className="p-4 max-h-[400px] overflow-y-auto">
            {displayRuns.map((run) => (
              <TestCaseResult
                key={run.rank}
                run={run}
                isExpanded={expandedCases.has(run.rank)}
                onToggle={() => toggleCaseExpanded(run.rank)}
              />
            ))}
          </div>
        </div>
      )}

      {/* SSE connection status */}
      {isConnected && (
        <div className="fixed bottom-4 right-4 bg-green-500 text-white px-3 py-1 rounded-full text-sm shadow-lg animate-pulse">
          Live updates connected
        </div>
      )}
    </div>
  )
}

// Helper functions
function formatRuntime(seconds: number): string {
  if (seconds === undefined || seconds === null || seconds === 0) return '-'
  return `${seconds.toFixed(3)}s`
}

function formatMemory(kb: number): string {
  if (kb === undefined || kb === null || kb === 0) return '-'
  return `${(kb / 1024).toFixed(2)} MB`
}

function formatTime(timeStr: string): string {
  if (!timeStr) return '-'
  return new Date(timeStr).toLocaleString()
}

function getVerdictKey(verdict: string | number | Verdict): string {
  if (!verdict) return ''
  // Handle Verdict type directly
  if (typeof verdict === 'string' && !verdict.startsWith('VERDICT_')) {
    return verdict
  }
  // Handle both string (VERDICT_CORRECT) and number (enum value)
  if (typeof verdict === 'number') {
    const verdictMap: Record<number, string> = {
      1: 'correct',
      2: 'wrong-answer',
      3: 'timelimit',
      4: 'memory-limit',
      5: 'run-error',
      6: 'compiler-error',
      7: 'output-limit',
      8: 'presentation',
    }
    return verdictMap[verdict] || ''
  }
  // Convert VERDICT_CORRECT -> correct
  return String(verdict).replace('VERDICT_', '').toLowerCase()
}