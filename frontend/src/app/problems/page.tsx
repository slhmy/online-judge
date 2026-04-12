'use client'

import { useProblems } from '@/hooks/useApi'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent } from '@/components/ui/card'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'

interface Problem {
  id: string
  name: string
  difficulty: string
  time_limit: number
  memory_limit: number
  points: number
}

interface ProblemsResponse {
  problems: Problem[]
  pagination: {
    total: number
    page: number
    page_size: number
  }
}

export default function ProblemsPage() {
  const { data, isLoading, error } = useProblems(1, 20) as {
    data: ProblemsResponse | undefined
    isLoading: boolean
    error: Error | null
  }
  const problems: Problem[] = data?.problems || []

  const difficultyColors = {
    easy: 'bg-emerald-100 text-emerald-800 dark:bg-emerald-500/20 dark:text-emerald-300',
    medium: 'bg-amber-100 text-amber-800 dark:bg-amber-500/20 dark:text-amber-300',
    hard: 'bg-red-100 text-red-800 dark:bg-red-500/20 dark:text-red-300',
  }

  if (error) {
    return (
      <div className="px-4 py-6">
        <h1 className="mb-6 font-heading text-2xl font-bold text-foreground">Problems</h1>
        <div className="py-10 text-center text-destructive">
          Error loading problems: {error.message}
        </div>
      </div>
    )
  }

  return (
    <div className="px-4 py-6">
      <h1 className="mb-6 font-heading text-2xl font-bold text-foreground">Problems</h1>

      {isLoading ? (
        <div className="py-10 text-center text-muted-foreground">Loading...</div>
      ) : problems.length === 0 ? (
        <div className="py-10 text-center text-muted-foreground">
          No problems available.
        </div>
      ) : (
        <Card>
          <CardContent className="p-0">
            <Table>
              <TableHeader>
                <TableRow className="bg-muted/40 hover:bg-muted/40">
                  <TableHead className="px-4">#</TableHead>
                  <TableHead className="px-4">Name</TableHead>
                  <TableHead className="px-4">Difficulty</TableHead>
                  <TableHead className="px-4">Time Limit</TableHead>
                  <TableHead className="px-4">Memory Limit</TableHead>
                  <TableHead className="px-4">Points</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
              {problems.map((problem, index) => (
                <TableRow key={problem.id} className="cursor-pointer" onClick={() => window.location.href = `/problems/${problem.id}`}>
                  <TableCell className="px-4 text-muted-foreground">{index + 1}</TableCell>
                  <TableCell className="px-4 text-sm font-medium text-primary">
                    {problem.name}
                  </TableCell>
                  <TableCell className="px-4">
                    <Badge className={difficultyColors[problem.difficulty as keyof typeof difficultyColors] || 'bg-muted text-foreground'}>
                      {problem.difficulty}
                    </Badge>
                  </TableCell>
                  <TableCell className="px-4">{problem.time_limit}s</TableCell>
                  <TableCell className="px-4">{(problem.memory_limit / 1024).toFixed(0)} MB</TableCell>
                  <TableCell className="px-4">{problem.points}</TableCell>
                </TableRow>
              ))}
              </TableBody>
            </Table>
          </CardContent>
        </Card>
      )}
    </div>
  )
}