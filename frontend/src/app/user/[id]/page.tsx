'use client'

import { useState } from 'react'
import { useParams } from 'next/navigation'
import Link from 'next/link'
import { useUserProfile, useUserStats, useUserSubmissions } from '@/hooks/useApi'
import { Badge } from '@/components/ui/badge'
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

function getVerdictBadgeClass(verdict: Verdict | null): string {
  if (!verdict) return 'bg-muted text-foreground'
  const config = VERDICT_CONFIG[verdict]
  return config?.color ? `${config.color} text-white hover:opacity-90` : 'bg-muted text-foreground'
}

function getVerdictBadgeVariant(verdict: Verdict | null): 'default' | 'destructive' {
  return verdict === 'correct' ? 'default' : 'destructive'
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
  let color = 'bg-muted'
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
    color = 'bg-primary'
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
      <div className="bg-card rounded-xl shadow p-4">
        <div className="text-sm text-muted-foreground">Solved</div>
        <div className="text-2xl font-bold text-green-600 dark:text-green-400">{stats.solved_count}</div>
      </div>
      <div className="bg-card rounded-xl shadow p-4">
        <div className="text-sm text-muted-foreground">Submissions</div>
        <div className="text-2xl font-bold text-foreground">{stats.submission_count}</div>
      </div>
      <div className="bg-card rounded-xl shadow p-4">
        <div className="text-sm text-muted-foreground">Acceptance Rate</div>
        <div className="text-2xl font-bold text-primary">
          {(stats.acceptance_rate * 100).toFixed(1)}%
        </div>
      </div>
      <div className="bg-card rounded-xl shadow p-4">
        <div className="text-sm text-muted-foreground">Rating</div>
        <div className="text-2xl font-bold text-foreground">{stats.rating}</div>
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
    { label: 'CE', count: stats.compile_error_count, color: 'bg-primary' },
  ].filter(v => v.count > 0)

  return (
    <div className="bg-card rounded-xl shadow p-4 mb-6">
      <h3 className="text-lg font-semibold text-foreground mb-3">Verdict Breakdown</h3>
      <div className="flex gap-2 flex-wrap">
        {verdicts.map(v => (
          <div key={v.label} className="flex items-center gap-2">
            <div className={`${v.color} w-4 h-4 rounded`}></div>
            <span className="text-sm text-foreground">{v.label}: {v.count}</span>
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
      <div className="bg-card rounded-xl shadow p-4 mb-6">
        <div className="text-red-400">Error loading submissions: {error.message}</div>
      </div>
    )
  }

  if (isLoading) {
    return (
      <div className="bg-card rounded-xl shadow p-4 mb-6">
        <div className="text-muted-foreground">Loading submissions...</div>
      </div>
    )
  }

  if (submissions.length === 0) {
    return (
      <div className="bg-card rounded-xl shadow p-4 mb-6">
        <div className="text-muted-foreground text-center">
          No submissions yet.
        </div>
      </div>
    )
  }

  const totalPages = Math.ceil(pagination.total / pagination.page_size)

  return (
    <div className="bg-card rounded-xl shadow overflow-hidden mb-6">
      <div className="p-4 border-b border-border">
        <h3 className="text-lg font-semibold text-foreground">Recent Submissions</h3>
      </div>
      <div className="overflow-x-auto">
        <table className="min-w-full">
          <thead className="bg-muted">
            <tr>
              <th className="px-4 py-3 text-left text-sm font-medium text-foreground">ID</th>
              <th className="px-4 py-3 text-left text-sm font-medium text-foreground">Problem</th>
              <th className="px-4 py-3 text-left text-sm font-medium text-foreground">Language</th>
              <th className="px-4 py-3 text-left text-sm font-medium text-foreground">Verdict</th>
              <th className="px-4 py-3 text-left text-sm font-medium text-foreground">Time</th>
              <th className="px-4 py-3 text-left text-sm font-medium text-foreground">Memory</th>
              <th className="px-4 py-3 text-left text-sm font-medium text-foreground">Submitted</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-200 dark:divide-gray-700">
            {submissions.map((sub) => {
              const verdictKey = getVerdictKey(sub.verdict)
              const verdictInfo = verdictKey ? VERDICT_CONFIG[verdictKey as Verdict] : null
              return (
                <tr key={sub.id} className="hover:bg-muted/60">
                  <td className="px-4 py-3 text-sm">
                    <Link href={`/submissions/${sub.id}`} className="text-primary hover:underline font-mono">
                      {sub.id.slice(0, 8)}
                    </Link>
                  </td>
                  <td className="px-4 py-3 text-sm">
                    <Link href={`/problems/${sub.problem_id}`} className="text-primary hover:underline">
                      {sub.problem_name || sub.problem_id.slice(0, 8)}
                    </Link>
                  </td>
                  <td className="px-4 py-3 text-sm text-foreground capitalize">{sub.language_id}</td>
                  <td className="px-4 py-3 text-sm">
                    <Badge
                      variant={getVerdictBadgeVariant(verdictKey as Verdict)}
                      className={getVerdictBadgeClass(verdictKey as Verdict)}
                    >
                      {verdictInfo?.label || 'Pending'}
                    </Badge>
                  </td>
                  <td className="px-4 py-3 text-sm text-foreground">{formatRuntime(sub.runtime)}</td>
                  <td className="px-4 py-3 text-sm text-foreground">{formatMemory(sub.memory)}</td>
                  <td className="px-4 py-3 text-sm text-muted-foreground">{formatRelativeTime(sub.submit_time)}</td>
                </tr>
              )
            })}
          </tbody>
        </table>
      </div>
      {totalPages > 1 && (
        <div className="p-4 border-t border-border flex justify-center gap-2">
          <button
            onClick={() => onPageChange(pagination.page - 1)}
            disabled={pagination.page <= 1}
            className="px-3 py-1 rounded bg-muted disabled:opacity-50 text-foreground"
          >
            Prev
          </button>
          <span className="px-3 py-1 text-foreground">
            Page {pagination.page} of {totalPages}
          </span>
          <button
            onClick={() => onPageChange(pagination.page + 1)}
            disabled={pagination.page >= totalPages}
            className="px-3 py-1 rounded bg-muted disabled:opacity-50 text-foreground"
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
        <h1 className="text-2xl font-bold mb-6 text-foreground">User Profile</h1>
        <div className="text-center py-10 text-red-400">
          Error loading profile: {profileError.message}
        </div>
      </div>
    )
  }

  if (profileLoading) {
    return (
      <div className="px-4 py-6">
        <h1 className="text-2xl font-bold mb-6 text-foreground">User Profile</h1>
        <div className="text-center py-10 text-muted-foreground">Loading...</div>
      </div>
    )
  }

  if (!profileData) {
    return (
      <div className="px-4 py-6">
        <h1 className="text-2xl font-bold mb-6 text-foreground">User Profile</h1>
        <div className="text-center py-10 text-muted-foreground">User not found</div>
      </div>
    )
  }

  const profile = profileData

  return (
    <div className="px-4 py-6">
      {/* Profile header */}
      <div className="bg-card rounded-xl shadow p-6 mb-6">
        <div className="flex items-start gap-4">
          {/* Avatar */}
          <div className="w-16 h-16 rounded-full bg-primary flex items-center justify-center text-white text-2xl font-bold">
            {profile.username?.[0]?.toUpperCase() || 'U'}
          </div>
          <div>
            <h1 className="text-2xl font-bold text-foreground">
              {profile.display_name || profile.username}
            </h1>
            <div className="text-muted-foreground">@{profile.username}</div>
            {profile.rating > 0 && (
              <div className="mt-2">
                <RatingBadge rating={profile.rating} />
              </div>
            )}
            {profile.bio && (
              <div className="mt-2 text-foreground">{profile.bio}</div>
            )}
            {profile.country && (
              <div className="mt-1 text-sm text-muted-foreground flex items-center gap-1">
                <span>📍</span> {profile.country}
              </div>
            )}
          </div>
        </div>
        <div className="mt-4 pt-4 border-t border-border text-sm text-muted-foreground">
          Member since {formatTime(profile.created_at)}
        </div>
      </div>

      {/* Stats */}
      <h2 className="text-xl font-semibold text-foreground mb-4">Statistics</h2>
      {!statsLoading && <StatsCard stats={statsData || null} />}
      {!statsLoading && <VerdictBreakdown stats={statsData || null} />}

      {/* Submission history */}
      <h2 className="text-xl font-semibold text-foreground mb-4">Submission History</h2>
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