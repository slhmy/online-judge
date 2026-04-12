'use client'

import Link from 'next/link'
import { useContests } from '@/hooks/useApi'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'

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
        color: 'bg-amber-100 text-amber-800 border-amber-200 dark:bg-amber-500/20 dark:text-amber-300 dark:border-amber-500/50',
      }
    }
    if (now > end) {
      return {
        label: 'Ended',
        color: 'bg-muted text-muted-foreground border-border',
      }
    }
    return {
      label: 'Running',
      color: 'bg-emerald-100 text-emerald-800 border-emerald-200 dark:bg-emerald-500/20 dark:text-emerald-300 dark:border-emerald-500/50',
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
        <h1 className="mb-6 font-heading text-2xl font-bold text-foreground">Contests</h1>
        <div className="py-10 text-center text-destructive">
          Error loading contests: {error.message}
        </div>
      </div>
    )
  }

  return (
    <div className="px-4 py-6">
      <h1 className="mb-6 font-heading text-2xl font-bold text-foreground">Contests</h1>

      {isLoading ? (
        <div className="py-10 text-center text-muted-foreground">Loading...</div>
      ) : contests.length === 0 ? (
        <div className="py-10 text-center text-muted-foreground">
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
                className="block"
              >
                <Card className="h-full transition-colors hover:bg-muted/40">
                  <CardHeader>
                    <div className="mb-2 flex items-start justify-between gap-2">
                      <CardTitle className="line-clamp-2 text-lg">{contest.name}</CardTitle>
                      <Badge variant="outline" className={status.color}>
                        {status.label}
                      </Badge>
                    </div>
                    <CardDescription>{contest.short_name}</CardDescription>
                  </CardHeader>
                  <CardContent className="space-y-1 text-sm">
                    <div className="flex items-center justify-between">
                      <span className="text-muted-foreground">Start:</span>
                      <span>{formatDateTime(contest.start_time)}</span>
                    </div>
                    <div className="flex items-center justify-between">
                      <span className="text-muted-foreground">End:</span>
                      <span>{formatDateTime(contest.end_time)}</span>
                    </div>
                    <div className="flex items-center justify-between">
                      <span className="text-muted-foreground">Duration:</span>
                      <span>{formatDuration(contest.start_time, contest.end_time)}</span>
                    </div>
                  </CardContent>
                  {!contest.public && (
                    <div className="px-4 pb-4">
                      <Badge variant="secondary">Private</Badge>
                    </div>
                  )}
                </Card>
              </Link>
            )
          })}
        </div>
      )}
    </div>
  )
}