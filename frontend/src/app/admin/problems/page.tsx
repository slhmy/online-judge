'use client'

import { useEffect, useState } from 'react'
import { useRouter } from 'next/navigation'
import { useAuthStore } from '@/stores/authStore'
import { useProblems, useCreateProblem, useUpdateProblem, useDeleteProblem } from '@/hooks/useApi'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import Editor from '@monaco-editor/react'

const BFF_URL = process.env.NEXT_PUBLIC_BFF_URL || 'http://localhost:8080'

// Form schema
const problemSchema = z.object({
  external_id: z.string().min(1, 'Problem ID is required').max(10, 'ID must be 10 characters or less'),
  name: z.string().min(1, 'Name is required'),
  time_limit: z.coerce.number().min(0.1, 'Time limit must be at least 0.1 seconds').max(60, 'Time limit cannot exceed 60 seconds'),
  memory_limit: z.coerce.number().min(256, 'Memory limit must be at least 256 KB').max(524288, 'Memory limit cannot exceed 512 MB'),
  difficulty: z.enum(['easy', 'medium', 'hard']),
  points: z.coerce.number().min(1, 'Points must be at least 1').max(1000, 'Points cannot exceed 1000'),
  description: z.string().optional(),
})

type ProblemFormData = z.infer<typeof problemSchema>

interface Problem {
  id: string
  external_id: string
  name: string
  difficulty: string
  time_limit: number
  memory_limit: number
  points: number
  allow_submit: boolean
}

interface ProblemsResponse {
  problems: Problem[]
  pagination: {
    total: number
    page: number
    page_size: number
  }
}

export default function AdminProblemsPage() {
  const router = useRouter()
  const { user, isAuthenticated, token } = useAuthStore()
  const [showForm, setShowForm] = useState(false)
  const [editingProblem, setEditingProblem] = useState<Problem | null>(null)
  const [markdownContent, setMarkdownContent] = useState('')
  const [deleteConfirmId, setDeleteConfirmId] = useState<string | null>(null)

  const { data, isLoading, error } = useProblems() as {
    data: ProblemsResponse | undefined
    isLoading: boolean
    error: Error | null
  }
  const problems: Problem[] = data?.problems || []

  const createMutation = useCreateProblem()
  const updateMutation = editingProblem ? useUpdateProblem(editingProblem.id) : null
  const deleteMutation = useDeleteProblem()

  const {
    register,
    handleSubmit,
    reset,
    setValue,
    formState: { errors, isSubmitting },
  } = useForm<ProblemFormData>({
    resolver: zodResolver(problemSchema),
    defaultValues: {
      external_id: '',
      name: '',
      time_limit: 2,
      memory_limit: 262144, // 256 MB
      difficulty: 'medium',
      points: 100,
      description: '',
    },
  })

  useEffect(() => {
    // Check if user is admin
    if (!isAuthenticated || user?.role !== 'admin') {
      router.push('/')
      return
    }
  }, [isAuthenticated, user, router])

  useEffect(() => {
    if (editingProblem) {
      setValue('external_id', editingProblem.external_id)
      setValue('name', editingProblem.name)
      setValue('time_limit', editingProblem.time_limit)
      setValue('memory_limit', editingProblem.memory_limit)
      setValue('difficulty', editingProblem.difficulty as 'easy' | 'medium' | 'hard')
      setValue('points', editingProblem.points)
      setMarkdownContent('')
      setShowForm(true)
    }
  }, [editingProblem, setValue])

  const onSubmit = async (data: ProblemFormData) => {
    try {
      const problemData = {
        ...data,
        description: markdownContent,
      }

      if (editingProblem) {
        await updateMutation?.mutateAsync({
          name: problemData.name,
          time_limit: problemData.time_limit,
          memory_limit: problemData.memory_limit,
          difficulty: problemData.difficulty,
          points: problemData.points,
          description: problemData.description,
        })
        setEditingProblem(null)
      } else {
        await createMutation.mutateAsync(problemData)
      }

      reset()
      setMarkdownContent('')
      setShowForm(false)
    } catch (err) {
      console.error('Failed to save problem:', err)
    }
  }

  const handleDelete = async (id: string) => {
    try {
      await deleteMutation.mutateAsync(id)
      setDeleteConfirmId(null)
    } catch (err) {
      console.error('Failed to delete problem:', err)
    }
  }

  const handleEdit = async (problem: Problem) => {
    // Fetch full problem details including description
    try {
      const res = await fetch(`${BFF_URL}/api/v1/problems/${problem.id}`, {
        headers: {
          Authorization: `Bearer ${token}`,
        },
      })
      if (res.ok) {
        const data = await res.json()
        setEditingProblem(data.problem || problem)
        if (data.problem?.problem_statement_path) {
          // If there's a problem statement path, we might need to fetch it separately
          setMarkdownContent('')
        }
      } else {
        setEditingProblem(problem)
      }
    } catch (err) {
      console.error('Failed to fetch problem details:', err)
      setEditingProblem(problem)
    }
  }

  const handleCancel = () => {
    setShowForm(false)
    setEditingProblem(null)
    reset()
    setMarkdownContent('')
  }

  if (!isAuthenticated || user?.role !== 'admin') {
    return null
  }

  const difficultyColors = {
    easy: 'text-green-700 bg-green-100 dark:text-green-400 dark:bg-green-900/50',
    medium: 'text-yellow-700 bg-yellow-100 dark:text-yellow-400 dark:bg-yellow-900/50',
    hard: 'text-red-700 bg-red-100 dark:text-red-400 dark:bg-red-900/50',
  }

  return (
    <div className="px-4 py-6">
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-bold text-gray-900 dark:text-gray-100">Admin: Problem Management</h1>
        <button
          onClick={() => setShowForm(!showForm)}
          className="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg font-medium transition-colors"
        >
          {showForm ? 'Cancel' : 'Create Problem'}
        </button>
      </div>

      {/* Problem Form */}
      {showForm && (
        <div className="bg-white dark:bg-gray-800 rounded-lg shadow p-6 mb-8">
          <h2 className="text-lg font-semibold mb-4 text-gray-900 dark:text-gray-100">
            {editingProblem ? `Edit Problem: ${editingProblem.name}` : 'Create New Problem'}
          </h2>

          <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              {/* Problem ID */}
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                  Problem ID (e.g., A, B, 101)
                </label>
                <input
                  {...register('external_id')}
                  type="text"
                  disabled={!!editingProblem}
                  className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 disabled:opacity-50"
                  placeholder="A"
                />
                {errors.external_id && (
                  <p className="mt-1 text-sm text-red-600 dark:text-red-400">{errors.external_id.message}</p>
                )}
              </div>

              {/* Name */}
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                  Problem Name
                </label>
                <input
                  {...register('name')}
                  type="text"
                  className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100"
                  placeholder="Two Sum"
                />
                {errors.name && (
                  <p className="mt-1 text-sm text-red-600 dark:text-red-400">{errors.name.message}</p>
                )}
              </div>

              {/* Time Limit */}
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                  Time Limit (seconds)
                </label>
                <input
                  {...register('time_limit')}
                  type="number"
                  step="0.1"
                  className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100"
                  placeholder="2"
                />
                {errors.time_limit && (
                  <p className="mt-1 text-sm text-red-600 dark:text-red-400">{errors.time_limit.message}</p>
                )}
              </div>

              {/* Memory Limit */}
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                  Memory Limit (KB)
                </label>
                <input
                  {...register('memory_limit')}
                  type="number"
                  className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100"
                  placeholder="262144"
                />
                {errors.memory_limit && (
                  <p className="mt-1 text-sm text-red-600 dark:text-red-400">{errors.memory_limit.message}</p>
                )}
                <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">
                  Common values: 262144 (256MB), 524288 (512MB)
                </p>
              </div>

              {/* Difficulty */}
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                  Difficulty
                </label>
                <select
                  {...register('difficulty')}
                  className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100"
                >
                  <option value="easy">Easy</option>
                  <option value="medium">Medium</option>
                  <option value="hard">Hard</option>
                </select>
                {errors.difficulty && (
                  <p className="mt-1 text-sm text-red-600 dark:text-red-400">{errors.difficulty.message}</p>
                )}
              </div>

              {/* Points */}
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                  Points
                </label>
                <input
                  {...register('points')}
                  type="number"
                  className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100"
                  placeholder="100"
                />
                {errors.points && (
                  <p className="mt-1 text-sm text-red-600 dark:text-red-400">{errors.points.message}</p>
                )}
              </div>
            </div>

            {/* Markdown Editor for Problem Statement */}
            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                Problem Statement (Markdown)
              </label>
              <div className="border border-gray-300 dark:border-gray-600 rounded-lg overflow-hidden">
                <Editor
                  height="300px"
                  defaultLanguage="markdown"
                  value={markdownContent}
                  onChange={(value) => setMarkdownContent(value || '')}
                  theme="vs-dark"
                  options={{
                    minimap: { enabled: false },
                    fontSize: 14,
                    lineNumbers: 'on',
                    wordWrap: 'on',
                    scrollBeyondLastLine: false,
                    automaticLayout: true,
                  }}
                />
              </div>
              <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">
                Use Markdown format for the problem description. Include input/output format, examples, and constraints.
              </p>
            </div>

            {/* Submit Button */}
            <div className="flex gap-3">
              <button
                type="submit"
                disabled={isSubmitting || createMutation.isPending || updateMutation?.isPending}
                className="px-4 py-2 bg-green-600 hover:bg-green-700 disabled:opacity-50 text-white rounded-lg font-medium transition-colors"
              >
                {isSubmitting || createMutation.isPending || updateMutation?.isPending
                  ? 'Saving...'
                  : editingProblem
                  ? 'Update Problem'
                  : 'Create Problem'}
              </button>
              <button
                type="button"
                onClick={handleCancel}
                className="px-4 py-2 bg-gray-200 hover:bg-gray-300 dark:bg-gray-700 dark:hover:bg-gray-600 text-gray-700 dark:text-gray-200 rounded-lg font-medium transition-colors"
              >
                Cancel
              </button>
            </div>

            {/* Error Messages */}
            {(createMutation.isError || updateMutation?.isError) && (
              <div className="p-3 bg-red-100 dark:bg-red-900/50 text-red-700 dark:text-red-400 rounded-lg">
                Failed to save problem. Please try again.
              </div>
            )}
          </form>
        </div>
      )}

      {/* Problem List */}
      {error && (
        <div className="text-center py-10 text-red-400">
          Error loading problems: {error.message}
        </div>
      )}

      {isLoading ? (
        <div className="text-center py-10 text-gray-600 dark:text-gray-400">Loading...</div>
      ) : problems.length === 0 ? (
        <div className="text-center py-10 text-gray-500 dark:text-gray-400">
          No problems available. Create one above.
        </div>
      ) : (
        <div className="overflow-x-auto">
          <table className="min-w-full bg-white dark:bg-gray-800 rounded-lg shadow">
            <thead className="bg-gray-100 dark:bg-gray-700">
              <tr>
                <th className="px-4 py-3 text-left text-sm font-medium text-gray-700 dark:text-gray-300">ID</th>
                <th className="px-4 py-3 text-left text-sm font-medium text-gray-700 dark:text-gray-300">Name</th>
                <th className="px-4 py-3 text-left text-sm font-medium text-gray-700 dark:text-gray-300">Difficulty</th>
                <th className="px-4 py-3 text-left text-sm font-medium text-gray-700 dark:text-gray-300">Time</th>
                <th className="px-4 py-3 text-left text-sm font-medium text-gray-700 dark:text-gray-300">Memory</th>
                <th className="px-4 py-3 text-left text-sm font-medium text-gray-700 dark:text-gray-300">Points</th>
                <th className="px-4 py-3 text-left text-sm font-medium text-gray-700 dark:text-gray-300">Actions</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-200 dark:divide-gray-700">
              {problems.map((problem) => (
                <tr key={problem.id} className="hover:bg-gray-50 dark:hover:bg-gray-700/50">
                  <td className="px-4 py-3 text-sm font-medium text-gray-900 dark:text-gray-100">
                    {problem.external_id}
                  </td>
                  <td className="px-4 py-3 text-sm text-gray-900 dark:text-gray-100">
                    {problem.name}
                  </td>
                  <td className="px-4 py-3 text-sm">
                    <span className={`px-2 py-1 rounded text-xs font-medium ${difficultyColors[problem.difficulty as keyof typeof difficultyColors] || 'bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-300'}`}>
                      {problem.difficulty}
                    </span>
                  </td>
                  <td className="px-4 py-3 text-sm text-gray-700 dark:text-gray-300">
                    {problem.time_limit}s
                  </td>
                  <td className="px-4 py-3 text-sm text-gray-700 dark:text-gray-300">
                    {(problem.memory_limit / 1024).toFixed(0)} MB
                  </td>
                  <td className="px-4 py-3 text-sm text-gray-700 dark:text-gray-300">
                    {problem.points}
                  </td>
                  <td className="px-4 py-3 text-sm">
                    <div className="flex gap-2">
                      <button
                        onClick={() => handleEdit(problem)}
                        className="text-blue-600 hover:text-blue-800 dark:text-blue-400 dark:hover:text-blue-300 font-medium"
                      >
                        Edit
                      </button>
                      <button
                        onClick={() => router.push(`/admin/problems/${problem.id}/testcases`)}
                        className="text-purple-600 hover:text-purple-800 dark:text-purple-400 dark:hover:text-purple-300 font-medium"
                      >
                        Test Cases
                      </button>
                      {deleteConfirmId === problem.id ? (
                        <div className="flex gap-1">
                          <button
                            onClick={() => handleDelete(problem.id)}
                            className="text-red-600 hover:text-red-800 dark:text-red-400 dark:hover:text-red-300 font-medium"
                          >
                            Confirm
                          </button>
                          <button
                            onClick={() => setDeleteConfirmId(null)}
                            className="text-gray-600 hover:text-gray-800 dark:text-gray-400 dark:hover:text-gray-300 font-medium"
                          >
                            Cancel
                          </button>
                        </div>
                      ) : (
                        <button
                          onClick={() => setDeleteConfirmId(problem.id)}
                          className="text-red-600 hover:text-red-800 dark:text-red-400 dark:hover:text-red-300 font-medium"
                        >
                          Delete
                        </button>
                      )}
                      <button
                        onClick={() => router.push(`/problems/${problem.id}`)}
                        className="text-gray-600 hover:text-gray-800 dark:text-gray-400 dark:hover:text-gray-300 font-medium"
                      >
                        View
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {/* Back to Admin Dashboard */}
      <div className="mt-6">
        <button
          onClick={() => router.push('/admin')}
          className="text-gray-600 hover:text-gray-800 dark:text-gray-400 dark:hover:text-gray-300"
        >
          ← Back to Admin Dashboard
        </button>
      </div>
    </div>
  )
}