'use client'

import { useProblems } from '@/hooks/useApi'

interface Problem {
  id: string
  external_id: string
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
    easy: 'text-green-600 dark:text-green-400 bg-green-100 dark:bg-green-900/50',
    medium: 'text-yellow-600 dark:text-yellow-400 bg-yellow-100 dark:bg-yellow-900/50',
    hard: 'text-red-600 dark:text-red-400 bg-red-100 dark:bg-red-900/50',
  }

  if (error) {
    return (
      <div className="px-4 py-6">
        <h1 className="text-2xl font-bold mb-6 text-gray-900 dark:text-gray-100">Problems</h1>
        <div className="text-center py-10 text-red-400">
          Error loading problems: {error.message}
        </div>
      </div>
    )
  }

  return (
    <div className="px-4 py-6">
      <h1 className="text-2xl font-bold mb-6 text-gray-900 dark:text-gray-100">Problems</h1>

      {isLoading ? (
        <div className="text-center py-10 text-gray-600 dark:text-gray-400">Loading...</div>
      ) : problems.length === 0 ? (
        <div className="text-center py-10 text-gray-500">
          No problems available.
        </div>
      ) : (
        <div className="overflow-x-auto">
          <table className="min-w-full bg-white dark:bg-gray-800 rounded-lg shadow">
            <thead className="bg-gray-100 dark:bg-gray-700">
              <tr>
                <th className="px-4 py-3 text-left text-sm font-medium text-gray-700 dark:text-gray-300">#</th>
                <th className="px-4 py-3 text-left text-sm font-medium text-gray-700 dark:text-gray-300">Name</th>
                <th className="px-4 py-3 text-left text-sm font-medium text-gray-700 dark:text-gray-300">Difficulty</th>
                <th className="px-4 py-3 text-left text-sm font-medium text-gray-700 dark:text-gray-300">Time Limit</th>
                <th className="px-4 py-3 text-left text-sm font-medium text-gray-700 dark:text-gray-300">Memory Limit</th>
                <th className="px-4 py-3 text-left text-sm font-medium text-gray-700 dark:text-gray-300">Points</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-200 dark:divide-gray-700">
              {problems.map((problem, index) => (
                <tr key={problem.id} className="hover:bg-gray-100 dark:bg-gray-700/50 cursor-pointer" onClick={() => window.location.href = `/problems/${problem.id}`}>
                  <td className="px-4 py-3 text-sm text-gray-600 dark:text-gray-400">{index + 1}</td>
                  <td className="px-4 py-3 text-sm font-medium text-blue-600 dark:text-blue-400 hover:text-blue-700 dark:hover:text-blue-300">
                    {problem.external_id}. {problem.name}
                  </td>
                  <td className="px-4 py-3 text-sm">
                    <span className={`px-2 py-1 rounded text-xs font-medium ${difficultyColors[problem.difficulty as keyof typeof difficultyColors] || 'bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-300'}`}>
                      {problem.difficulty}
                    </span>
                  </td>
                  <td className="px-4 py-3 text-sm text-gray-700 dark:text-gray-300">{problem.time_limit}s</td>
                  <td className="px-4 py-3 text-sm text-gray-700 dark:text-gray-300">{(problem.memory_limit / 1024).toFixed(0)} MB</td>
                  <td className="px-4 py-3 text-sm text-gray-700 dark:text-gray-300">{problem.points}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}