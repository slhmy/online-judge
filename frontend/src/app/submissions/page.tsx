'use client'

import { useState } from 'react'
import Link from 'next/link'
import { useSubmissions } from '@/hooks/useApi'
import { VERDICT_CONFIG } from '@/types'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { Badge } from '@/components/ui/badge'
import {
  Pagination,
  PaginationContent,
  PaginationEllipsis,
  PaginationItem,
  PaginationLink,
  PaginationNext,
  PaginationPrevious,
} from '@/components/ui/pagination'

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

const getVisiblePages = (currentPage: number, totalPages: number): number[] => {
  if (totalPages <= 7) {
    return Array.from({ length: totalPages }, (_, i) => i + 1)
  }

  if (currentPage <= 3) {
    return [1, 2, 3, 4, 5, -1, totalPages]
  }

  if (currentPage >= totalPages - 2) {
    return [1, -1, totalPages - 4, totalPages - 3, totalPages - 2, totalPages - 1, totalPages]
  }

  return [1, -1, currentPage - 1, currentPage, currentPage + 1, -1, totalPages]
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
  const visiblePages = getVisiblePages(page, totalPages)

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

  const getVerdictBadgeVariant = (verdict: string | number): 'default' | 'secondary' | 'destructive' => {
    const key = getVerdictKey(verdict)
    if (!key) return 'secondary'
    if (key === 'correct') return 'default'
    return 'destructive'
  }

  const getVerdictBadgeClass = (verdict: string | number): string => {
    const key = getVerdictKey(verdict)
    if (!key) return ''
    const verdictInfo = VERDICT_CONFIG[key as keyof typeof VERDICT_CONFIG]
    return verdictInfo?.color ? `${verdictInfo.color} text-white hover:opacity-90` : ''
  }

  if (error) {
    return (
      <div className="px-4 py-6">
        <h1 className="text-2xl font-bold mb-6 text-foreground">Submissions</h1>
        <div className="text-center py-10 text-red-400">
          Error loading submissions: {error.message}
        </div>
      </div>
    )
  }

  return (
    <div className="px-4 py-6">
      <Card>
        <CardHeader>
          <CardTitle>Submissions</CardTitle>
          <CardDescription>Track recent runs, verdicts, and submitters.</CardDescription>
        </CardHeader>
        <CardContent>

      {isLoading ? (
          <div className="py-10 text-center text-muted-foreground">Loading...</div>
      ) : submissions.length === 0 ? (
          <div className="py-10 text-center text-muted-foreground">
            No submissions yet.{' '}
            <Link href="/problems" className="font-medium text-primary hover:underline">
              Solve some problems!
            </Link>
        </div>
      ) : (
        <div className="space-y-4">
            <Table className="[&_tr]:border-zinc-200 dark:[&_tr]:border-zinc-800">
              <TableHeader>
                <tr>
                  <TableHead>ID</TableHead>
                  <TableHead>Submitter</TableHead>
                  <TableHead>Problem</TableHead>
                  <TableHead>Language</TableHead>
                  <TableHead>Verdict</TableHead>
                  <TableHead>Time</TableHead>
                  <TableHead>Memory</TableHead>
                  <TableHead>Submitted</TableHead>
                </tr>
              </TableHeader>
              <TableBody>
                {submissions.map((sub) => {
                  const verdictKey = getVerdictKey(sub.verdict)
                  const verdictInfo = verdictKey ? VERDICT_CONFIG[verdictKey as keyof typeof VERDICT_CONFIG] : null
                  return (
                    <TableRow key={sub.id}>
                      <TableCell>
                        <Link href={`/submissions/${sub.id}`} className="text-primary hover:underline font-mono">
                          {sub.id.slice(0, 8)}
                        </Link>
                      </TableCell>
                      <TableCell>{formatSubmitter(sub)}</TableCell>
                      <TableCell>
                        <Link href={`/problems/${sub.problem_id}`} className="text-primary hover:underline">
                          {sub.problem_name || sub.problem_id}
                        </Link>
                      </TableCell>
                      <TableCell>{sub.language_id}</TableCell>
                      <TableCell>
                        <Badge
                          variant={getVerdictBadgeVariant(sub.verdict)}
                          className={getVerdictBadgeClass(sub.verdict)}
                        >
                          {verdictInfo?.label || 'Pending'}
                        </Badge>
                      </TableCell>
                      <TableCell>{formatRuntime(sub.runtime)}</TableCell>
                      <TableCell>{formatMemory(sub.memory)}</TableCell>
                      <TableCell className="text-muted-foreground">
                        {formatTime(sub.submit_time)}
                      </TableCell>
                    </TableRow>
                  )
                })}
              </TableBody>
            </Table>

            <div className="flex items-center justify-between rounded-xl border border-zinc-200 bg-zinc-50/60 px-4 py-3 dark:border-zinc-800 dark:bg-zinc-900/40">
              <div className="text-sm text-muted-foreground">
                Page {page} / {totalPages} · Total {total}
            </div>
              <Pagination className="mx-0 w-auto justify-end">
                <PaginationContent>
                  <PaginationItem>
                    <PaginationPrevious
                      href="#"
                      aria-disabled={isLoading || page <= 1}
                      className={page <= 1 ? 'pointer-events-none opacity-50' : ''}
                      onClick={(e) => {
                        e.preventDefault()
                        goToPrevPage()
                      }}
                    />
                  </PaginationItem>
                  {visiblePages.map((pageNumber, index) => (
                    <PaginationItem key={`page-${pageNumber}-${index}`}>
                      {pageNumber === -1 ? (
                        <PaginationEllipsis />
                      ) : (
                        <PaginationLink
                          href="#"
                          isActive={pageNumber === page}
                          onClick={(e) => {
                            e.preventDefault()
                            setCurrentPage(pageNumber)
                          }}
                        >
                          {pageNumber}
                        </PaginationLink>
                      )}
                    </PaginationItem>
                  ))}
                  <PaginationItem>
                    <PaginationNext
                      href="#"
                      aria-disabled={isLoading || page >= totalPages}
                      className={page >= totalPages ? 'pointer-events-none opacity-50' : ''}
                      onClick={(e) => {
                        e.preventDefault()
                        goToNextPage()
                      }}
                    />
                  </PaginationItem>
                </PaginationContent>
              </Pagination>
            </div>
          </div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}