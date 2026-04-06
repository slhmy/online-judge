'use client'

import { useParams } from 'next/navigation'
import Link from 'next/link'
import { useSubmission } from '@/hooks/useApi'
import { VERDICT_CONFIG } from '@/types'

interface SubmissionResponse {
  submission: {
    id: string
    user_id: string
    problem_id: string
    contest_id: string
    language_id: string
    source_path: string // Now contains actual source code
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

export default function SubmissionDetailPage() {
  const params = useParams()
  const submissionId = params.id as string

  const { data, isLoading, error } = useSubmission(submissionId) as {
    data: SubmissionResponse | undefined
    isLoading: boolean
    error: Error | null
  }

  const formatRuntime = (seconds: number) => {
    if (seconds === undefined || seconds === null || seconds === 0) return '-'
    return `${seconds.toFixed(3)}s`
  }

  const formatMemory = (kb: number) => {
    if (kb === undefined || kb === null || kb === 0) return '-'
    return `${(kb / 1024).toFixed(2)} MB`
  }

  const formatTime = (timeStr: string) => {
    if (!timeStr) return '-'
    return new Date(timeStr).toLocaleString()
  }

  const getVerdictKey = (verdict: string | number): string => {
    if (!verdict) return ''
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

  if (error) {
    return (
      <div className="px-4 py-6">
        <h1 className="text-2xl font-bold mb-6 text-gray-900 dark:text-gray-100">Submission</h1>
        <div className="text-center py-10 text-red-400">
          Error loading submission: {error.message}
        </div>
      </div>
    )
  }

  if (isLoading) {
    return (
      <div className="px-4 py-6">
        <h1 className="text-2xl font-bold mb-6 text-gray-900 dark:text-gray-100">Submission</h1>
        <div className="text-center py-10 text-gray-600 dark:text-gray-400">Loading...</div>
      </div>
    )
  }

  if (!data || !data.submission) {
    return (
      <div className="px-4 py-6">
        <h1 className="text-2xl font-bold mb-6 text-gray-900 dark:text-gray-100">Submission</h1>
        <div className="text-center py-10 text-gray-500">Submission not found</div>
      </div>
    )
  }

  const submission = data.submission
  const judging = data.latest_judging
  const verdictKey = judging?.verdict ? getVerdictKey(judging.verdict) : ''
  const verdictInfo = verdictKey ? VERDICT_CONFIG[verdictKey as keyof typeof VERDICT_CONFIG] : null

  return (
    <div className="px-4 py-6">
      <div className="flex items-center gap-2 mb-6">
        <Link href="/submissions" className="text-blue-600 dark:text-blue-400 hover:underline">
          Submissions
        </Link>
        <span className="text-gray-500">/</span>
        <span className="text-gray-900 dark:text-gray-100 font-mono">{submission.id.slice(0, 8)}</span>
      </div>

      <div className="bg-white dark:bg-gray-800 rounded-lg shadow p-6">
        <div className="grid grid-cols-2 gap-4 mb-6">
          <div>
            <label className="text-gray-600 dark:text-gray-400 text-sm">ID</label>
            <span className="text-gray-900 dark:text-gray-100 font-mono block">{submission.id}</span>
          </div>
          <div>
            <label className="text-gray-600 dark:text-gray-400 text-sm">Problem</label>
            <Link href={`/problems/${submission.problem_id}`} className="text-blue-600 dark:text-blue-400 hover:underline block">
              {submission.problem_id}
            </Link>
          </div>
          <div>
            <label className="text-gray-600 dark:text-gray-400 text-sm">User</label>
            <span className="text-gray-900 dark:text-gray-100 block">{submission.user_id}</span>
          </div>
          <div>
            <label className="text-gray-600 dark:text-gray-400 text-sm">Language</label>
            <span className="text-gray-900 dark:text-gray-100 block">{submission.language_id}</span>
          </div>
          <div>
            <label className="text-gray-600 dark:text-gray-400 text-sm">Verdict</label>
            <span className={`px-2 py-1 rounded text-white text-sm font-medium inline-block ${verdictInfo?.color || 'bg-gray-500'}`}>
              {verdictInfo?.label || 'Pending'}
            </span>
          </div>
          <div>
            <label className="text-gray-600 dark:text-gray-400 text-sm">Time</label>
            <span className="text-gray-900 dark:text-gray-100 block">{formatRuntime(judging?.max_runtime || 0)}</span>
          </div>
          <div>
            <label className="text-gray-600 dark:text-gray-400 text-sm">Memory</label>
            <span className="text-gray-900 dark:text-gray-100 block">{formatMemory(judging?.max_memory || 0)}</span>
          </div>
          <div>
            <label className="text-gray-600 dark:text-gray-400 text-sm">Submitted</label>
            <span className="text-gray-900 dark:text-gray-100 block">{formatTime(submission.submit_time)}</span>
          </div>
        </div>

        {judging && getVerdictKey(judging.verdict) === 'compiler-error' && (
          <div className="mb-6">
            <label className="text-gray-600 dark:text-gray-400 text-sm mb-2 block">Compile Error</label>
            <pre className="bg-gray-900 p-4 rounded text-sm text-red-300 overflow-auto max-h-64">
              Compilation failed. Check your code for syntax errors.
            </pre>
          </div>
        )}

        {submission.source_path && submission.source_path !== 'stored-in-db' && (
          <div className="mt-6">
            <label className="text-gray-600 dark:text-gray-400 text-sm mb-2 block">Source Code</label>
            <pre className="bg-gray-900 p-4 rounded text-sm text-gray-300 overflow-auto max-h-64">
              {submission.source_path}
            </pre>
          </div>
        )}
      </div>
    </div>
  )
}