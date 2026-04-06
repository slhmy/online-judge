'use client'

import { cn } from '@/lib/utils'

interface ProblemScore {
  problem_short_name: string
  num_pending: number
  num_correct: number
  time: number
  is_pending: boolean
}

interface ScoreboardEntry {
  rank: number
  team_id: string
  team_name: string
  affiliation: string
  num_solved: number
  total_time: number
  problems: ProblemScore[]
}

interface ScoreboardResponse {
  entries: ScoreboardEntry[]
  contest_time: string
  is_frozen: boolean
}

interface ScoreboardTableProps {
  data: ScoreboardResponse | undefined
  isLoading: boolean
  problemNames: string[] // Short names like ['A', 'B', 'C']
}

// Problem cell component for ICPC-style scoreboard
function ProblemCell({ score }: { score: ProblemScore | undefined }) {
  if (!score) {
    return (
      <td className="px-2 py-1 text-center bg-gray-100 dark:bg-gray-700">
        <span className="text-gray-400">-</span>
      </td>
    )
  }

  // Solved: green with checkmark and time
  if (score.num_correct === 1) {
    return (
      <td className="px-2 py-1 text-center">
        <div className="bg-green-500 dark:bg-green-600 rounded px-1 py-0.5 text-white">
          <div className="flex items-center justify-center gap-1">
            <span className="text-xs font-bold">✓</span>
            <span className="text-xs">{score.time}</span>
          </div>
          {score.num_pending > 0 && (
            <div className="text-xs text-green-200">+{score.num_pending}</div>
          )}
        </div>
      </td>
    )
  }

  // Pending: yellow/orange with attempt count
  if (score.is_pending || score.num_pending > 0) {
    return (
      <td className="px-2 py-1 text-center">
        <div className="bg-yellow-500 dark:bg-yellow-600 rounded px-1 py-0.5 text-white">
          <div className="flex items-center justify-center">
            <span className="text-xs font-bold">{score.num_pending}</span>
            <span className="text-xs ml-1">→</span>
          </div>
        </div>
      </td>
    )
  }

  // Not attempted: gray
  return (
    <td className="px-2 py-1 text-center bg-gray-100 dark:bg-gray-700">
      <span className="text-gray-400">-</span>
    </td>
  )
}

export function ScoreboardTable({ data, isLoading, problemNames }: ScoreboardTableProps) {
  if (isLoading) {
    return (
      <div className="text-center py-8 text-gray-400">
        Loading scoreboard...
      </div>
    )
  }

  if (!data || !data.entries || data.entries.length === 0) {
    return (
      <div className="text-center py-8 text-gray-500">
        No scoreboard entries yet
      </div>
    )
  }

  const entries = data.entries

  return (
    <div className="overflow-x-auto">
      {/* Frozen indicator */}
      {data.is_frozen && (
        <div className="mb-2 px-3 py-1 bg-blue-100 dark:bg-blue-900 text-blue-700 dark:text-blue-300 rounded inline-flex items-center gap-2">
          <span className="animate-pulse">❄️</span>
          <span className="text-sm font-medium">Scoreboard Frozen</span>
        </div>
      )}

      <table className="min-w-full border-collapse">
        <thead>
          <tr className="bg-gray-200 dark:bg-gray-700">
            <th className="px-3 py-2 text-left text-sm font-medium text-gray-700 dark:text-gray-300 border-b border-gray-300 dark:border-gray-600">
              Rank
            </th>
            <th className="px-3 py-2 text-left text-sm font-medium text-gray-700 dark:text-gray-300 border-b border-gray-300 dark:border-gray-600">
              Team
            </th>
            <th className="px-3 py-2 text-left text-sm font-medium text-gray-700 dark:text-gray-300 border-b border-gray-300 dark:border-gray-600">
              Affiliation
            </th>
            <th className="px-3 py-2 text-center text-sm font-medium text-gray-700 dark:text-gray-300 border-b border-gray-300 dark:border-gray-600">
              Solved
            </th>
            <th className="px-3 py-2 text-center text-sm font-medium text-gray-700 dark:text-gray-300 border-b border-gray-300 dark:border-gray-600">
              Time
            </th>
            {problemNames.map((name) => (
              <th
                key={name}
                className="px-2 py-2 text-center text-sm font-medium text-gray-700 dark:text-gray-300 border-b border-gray-300 dark:border-gray-600 w-16"
              >
                {name}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {entries.map((entry, idx) => (
            <tr
              key={entry.team_id}
              className={cn(
                idx % 2 === 0 ? 'bg-white dark:bg-gray-800' : 'bg-gray-50 dark:bg-gray-750',
                'hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors'
              )}
            >
              <td className="px-3 py-2 text-sm font-medium text-gray-900 dark:text-gray-100">
                {entry.rank}
              </td>
              <td className="px-3 py-2 text-sm text-gray-900 dark:text-gray-100">
                {entry.team_name}
              </td>
              <td className="px-3 py-2 text-sm text-gray-600 dark:text-gray-400">
                {entry.affiliation || '-'}
              </td>
              <td className="px-3 py-2 text-center text-sm font-medium text-green-600 dark:text-green-400">
                {entry.num_solved}
              </td>
              <td className="px-3 py-2 text-center text-sm text-gray-900 dark:text-gray-100">
                {entry.total_time}
              </td>
              {problemNames.map((name) => {
                const score = entry.problems?.find((p) => p.problem_short_name === name)
                return <ProblemCell key={name} score={score} />
              })}
            </tr>
          ))}
        </tbody>
      </table>

      {/* Contest time display */}
      {data.contest_time && (
        <div className="mt-4 text-sm text-gray-500 dark:text-gray-400 text-right">
          Contest time: {data.contest_time}
        </div>
      )}
    </div>
  )
}