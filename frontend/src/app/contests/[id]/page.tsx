'use client'

import { useParams } from 'next/navigation'
import Link from 'next/link'
import { useState } from 'react'
import { useContest, useScoreboard } from '@/hooks/useApi'
import { CountdownTimer } from '@/components/ui/CountdownTimer'
import { ScoreboardTable } from '@/components/contest/ScoreboardTable'
import { RegistrationModal } from '@/components/contest/RegistrationModal'
import { useAuthStore } from '@/stores/authStore'

const BFF_URL = process.env.NEXT_PUBLIC_BFF_URL || ''

interface Contest {
  id: string
  external_id: string
  name: string
  short_name: string
  start_time: string
  end_time: string
  freeze_time: string
  unfreeze_time: string
  public: boolean
  created_at: string
}

interface ContestProblem {
  problem_id: string
  short_name: string
  rank: number
  color: string
  points: number
  allow_submit: boolean
}

interface ContestResponse {
  contest: Contest
  problems: ContestProblem[]
}

interface ScoreboardResponse {
  entries: Array<{
    rank: number
    team_id: string
    team_name: string
    affiliation: string
    num_solved: number
    total_time: number
    problems: Array<{
      problem_short_name: string
      num_pending: number
      num_correct: number
      time: number
      is_pending: boolean
    }>
  }>
  contest_time: string
  is_frozen: boolean
}

// Contest status calculation
function getContestStatus(contest: Contest) {
  const now = new Date()
  const start = new Date(contest.start_time)
  const end = new Date(contest.end_time)

  if (now < start) {
    return {
      label: 'Upcoming',
      color: 'bg-yellow-100 text-yellow-800 border-yellow-300 dark:bg-yellow-500/20 dark:text-yellow-300 dark:border-yellow-500/50',
      isUpcoming: true,
    }
  }
  if (now > end) {
    return {
      label: 'Ended',
      color: 'bg-muted text-foreground border-border dark:bg-muted/40',
      isEnded: true,
    }
  }
  return {
    label: 'Running',
    color: 'bg-green-100 text-green-800 border-green-300 dark:bg-green-500/20 dark:text-green-300 dark:border-green-500/50',
    isRunning: true,
  }
}

export default function ContestDetailPage() {
  const params = useParams()
  const contestId = params.id as string
  const { isAuthenticated } = useAuthStore()

  const [showRegisterModal, setShowRegisterModal] = useState(false)
  const [isRegistered, setIsRegistered] = useState(false)

  const { data: contestData, isLoading: contestLoading, error: contestError } = useContest(contestId) as {
    data: ContestResponse | undefined
    isLoading: boolean
    error: Error | null
  }

  const { data: scoreboardData, isLoading: scoreboardLoading } = useScoreboard(contestId) as {
    data: ScoreboardResponse | undefined
    isLoading: boolean
  }

  const contest = contestData?.contest
  const problems = contestData?.problems || []
  const status = contest ? getContestStatus(contest) : null

  // Get problem short names for scoreboard columns
  const problemNames = problems.map((p) => p.short_name).sort((a, b) => a.localeCompare(b))

  const handleRegisterSuccess = () => {
    setIsRegistered(true)
  }

  if (contestError) {
    return (
      <div className="px-4 py-6">
        <h1 className="text-2xl font-bold mb-6 text-foreground">Contest</h1>
        <div className="text-center py-10 text-red-600 dark:text-red-400">
          Error loading contest: {contestError.message}
        </div>
      </div>
    )
  }

  if (contestLoading) {
    return (
      <div className="px-4 py-6">
        <h1 className="text-2xl font-bold mb-6 text-foreground">Contest</h1>
        <div className="text-center py-10 text-muted-foreground">Loading contest...</div>
      </div>
    )
  }

  if (!contest) {
    return (
      <div className="px-4 py-6">
        <h1 className="text-2xl font-bold mb-6 text-foreground">Contest</h1>
        <div className="text-center py-10 text-muted-foreground">Contest not found</div>
      </div>
    )
  }

  const formatDateTime = (dateStr: string) => {
    if (!dateStr) return '-'
    return new Date(dateStr).toLocaleString()
  }

  const formatDuration = (startTime: string, endTime: string) => {
    if (!startTime || !endTime) return '-'
    const start = new Date(startTime)
    const end = new Date(endTime)
    const diffMs = end.getTime() - start.getTime()
    const hours = Math.floor(diffMs / 3600000)
    const mins = Math.floor((diffMs % 3600000) / 60000)
    if (hours >= 24) {
      const days = Math.floor(hours / 24)
      return `${days}d ${hours % 24}h ${mins}m`
    }
    return `${hours}h ${mins}m`
  }

  return (
    <div className="px-4 py-6">
      {/* Header */}
      <div className="mb-6">
        <div className="flex items-center gap-2 mb-2">
          <Link href="/contests" className="text-primary hover:underline">
            Contests
          </Link>
          <span className="text-muted-foreground">/</span>
          <span className="text-foreground">{contest.short_name}</span>
        </div>
      </div>

      {/* Contest Info Card */}
      <div className="bg-card rounded-xl shadow mb-6 border border-border">
        <div className="p-6">
          <div className="flex items-start justify-between mb-4">
            <div>
              <h1 className="text-2xl font-bold text-foreground mb-1">{contest.name}</h1>
              <p className="text-muted-foreground">{contest.short_name}</p>
            </div>
            <div className="flex items-center gap-3">
              {status && (
                <span className={`px-3 py-1 rounded text-sm font-medium border ${status.color}`}>
                  {status.label}
                </span>
              )}
              {!contest.public && (
                <span className="text-xs text-muted-foreground bg-muted px-2 py-1 rounded">Private</span>
              )}
            </div>
          </div>

          {/* Time info and countdown */}
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-4">
            <div>
              <label className="text-muted-foreground text-xs uppercase tracking-wide">Start</label>
              <div className="text-foreground">{formatDateTime(contest.start_time)}</div>
            </div>
            <div>
              <label className="text-muted-foreground text-xs uppercase tracking-wide">End</label>
              <div className="text-foreground">{formatDateTime(contest.end_time)}</div>
            </div>
            <div>
              <label className="text-muted-foreground text-xs uppercase tracking-wide">Duration</label>
              <div className="text-foreground">{formatDuration(contest.start_time, contest.end_time)}</div>
            </div>
            <div>
              <label className="text-muted-foreground text-xs uppercase tracking-wide">
                {status?.isRunning ? 'Remaining' : status?.isUpcoming ? 'Starts in' : 'Ended'}
              </label>
              <div className="text-foreground">
                {status?.isRunning && (
                  <CountdownTimer targetTime={contest.end_time} showDays={true} />
                )}
                {status?.isUpcoming && (
                  <CountdownTimer targetTime={contest.start_time} showDays={true} />
                )}
                {status?.isEnded && <span className="text-muted-foreground">-</span>}
              </div>
            </div>
          </div>

          {/* Registration button */}
          {isAuthenticated && !isRegistered && status && !status.isEnded && (
            <button
              onClick={() => setShowRegisterModal(true)}
              className="mt-4 px-4 py-2 bg-primary text-white rounded-md hover:bg-primary/90 transition-colors"
            >
              Register for Contest
            </button>
          )}
          {isRegistered && (
            <div className="mt-4 px-4 py-2 bg-green-100 text-green-800 dark:bg-green-600/20 dark:text-green-300 rounded-md inline-block">
              ✓ You are registered
            </div>
          )}
        </div>
      </div>

      {/* Problems Section */}
      <div className="bg-card rounded-xl shadow mb-6 border border-border">
        <div className="p-4 border-b border-border">
          <h2 className="text-lg font-semibold text-foreground">Problems</h2>
        </div>
        <div className="p-4">
          {problems.length === 0 ? (
            <div className="text-center py-6 text-muted-foreground">No problems available</div>
          ) : (
            <div className="grid grid-cols-2 md:grid-cols-4 lg:grid-cols-6 gap-3">
              {problems
                .sort((a, b) => a.rank - b.rank)
                .map((problem) => (
                  <Link
                    key={problem.problem_id}
                    href={`/problems/${problem.problem_id}?contest=${contestId}`}
                    className={`block p-4 rounded-xl border transition-all hover:shadow-sm ${
                      problem.allow_submit
                        ? 'bg-muted/40 border-border hover:border-primary hover:bg-muted'
                        : 'bg-muted border-border opacity-50 cursor-not-allowed'
                    }`}
                  >
                    <div className="flex items-center gap-2 mb-2">
                      {problem.color && (
                        <div
                          className="w-4 h-4 rounded"
                          style={{ backgroundColor: problem.color }}
                        />
                      )}
                      <span className="text-lg font-bold text-foreground">
                        {problem.short_name}
                      </span>
                    </div>
                    <div className="text-sm text-muted-foreground">
                      {problem.points} points
                    </div>
                    {!problem.allow_submit && (
                      <div className="text-xs text-red-400 mt-1">Not available</div>
                    )}
                  </Link>
                ))}
            </div>
          )}
        </div>
      </div>

      {/* Scoreboard Section */}
      <div className="bg-card rounded-xl shadow border border-border">
        <div className="p-4 border-b border-border">
          <div className="flex items-center justify-between">
            <h2 className="text-lg font-semibold text-foreground">Scoreboard</h2>
            <span className="text-xs text-muted-foreground">Auto-updates every 5s</span>
          </div>
        </div>
        <div className="p-4">
          <ScoreboardTable
            data={scoreboardData}
            isLoading={scoreboardLoading}
            problemNames={problemNames}
          />
        </div>
      </div>

      {/* Registration Modal */}
      {showRegisterModal && (
        <RegistrationModal
          contestId={contestId}
          contestName={contest.name}
          isOpen={showRegisterModal}
          onClose={() => setShowRegisterModal(false)}
          onSuccess={handleRegisterSuccess}
        />
      )}
    </div>
  )
}