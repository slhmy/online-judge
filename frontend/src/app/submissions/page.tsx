'use client'

import { useState } from 'react'
import Link from 'next/link'
import { useSubmissions } from '@/hooks/useApi'
import { VERDICT_CONFIG } from '@/types'

interface Submission {
  id: string
  user?: {
    id?: string
    username?: string
  }
  problem_id: string
  problem_name: string
  language_id: string
  verdict: string | number
  runtime: number
  memory: number
  submit_time: string
}

interface SubmissionsResponse {
  submissions: Submission[]
  pagination: {
    total: number
    page: number
    page_size: number
  }
}

// Convert verdict to key for VERDICT_CONFIG
const getVerdictKey = (verdict: string | number): string => {
  if (!verdict && verdict !== 0) return ''
  // Handle number (proto enum value)
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
  // Handle string like "VERDICT_CORRECT" or "correct"
  const str = String(verdict)
  if (str.startsWith('VERDICT_')) {
    return str.replace('VERDICT_', '').toLowerCase().replace('_', '-')
  }
  return str
}

export default function SubmissionsPage() {
  const [currentPage, setCurrentPage] = useState(1)
  const pageSize = 20

  const { data, isLoading, error } = useSubmissions(currentPage, pageSize) as {
    data: SubmissionsResponse | undefined
    isLoading: boolean
    error: Error | null
  }
  const submissions: Submission[] = data?.submissions || []
  const pagination = data?.pagination
  const total = pagination?.total || 0
  const page = pagination?.page || currentPage
  const size = pagination?.page_size || pageSize
  const totalPages = Math.max(1, Math.ceil(total / size))

  const goToPrevPage = () => {
    if (page > 1) {
      setCurrentPage(page - 1)
    }
  }

  const goToNextPage = () => {
    if (page < totalPages) {
      setCurrentPage(page + 1)
    }
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
    const date = new Date(timeStr)
    const now = new Date()
    const diffMs = now.getTime() - date.getTime()
    const diffMins = Math.floor(diffMs / 60000)
    const diffHours = Math.floor(diffMs / 3600000)
    const diffDays = Math.floor(diffMs / 86400000)

    if (diffMins < 1) return 'Just now'
    if (diffMins < 60) return `${diffMins}m ago`
    if (diffHours < 24) return `${diffHours}h ago`
    if (diffDays < 7) return `${diffDays}d ago`
    return date.toLocaleDateString()
  }

  const formatSubmitter = (sub: Submission) => {
    if (sub.user?.username) return sub.user.username
    if (sub.user?.id) return sub.user.id.slice(0, 8)
    return '-'
  }

  if (error) {
    return (
      <div className="px-4 py-6">
        <h1 className="text-2xl font-bold mb-6 text-gray-900 dark:text-gray-100">Submissions</h1>
        <div className="text-center py-10 text-red-400">
          Error loading submissions: {error.message}
        </div>
      </div>
    )
  }

  return (
    <div className="px-4 py-6">
      <h1 className="text-2xl font-bold mb-6 text-gray-900 dark:text-gray-100">Submissions</h1>

      {isLoading ? (
        <div className="text-center py-10 text-gray-600 dark:text-gray-400">Loading...</div>
      ) : submissions.length === 0 ? (
        <div className="text-center py-10 text-gray-500">
          No submissions yet. <Link href="/problems" className="text-blue-600 dark:text-blue-400 hover:underline">Solve some problems!</Link>
        </div>
      ) : (
        <div className="space-y-4">
          <div className="overflow-x-auto">
            <table className="min-w-full bg-white dark:bg-gray-800 rounded-lg shadow">
              <thead className="bg-gray-100 dark:bg-gray-700">
                <tr>
                  <th className="px-4 py-3 text-left text-sm font-medium text-gray-700 dark:text-gray-300">ID</th>
                  <th className="px-4 py-3 text-left text-sm font-medium text-gray-700 dark:text-gray-300">Submitter</th>
                  <th className="px-4 py-3 text-left text-sm font-medium text-gray-700 dark:text-gray-300">Problem</th>
                  <th className="px-4 py-3 text-left text-sm font-medium text-gray-700 dark:text-gray-300">Language</th>
                  <th className="px-4 py-3 text-left text-sm font-medium text-gray-700 dark:text-gray-300">Verdict</th>
                  <th className="px-4 py-3 text-left text-sm font-medium text-gray-700 dark:text-gray-300">Time</th>
                  <th className="px-4 py-3 text-left text-sm font-medium text-gray-700 dark:text-gray-300">Memory</th>
                  <th className="px-4 py-3 text-left text-sm font-medium text-gray-700 dark:text-gray-300">Submitted</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-200 dark:divide-gray-700">
                {submissions.map((sub) => {
                  const verdictKey = getVerdictKey(sub.verdict)
                  const verdictInfo = verdictKey ? VERDICT_CONFIG[verdictKey as keyof typeof VERDICT_CONFIG] : null
                  return (
                    <tr key={sub.id} className="hover:bg-gray-100 dark:bg-gray-700/50">
                      <td className="px-4 py-3 text-sm">
                        <Link href={`/submissions/${sub.id}`} className="text-blue-600 dark:text-blue-400 hover:underline font-mono">
                          {sub.id.slice(0, 8)}
                        </Link>
                      </td>
                      <td className="px-4 py-3 text-sm text-gray-700 dark:text-gray-300">{formatSubmitter(sub)}</td>
                      <td className="px-4 py-3 text-sm">
                        <Link href={`/problems/${sub.problem_id}`} className="text-blue-600 dark:text-blue-400 hover:underline">
                          {sub.problem_name || sub.problem_id}
                        </Link>
                      </td>
                      <td className="px-4 py-3 text-sm text-gray-700 dark:text-gray-300">{sub.language_id}</td>
                      <td className="px-4 py-3 text-sm">
                        <span className={`px-2 py-1 rounded text-white text-xs font-medium ${verdictInfo?.color || 'bg-gray-500'}`}>
                          {verdictInfo?.label || 'Pending'}
                        </span>
                      </td>
                      <td className="px-4 py-3 text-sm text-gray-700 dark:text-gray-300">{formatRuntime(sub.runtime)}</td>
                      <td className="px-4 py-3 text-sm text-gray-700 dark:text-gray-300">{formatMemory(sub.memory)}</td>
                      <td className="px-4 py-3 text-sm text-gray-600 dark:text-gray-400">
                        {formatTime(sub.submit_time)}
                      </td>
                    </tr>
                  )
                })}
              </tbody>
            </table>
          </div>

          <div className="flex items-center justify-between bg-white dark:bg-gray-800 rounded-lg shadow px-4 py-3">
            <div className="text-sm text-gray-600 dark:text-gray-400">
              Page {page} / {totalPages} • Total {total}
            </div>
            <div className="flex items-center gap-2">
              <button
                type="button"
                onClick={goToPrevPage}
                disabled={isLoading || page <= 1}
                className="px-3 py-1.5 rounded-md border border-gray-300 dark:border-gray-600 text-sm text-gray-700 dark:text-gray-200 disabled:opacity-50 disabled:cursor-not-allowed hover:bg-gray-100 dark:hover:bg-gray-700"
              >
                Previous
              </button>
              <button
                type="button"
                onClick={goToNextPage}
                disabled={isLoading || page >= totalPages}
                className="px-3 py-1.5 rounded-md border border-gray-300 dark:border-gray-600 text-sm text-gray-700 dark:text-gray-200 disabled:opacity-50 disabled:cursor-not-allowed hover:bg-gray-100 dark:hover:bg-gray-700"
              >
                Next
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}