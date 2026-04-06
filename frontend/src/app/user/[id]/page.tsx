'use client'

import { useState } from 'react'
import { useParams } from 'next/navigation'
import Link from 'next/link'
import { useUserProfile, useUserStats, useUserSubmissions } from '@/hooks/useApi'
import { VERDICT_CONFIG, type Verdict } from '@/types'

interface UserProfile {
  user_id: string
  username: string
  display_name: string
  rating: number
  solved_count: number
  submission_count: number
  avatar_url: string
  bio: string
  country: string
  created_at: string
  updated_at: string
}

interface UserStats {
  user_id: string
  solved_count: number
  submission_count: number
  rating: number
  accepted_count: number
  wrong_answer_count: number
  time_limit_count: number
  memory_limit_count: number
  runtime_error_count: number
  compile_error_count: number
  acceptance_rate: number
}

interface UserSubmission {
  id: string
  problem_id: string
  problem_name: string
  language_id: string
  verdict: string
  runtime: number
  memory: number
  submit_time: string
  contest_id: string
  contest_name: string
}

interface SubmissionsResponse {
  submissions: UserSubmission[]
  pagination: {
    total: number
    page: number
    page_size: number
  }
}

// Helper functions
function getVerdictKey(verdict: string): string {
  if (!verdict) return ''
  const str = String(verdict)
  if (str.startsWith('VERDICT_')) {
    return str.replace('VERDICT_', '').toLowerCase().replace('_', '-')
  }
  return str
}

function formatRuntime(seconds: number): string {
  if (!seconds) return '-'
  return `${seconds.toFixed(3)}s`
}

function formatMemory(kb: number): string {
  if (!kb) return '-'
  return `${(kb / 1024).toFixed(2)} MB`
}

function formatTime(timeStr: string): string {
  if (!timeStr) return '-'
  return new Date(timeStr).toLocaleString()
}

function formatRelativeTime(timeStr: string): string {
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

// Rating badge component
function RatingBadge({ rating }: { rating: number }) {
  let color = 'bg-gray-500'
  let label = 'Newbie'

  if (rating >= 2000) {
    color = 'bg-red-500'
    label = 'Master'
  } else if (rating >= 1600) {
    color = 'bg-orange-500'
    label = 'Expert'
  } else if (rating >= 1200) {
    color = 'bg-purple-500'
    label = 'Specialist'
  } else if (rating >= 800) {
    color = 'bg-blue-500'
    label = 'Pupil'
  }

  return (
    <span className={`${color} text-white px-3 py-1 rounded-full text-sm font-medium`}>
      {label} ({rating})
    </span>
  )
}

// Stats card component
function StatsCard({ stats }: { stats: UserStats | null }) {
  if (!stats) return null

  return (
    <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-6">
      <div className="bg-white dark:bg-gray-800 rounded-lg shadow p-4">
        <div className="text-sm text-gray-600 dark:text-gray-400">Solved</div>
        <div className="text-2xl font-bold text-green-600 dark:text-green-400">{stats.solved_count}</div>
      </div>
      <div className="bg-white dark:bg-gray-800 rounded-lg shadow p-4">
        <div className="text-sm text-gray-600 dark:text-gray-400">Submissions</div>
        <div className="text-2xl font-bold text-gray-900 dark:text-gray-100">{stats.submission_count}</div>
      </div>
      <div className="bg-white dark:bg-gray-800 rounded-lg shadow p-4">
        <div className="text-sm text-gray-600 dark:text-gray-400">Acceptance Rate</div>
        <div className="text-2xl font-bold text-blue-600 dark:text-blue-400">
          {(stats.acceptance_rate * 100).toFixed(1)}%
        </div>
      </div>
      <div className="bg-white dark:bg-gray-800 rounded-lg shadow p-4">
        <div className="text-sm text-gray-600 dark:text-gray-400">Rating</div>
        <div className="text-2xl font-bold text-gray-900 dark:text-gray-100">{stats.rating}</div>
      </div>
    </div>
  )
}

// Verdict breakdown chart
function VerdictBreakdown({ stats }: { stats: UserStats | null }) {
  if (!stats || stats.submission_count === 0) return null

  const verdicts = [
    { label: 'AC', count: stats.accepted_count, color: 'bg-green-500' },
    { label: 'WA', count: stats.wrong_answer_count, color: 'bg-red-500' },
    { label: 'TLE', count: stats.time_limit_count, color: 'bg-yellow-500' },
    { label: 'MLE', count: stats.memory_limit_count, color: 'bg-orange-500' },
    { label: 'RE', count: stats.runtime_error_count, color: 'bg-purple-500' },
    { label: 'CE', count: stats.compile_error_count, color: 'bg-blue-500' },
  ].filter(v => v.count > 0)

  return (
    <div className="bg-white dark:bg-gray-800 rounded-lg shadow p-4 mb-6">
      <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100 mb-3">Verdict Breakdown</h3>
      <div className="flex gap-2 flex-wrap">
        {verdicts.map(v => (
          <div key={v.label} className="flex items-center gap-2">
            <div className={`${v.color} w-4 h-4 rounded`}></div>
            <span className="text-sm text-gray-700 dark:text-gray-300">{v.label}: {v.count}</span>
          </div>
        ))}
      </div>
    </div>
  )
}

// Submission history table
function SubmissionHistory({
  submissions,
  isLoading,
  error,
  pagination,
  onPageChange
}: {
  submissions: UserSubmission[]
  isLoading: boolean
  error: Error | null
  pagination: { total: number; page: number; page_size: number }
  onPageChange: (page: number) => void
}) {
  if (error) {
    return (
      <div className="bg-white dark:bg-gray-800 rounded-lg shadow p-4 mb-6">
        <div className="text-red-400">Error loading submissions: {error.message}</div>
      </div>
    )
  }

  if (isLoading) {
    return (
      <div className="bg-white dark:bg-gray-800 rounded-lg shadow p-4 mb-6">
        <div className="text-gray-600 dark:text-gray-400">Loading submissions...</div>
      </div>
    )
  }

  if (submissions.length === 0) {
    return (
      <div className="bg-white dark:bg-gray-800 rounded-lg shadow p-4 mb-6">
        <div className="text-gray-500 dark:text-gray-400 text-center">
          No submissions yet.
        </div>
      </div>
    )
  }

  const totalPages = Math.ceil(pagination.total / pagination.page_size)

  return (
    <div className="bg-white dark:bg-gray-800 rounded-lg shadow overflow-hidden mb-6">
      <div className="p-4 border-b border-gray-200 dark:border-gray-700">
        <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100">Recent Submissions</h3>
      </div>
      <div className="overflow-x-auto">
        <table className="min-w-full">
          <thead className="bg-gray-100 dark:bg-gray-700">
            <tr>
              <th className="px-4 py-3 text-left text-sm font-medium text-gray-700 dark:text-gray-300">ID</th>
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
              const verdictInfo = verdictKey ? VERDICT_CONFIG[verdictKey as Verdict] : null
              return (
                <tr key={sub.id} className="hover:bg-gray-50 dark:hover:bg-gray-700/50">
                  <td className="px-4 py-3 text-sm">
                    <Link href={`/submissions/${sub.id}`} className="text-blue-600 dark:text-blue-400 hover:underline font-mono">
                      {sub.id.slice(0, 8)}
                    </Link>
                  </td>
                  <td className="px-4 py-3 text-sm">
                    <Link href={`/problems/${sub.problem_id}`} className="text-blue-600 dark:text-blue-400 hover:underline">
                      {sub.problem_name || sub.problem_id.slice(0, 8)}
                    </Link>
                  </td>
                  <td className="px-4 py-3 text-sm text-gray-700 dark:text-gray-300 capitalize">{sub.language_id}</td>
                  <td className="px-4 py-3 text-sm">
                    <span className={`px-2 py-1 rounded text-white text-xs font-medium ${verdictInfo?.color || 'bg-gray-500'}`}>
                      {verdictInfo?.label || 'Pending'}
                    </span>
                  </td>
                  <td className="px-4 py-3 text-sm text-gray-700 dark:text-gray-300">{formatRuntime(sub.runtime)}</td>
                  <td className="px-4 py-3 text-sm text-gray-700 dark:text-gray-300">{formatMemory(sub.memory)}</td>
                  <td className="px-4 py-3 text-sm text-gray-600 dark:text-gray-400">{formatRelativeTime(sub.submit_time)}</td>
                </tr>
              )
            })}
          </tbody>
        </table>
      </div>
      {totalPages > 1 && (
        <div className="p-4 border-t border-gray-200 dark:border-gray-700 flex justify-center gap-2">
          <button
            onClick={() => onPageChange(pagination.page - 1)}
            disabled={pagination.page <= 1}
            className="px-3 py-1 rounded bg-gray-200 dark:bg-gray-600 disabled:opacity-50 text-gray-700 dark:text-gray-200"
          >
            Prev
          </button>
          <span className="px-3 py-1 text-gray-700 dark:text-gray-300">
            Page {pagination.page} of {totalPages}
          </span>
          <button
            onClick={() => onPageChange(pagination.page + 1)}
            disabled={pagination.page >= totalPages}
            className="px-3 py-1 rounded bg-gray-200 dark:bg-gray-600 disabled:opacity-50 text-gray-700 dark:text-gray-200"
          >
            Next
          </button>
        </div>
      )}
    </div>
  )
}

export default function UserPage() {
  const params = useParams()
  const userId = params.id as string

  const [submissionPage, setSubmissionPage] = useState(1)

  // Fetch profile data
  const { data: profileData, isLoading: profileLoading, error: profileError } = useUserProfile(userId) as {
    data: UserProfile | undefined
    isLoading: boolean
    error: Error | null
  }

  const { data: statsData, isLoading: statsLoading } = useUserStats(userId) as {
    data: UserStats | undefined
    isLoading: boolean
  }

  const { data: submissionsData, isLoading: submissionsLoading, error: submissionsError } = useUserSubmissions(userId, submissionPage, 10) as {
    data: SubmissionsResponse | undefined
    isLoading: boolean
    error: Error | null
  }

  if (profileError) {
    return (
      <div className="px-4 py-6">
        <h1 className="text-2xl font-bold mb-6 text-gray-900 dark:text-gray-100">User Profile</h1>
        <div className="text-center py-10 text-red-400">
          Error loading profile: {profileError.message}
        </div>
      </div>
    )
  }

  if (profileLoading) {
    return (
      <div className="px-4 py-6">
        <h1 className="text-2xl font-bold mb-6 text-gray-900 dark:text-gray-100">User Profile</h1>
        <div className="text-center py-10 text-gray-600 dark:text-gray-400">Loading...</div>
      </div>
    )
  }

  if (!profileData) {
    return (
      <div className="px-4 py-6">
        <h1 className="text-2xl font-bold mb-6 text-gray-900 dark:text-gray-100">User Profile</h1>
        <div className="text-center py-10 text-gray-500">User not found</div>
      </div>
    )
  }

  const profile = profileData

  return (
    <div className="px-4 py-6">
      {/* Profile header */}
      <div className="bg-white dark:bg-gray-800 rounded-lg shadow p-6 mb-6">
        <div className="flex items-start gap-4">
          {/* Avatar */}
          <div className="w-16 h-16 rounded-full bg-blue-600 flex items-center justify-center text-white text-2xl font-bold">
            {profile.username?.[0]?.toUpperCase() || 'U'}
          </div>
          <div>
            <h1 className="text-2xl font-bold text-gray-900 dark:text-gray-100">
              {profile.display_name || profile.username}
            </h1>
            <div className="text-gray-600 dark:text-gray-400">@{profile.username}</div>
            {profile.rating > 0 && (
              <div className="mt-2">
                <RatingBadge rating={profile.rating} />
              </div>
            )}
            {profile.bio && (
              <div className="mt-2 text-gray-700 dark:text-gray-300">{profile.bio}</div>
            )}
            {profile.country && (
              <div className="mt-1 text-sm text-gray-500 dark:text-gray-400 flex items-center gap-1">
                <span>📍</span> {profile.country}
              </div>
            )}
          </div>
        </div>
        <div className="mt-4 pt-4 border-t border-gray-200 dark:border-gray-700 text-sm text-gray-500 dark:text-gray-400">
          Member since {formatTime(profile.created_at)}
        </div>
      </div>

      {/* Stats */}
      <h2 className="text-xl font-semibold text-gray-900 dark:text-gray-100 mb-4">Statistics</h2>
      {!statsLoading && <StatsCard stats={statsData || null} />}
      {!statsLoading && <VerdictBreakdown stats={statsData || null} />}

      {/* Submission history */}
      <h2 className="text-xl font-semibold text-gray-900 dark:text-gray-100 mb-4">Submission History</h2>
      <SubmissionHistory
        submissions={submissionsData?.submissions || []}
        isLoading={submissionsLoading}
        error={submissionsError}
        pagination={submissionsData?.pagination || { total: 0, page: 1, page_size: 10 }}
        onPageChange={(page) => setSubmissionPage(page)}
      />
    </div>
  )
}