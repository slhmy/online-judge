'use client'

import { useParams } from 'next/navigation'
import Link from 'next/link'
import { useState } from 'react'
import { useContest, useScoreboard } from '@/hooks/useApi'
import { CountdownTimer } from '@/components/ui/CountdownTimer'
import { ScoreboardTable } from '@/components/contest/ScoreboardTable'
import { RegistrationModal } from '@/components/contest/RegistrationModal'
import { useAuthStore } from '@/stores/authStore'

const BFF_URL = process.env.NEXT_PUBLIC_BFF_URL || 'http://localhost:8080'

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
    return { label: 'Upcoming', color: 'bg-yellow-500/20 text-yellow-400 border-yellow-500/50', isUpcoming: true }
  }
  if (now > end) {
    return { label: 'Ended', color: 'bg-gray-500/20 text-gray-400 border-gray-500/50', isEnded: true }
  }
  return { label: 'Running', color: 'bg-green-500/20 text-green-400 border-green-500/50', isRunning: true }
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
        <h1 className="text-2xl font-bold mb-6 text-gray-100">Contest</h1>
        <div className="text-center py-10 text-red-400">
          Error loading contest: {contestError.message}
        </div>
      </div>
    )
  }

  if (contestLoading) {
    return (
      <div className="px-4 py-6">
        <h1 className="text-2xl font-bold mb-6 text-gray-100">Contest</h1>
        <div className="text-center py-10 text-gray-400">Loading contest...</div>
      </div>
    )
  }

  if (!contest) {
    return (
      <div className="px-4 py-6">
        <h1 className="text-2xl font-bold mb-6 text-gray-100">Contest</h1>
        <div className="text-center py-10 text-gray-500">Contest not found</div>
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
          <Link href="/contests" className="text-blue-400 hover:underline">
            Contests
          </Link>
          <span className="text-gray-500">/</span>
          <span className="text-gray-300">{contest.short_name}</span>
        </div>
      </div>

      {/* Contest Info Card */}
      <div className="bg-gray-800 rounded-lg shadow mb-6 border border-gray-700">
        <div className="p-6">
          <div className="flex items-start justify-between mb-4">
            <div>
              <h1 className="text-2xl font-bold text-gray-100 mb-1">{contest.name}</h1>
              <p className="text-gray-400">{contest.short_name}</p>
            </div>
            <div className="flex items-center gap-3">
              {status && (
                <span className={`px-3 py-1 rounded text-sm font-medium border ${status.color}`}>
                  {status.label}
                </span>
              )}
              {!contest.public && (
                <span className="text-xs text-gray-500 bg-gray-700 px-2 py-1 rounded">Private</span>
              )}
            </div>
          </div>

          {/* Time info and countdown */}
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-4">
            <div>
              <label className="text-gray-500 text-xs uppercase tracking-wide">Start</label>
              <div className="text-gray-200">{formatDateTime(contest.start_time)}</div>
            </div>
            <div>
              <label className="text-gray-500 text-xs uppercase tracking-wide">End</label>
              <div className="text-gray-200">{formatDateTime(contest.end_time)}</div>
            </div>
            <div>
              <label className="text-gray-500 text-xs uppercase tracking-wide">Duration</label>
              <div className="text-gray-200">{formatDuration(contest.start_time, contest.end_time)}</div>
            </div>
            <div>
              <label className="text-gray-500 text-xs uppercase tracking-wide">
                {status?.isRunning ? 'Remaining' : status?.isUpcoming ? 'Starts in' : 'Ended'}
              </label>
              <div className="text-gray-200">
                {status?.isRunning && (
                  <CountdownTimer targetTime={contest.end_time} showDays={true} />
                )}
                {status?.isUpcoming && (
                  <CountdownTimer targetTime={contest.start_time} showDays={true} />
                )}
                {status?.isEnded && <span className="text-gray-500">-</span>}
              </div>
            </div>
          </div>

          {/* Registration button */}
          {isAuthenticated && !isRegistered && status && !status.isEnded && (
            <button
              onClick={() => setShowRegisterModal(true)}
              className="mt-4 px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 transition-colors"
            >
              Register for Contest
            </button>
          )}
          {isRegistered && (
            <div className="mt-4 px-4 py-2 bg-green-600/20 text-green-400 rounded-md inline-block">
              ✓ You are registered
            </div>
          )}
        </div>
      </div>

      {/* Problems Section */}
      <div className="bg-gray-800 rounded-lg shadow mb-6 border border-gray-700">
        <div className="p-4 border-b border-gray-700">
          <h2 className="text-lg font-semibold text-gray-100">Problems</h2>
        </div>
        <div className="p-4">
          {problems.length === 0 ? (
            <div className="text-center py-6 text-gray-500">No problems available</div>
          ) : (
            <div className="grid grid-cols-2 md:grid-cols-4 lg:grid-cols-6 gap-3">
              {problems
                .sort((a, b) => a.rank - b.rank)
                .map((problem) => (
                  <Link
                    key={problem.problem_id}
                    href={`/problems/${problem.problem_id}?contest=${contestId}`}
                    className={`block p-4 rounded-lg border transition-all hover:shadow-md ${
                      problem.allow_submit
                        ? 'bg-gray-750 border-gray-600 hover:border-blue-500 hover:bg-gray-700'
                        : 'bg-gray-700 border-gray-700 opacity-50 cursor-not-allowed'
                    }`}
                  >
                    <div className="flex items-center gap-2 mb-2">
                      {problem.color && (
                        <div
                          className="w-4 h-4 rounded"
                          style={{ backgroundColor: problem.color }}
                        />
                      )}
                      <span className="text-lg font-bold text-gray-100">
                        {problem.short_name}
                      </span>
                    </div>
                    <div className="text-sm text-gray-400">
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
      <div className="bg-gray-800 rounded-lg shadow border border-gray-700">
        <div className="p-4 border-b border-gray-700">
          <div className="flex items-center justify-between">
            <h2 className="text-lg font-semibold text-gray-100">Scoreboard</h2>
            <span className="text-xs text-gray-500">Auto-updates every 5s</span>
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