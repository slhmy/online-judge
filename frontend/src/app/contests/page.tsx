'use client'

import Link from 'next/link'
import { useContests } from '@/hooks/useApi'

interface Contest {
  id: string
  external_id: string
  name: string
  short_name: string
  start_time: string
  end_time: string
  public: boolean
}

interface ContestsResponse {
  contests: Contest[]
  pagination: {
    total: number
    page: number
    page_size: number
  }
}

export default function ContestsPage() {
  const { data, isLoading, error } = useContests(1, 50) as {
    data: ContestsResponse | undefined
    isLoading: boolean
    error: Error | null
  }
  const contests: Contest[] = data?.contests || []

  const getStatus = (contest: Contest) => {
    const now = new Date()
    const start = new Date(contest.start_time)
    const end = new Date(contest.end_time)

    if (now < start) {
      return {
        label: 'Upcoming',
        color: 'bg-yellow-100 text-yellow-800 border-yellow-300 dark:bg-yellow-500/20 dark:text-yellow-300 dark:border-yellow-500/50',
      }
    }
    if (now > end) {
      return {
        label: 'Ended',
        color: 'bg-gray-100 text-gray-700 border-gray-300 dark:bg-gray-500/20 dark:text-gray-300 dark:border-gray-500/50',
      }
    }
    return {
      label: 'Running',
      color: 'bg-green-100 text-green-800 border-green-300 dark:bg-green-500/20 dark:text-green-300 dark:border-green-500/50',
    }
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
      return `${days}d ${hours % 24}h`
    }
    return `${hours}h ${mins}m`
  }

  if (error) {
    return (
      <div className="px-4 py-6">
        <h1 className="text-2xl font-bold mb-6 text-gray-900 dark:text-gray-100">Contests</h1>
        <div className="text-center py-10 text-red-600 dark:text-red-400">
          Error loading contests: {error.message}
        </div>
      </div>
    )
  }

  return (
    <div className="px-4 py-6">
      <h1 className="text-2xl font-bold mb-6 text-gray-900 dark:text-gray-100">Contests</h1>

      {isLoading ? (
        <div className="text-center py-10 text-gray-500 dark:text-gray-400">Loading...</div>
      ) : contests.length === 0 ? (
        <div className="text-center py-10 text-gray-500 dark:text-gray-400">
          No contests available.
        </div>
      ) : (
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
          {contests.map((contest) => {
            const status = getStatus(contest)
            return (
              <Link
                key={contest.id}
                href={`/contests/${contest.id}`}
                className="block bg-white dark:bg-gray-800 rounded-lg shadow hover:shadow-lg transition-all hover:bg-gray-50 dark:hover:bg-gray-700 border border-gray-200 dark:border-gray-700 hover:border-gray-300 dark:hover:border-gray-600"
              >
                <div className="p-4">
                  <div className="flex items-start justify-between mb-2 gap-2">
                    <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100 line-clamp-2">{contest.name}</h2>
                    <span className={`px-2 py-1 rounded text-xs font-medium border shrink-0 ${status.color}`}>
                      {status.label}
                    </span>
                  </div>
                  <p className="text-sm text-gray-600 dark:text-gray-400 mb-3">{contest.short_name}</p>
                  <div className="text-sm text-gray-700 dark:text-gray-300 space-y-1">
                    <div className="flex items-center justify-between">
                      <span className="text-gray-500 dark:text-gray-400">Start:</span>
                      <span>{formatDateTime(contest.start_time)}</span>
                    </div>
                    <div className="flex items-center justify-between">
                      <span className="text-gray-500 dark:text-gray-400">End:</span>
                      <span>{formatDateTime(contest.end_time)}</span>
                    </div>
                    <div className="flex items-center justify-between">
                      <span className="text-gray-500 dark:text-gray-400">Duration:</span>
                      <span>{formatDuration(contest.start_time, contest.end_time)}</span>
                    </div>
                  </div>
                  {!contest.public && (
                    <div className="mt-3">
                      <span className="text-xs text-gray-600 dark:text-gray-300 bg-gray-100 dark:bg-gray-700 px-2 py-1 rounded">Private</span>
                    </div>
                  )}
                </div>
              </Link>
            )
          })}
        </div>
      )}
    </div>
  )
}