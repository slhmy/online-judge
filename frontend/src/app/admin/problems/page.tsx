'use client'

import { useEffect, useState } from 'react'
import { useRouter } from 'next/navigation'
import { useAuthStore } from '@/stores/authStore'
import { useProblems, useCreateProblem, useUpdateProblem, useDeleteProblem } from '@/hooks/useApi'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import Editor from '@monaco-editor/react'
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import remarkMath from 'remark-math'
import rehypeKatex from 'rehype-katex'
import 'katex/dist/katex.min.css'

const BFF_URL = process.env.NEXT_PUBLIC_BFF_URL || ''

// Form schema
const problemSchema = z.object({
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
  const { user, isAuthenticated } = useAuthStore()
  const [hydrated, setHydrated] = useState(false)
  const [showForm, setShowForm] = useState(false)
  const [editingProblem, setEditingProblem] = useState<Problem | null>(null)
  const [statementContent, setStatementContent] = useState('')
  const [statementFormat, setStatementFormat] = useState<'markdown' | 'html' | 'plain' | 'pdf'>('markdown')
  const [statementLanguage, setStatementLanguage] = useState('en')
  const [statementTitle, setStatementTitle] = useState('')
  const [previewMode, setPreviewMode] = useState<'edit' | 'preview' | 'split'>('split')
  const [deleteConfirmId, setDeleteConfirmId] = useState<string | null>(null)

  const { data, isLoading, error } = useProblems() as {
    data: ProblemsResponse | undefined
    isLoading: boolean
    error: Error | null
  }
  const problems: Problem[] = data?.problems || []

  const createMutation = useCreateProblem()
  const updateMutation = useUpdateProblem(editingProblem?.id || '')
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
      name: '',
      time_limit: 2,
      memory_limit: 262144, // 256 MB
      difficulty: 'medium',
      points: 100,
      description: '',
    },
  })

  useEffect(() => {
    setHydrated(useAuthStore.persist.hasHydrated())
    const unsubFinish = useAuthStore.persist.onFinishHydration(() => setHydrated(true))
    return () => {
      unsubFinish()
    }
  }, [])

  useEffect(() => {
    // Check if user is admin
    if (!hydrated) {
      return
    }

    if (!isAuthenticated || user?.role !== 'admin') {
      router.push('/')
      return
    }
  }, [hydrated, isAuthenticated, user, router])

  useEffect(() => {
    if (editingProblem) {
      setValue('name', editingProblem.name)
      setValue('time_limit', editingProblem.time_limit)
      setValue('memory_limit', editingProblem.memory_limit)
      setValue('difficulty', editingProblem.difficulty as 'easy' | 'medium' | 'hard')
      setValue('points', editingProblem.points)
      setStatementContent('')
      setStatementFormat('markdown')
      setStatementLanguage('en')
      setStatementTitle(editingProblem.name || '')
      setPreviewMode('split')
      setShowForm(true)
    }
  }, [editingProblem, setValue])

  const getEditorLanguage = (format: 'markdown' | 'html' | 'plain' | 'pdf') => {
    switch (format) {
      case 'html':
        return 'html'
      case 'plain':
        return 'plaintext'
      case 'pdf':
        return 'markdown'
      case 'markdown':
      default:
        return 'markdown'
    }
  }

  const renderStatementPreview = () => {
    if (!statementContent.trim()) {
      return <p className="text-muted-foreground">No content to preview.</p>
    }

    if (statementFormat === 'html') {
      return <div dangerouslySetInnerHTML={{ __html: statementContent }} />
    }

    if (statementFormat === 'plain') {
      return <pre className="whitespace-pre-wrap break-words text-sm">{statementContent}</pre>
    }

    if (statementFormat === 'pdf') {
      return (
        <div className="text-sm text-yellow-700 dark:text-yellow-300">
          PDF format is stored as content metadata. Inline PDF preview is not available in this editor.
        </div>
      )
    }

    return (
      <ReactMarkdown remarkPlugins={[remarkGfm, remarkMath]} rehypePlugins={[rehypeKatex]}>
        {statementContent}
      </ReactMarkdown>
    )
  }

  const upsertProblemStatement = async (problemID: string) => {
    const res = await fetch(`${BFF_URL}/api/v1/problems/${problemID}/statement`, {
      method: 'PUT',
      credentials: 'include',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        language: statementLanguage || 'en',
        format: statementFormat,
        title: statementTitle,
        content: statementContent,
      }),
    })

    if (!res.ok) {
      throw new Error(`Failed to save problem statement: ${res.status}`)
    }
  }

  const onSubmit = async (data: ProblemFormData) => {
    try {
      const problemData = {
        ...data,
      }

      let targetProblemID = editingProblem?.id || ''

      if (editingProblem) {
        await updateMutation?.mutateAsync({
          name: problemData.name,
          time_limit: problemData.time_limit,
          memory_limit: problemData.memory_limit,
          difficulty: problemData.difficulty,
          points: problemData.points,
        })
        setEditingProblem(null)
      } else {
        const created = await createMutation.mutateAsync(problemData)
        if (created && typeof created === 'object' && 'id' in created) {
          targetProblemID = String((created as { id?: string }).id || '')
        }
      }

      if (targetProblemID) {
        await upsertProblemStatement(targetProblemID)
      }

      reset()
      setStatementContent('')
      setStatementFormat('markdown')
      setStatementLanguage('en')
      setStatementTitle('')
      setPreviewMode('split')
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
        credentials: 'include',
      })
      if (res.ok) {
        const data = await res.json()
        setEditingProblem(data.problem || problem)

        // Fetch the problem statement content
        const statementRes = await fetch(`${BFF_URL}/api/v1/problems/${problem.id}/statement`, {
          credentials: 'include',
        })
        if (statementRes.ok) {
          const statementData = await statementRes.json()
          if (typeof statementData === 'string') {
            setStatementContent(statementData)
            setStatementFormat('markdown')
            setStatementLanguage('en')
            setStatementTitle(problem.name)
          } else if (statementData && typeof statementData.content === 'string') {
            setStatementContent(statementData.content)
            const fmt = (statementData.format || 'markdown') as 'markdown' | 'html' | 'plain' | 'pdf'
            setStatementFormat(fmt)
            setStatementLanguage(statementData.language || 'en')
            setStatementTitle(statementData.title || problem.name)
          } else {
            setStatementContent('')
            setStatementFormat('markdown')
            setStatementLanguage('en')
            setStatementTitle(problem.name)
          }
        } else {
          setStatementContent('')
          setStatementFormat('markdown')
          setStatementLanguage('en')
          setStatementTitle(problem.name)
        }

        setPreviewMode('split')
        setShowForm(true)
      } else {
        setEditingProblem(problem)
        setStatementContent('')
        setStatementFormat('markdown')
        setStatementLanguage('en')
        setStatementTitle(problem.name)
        setPreviewMode('split')
        setShowForm(true)
      }
    } catch (err) {
      console.error('Failed to fetch problem details:', err)
      setEditingProblem(problem)
      setStatementContent('')
      setStatementFormat('markdown')
      setStatementLanguage('en')
      setStatementTitle(problem.name)
      setPreviewMode('split')
      setShowForm(true)
    }
  }

  const handleCancel = () => {
    setShowForm(false)
    setEditingProblem(null)
    reset()
    setStatementContent('')
    setStatementFormat('markdown')
    setStatementLanguage('en')
    setStatementTitle('')
    setPreviewMode('split')
  }

  if (!hydrated || !isAuthenticated || user?.role !== 'admin') {
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
        <h1 className="text-2xl font-bold text-foreground">Problem Management</h1>
        <button
          onClick={() => setShowForm(!showForm)}
          className="px-4 py-2 bg-primary hover:bg-primary/90 text-white rounded-xl font-medium transition-colors"
        >
          {showForm ? 'Cancel' : 'Create Problem'}
        </button>
      </div>

      {/* Problem Form */}
      {showForm && (
        <div className="bg-card rounded-xl shadow p-6 mb-8">
          <h2 className="text-lg font-semibold mb-4 text-foreground">
            {editingProblem ? `Edit Problem: ${editingProblem.name}` : 'Create New Problem'}
          </h2>

          <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          {/* Name */}
              <div>
                <label className="block text-sm font-medium text-foreground mb-1">
                  Problem Name
                </label>
                <input
                  {...register('name')}
                  type="text"
                  className="w-full px-3 py-2 border border-border rounded-xl bg-white text-foreground"
                  placeholder="Two Sum"
                />
                {errors.name && (
                  <p className="mt-1 text-sm text-red-600 dark:text-red-400">{errors.name.message}</p>
                )}
              </div>

              {/* Time Limit */}
              <div>
                <label className="block text-sm font-medium text-foreground mb-1">
                  Time Limit (seconds)
                </label>
                <input
                  {...register('time_limit')}
                  type="number"
                  step="0.1"
                  className="w-full px-3 py-2 border border-border rounded-xl bg-white text-foreground"
                  placeholder="2"
                />
                {errors.time_limit && (
                  <p className="mt-1 text-sm text-red-600 dark:text-red-400">{errors.time_limit.message}</p>
                )}
              </div>

              {/* Memory Limit */}
              <div>
                <label className="block text-sm font-medium text-foreground mb-1">
                  Memory Limit (KB)
                </label>
                <input
                  {...register('memory_limit')}
                  type="number"
                  className="w-full px-3 py-2 border border-border rounded-xl bg-white text-foreground"
                  placeholder="262144"
                />
                {errors.memory_limit && (
                  <p className="mt-1 text-sm text-red-600 dark:text-red-400">{errors.memory_limit.message}</p>
                )}
                <p className="mt-1 text-xs text-muted-foreground">
                  Common values: 262144 (256MB), 524288 (512MB)
                </p>
              </div>

              {/* Difficulty */}
              <div>
                <label className="block text-sm font-medium text-foreground mb-1">
                  Difficulty
                </label>
                <select
                  {...register('difficulty')}
                  className="w-full px-3 py-2 border border-border rounded-xl bg-white text-foreground"
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
                <label className="block text-sm font-medium text-foreground mb-1">
                  Points
                </label>
                <input
                  {...register('points')}
                  type="number"
                  className="w-full px-3 py-2 border border-border rounded-xl bg-white text-foreground"
                  placeholder="100"
                />
                {errors.points && (
                  <p className="mt-1 text-sm text-red-600 dark:text-red-400">{errors.points.message}</p>
                )}
              </div>
            </div>

            {/* Problem Statement Editor */}
            <div>
              <label className="block text-sm font-medium text-foreground mb-1">
                Problem Statement
              </label>

              <div className="grid grid-cols-1 md:grid-cols-3 gap-3 mb-3">
                <div>
                  <label className="block text-xs text-muted-foreground mb-1">Format</label>
                  <select
                    value={statementFormat}
                    onChange={(e) => setStatementFormat(e.target.value as 'markdown' | 'html' | 'plain' | 'pdf')}
                    className="w-full px-3 py-2 border border-border rounded-xl bg-white text-foreground"
                  >
                    <option value="markdown">markdown</option>
                    <option value="html">html</option>
                    <option value="plain">plain</option>
                    <option value="pdf">pdf</option>
                  </select>
                </div>
                <div>
                  <label className="block text-xs text-muted-foreground mb-1">Language</label>
                  <input
                    type="text"
                    value={statementLanguage}
                    onChange={(e) => setStatementLanguage(e.target.value)}
                    className="w-full px-3 py-2 border border-border rounded-xl bg-white text-foreground"
                    placeholder="en"
                  />
                </div>
                <div>
                  <label className="block text-xs text-muted-foreground mb-1">Title</label>
                  <input
                    type="text"
                    value={statementTitle}
                    onChange={(e) => setStatementTitle(e.target.value)}
                    className="w-full px-3 py-2 border border-border rounded-xl bg-white text-foreground"
                    placeholder="Problem title"
                  />
                </div>
              </div>

              <div className="flex items-center gap-2 mb-3">
                <button
                  type="button"
                  onClick={() => setPreviewMode('edit')}
                  className={`px-3 py-1.5 rounded text-sm ${previewMode === 'edit' ? 'bg-primary text-white' : 'bg-muted text-foreground'}`}
                >
                  Edit
                </button>
                <button
                  type="button"
                  onClick={() => setPreviewMode('preview')}
                  className={`px-3 py-1.5 rounded text-sm ${previewMode === 'preview' ? 'bg-primary text-white' : 'bg-muted text-foreground'}`}
                >
                  Preview
                </button>
                <button
                  type="button"
                  onClick={() => setPreviewMode('split')}
                  className={`px-3 py-1.5 rounded text-sm ${previewMode === 'split' ? 'bg-primary text-white' : 'bg-muted text-foreground'}`}
                >
                  Split
                </button>
              </div>

              <div className={`grid gap-3 ${previewMode === 'split' ? 'grid-cols-1 md:grid-cols-2' : 'grid-cols-1'}`}>
                {previewMode !== 'preview' && (
                  <div className="border border-border rounded-xl overflow-hidden">
                    <Editor
                      height="320px"
                      defaultLanguage="markdown"
                      language={getEditorLanguage(statementFormat)}
                      value={statementContent}
                      onChange={(value) => setStatementContent(value || '')}
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
                )}

                {previewMode !== 'edit' && (
                  <div className="border border-border rounded-xl p-4 bg-muted/40 overflow-auto min-h-[320px]">
                    <div className="prose max-w-none dark:prose-invert">
                      {renderStatementPreview()}
                    </div>
                  </div>
                )}
              </div>

              <p className="mt-1 text-xs text-muted-foreground">
                Supports multiple statement formats and live preview.
              </p>
            </div>

            {/* Submit Button */}
            <div className="flex gap-3">
              <button
                type="submit"
                disabled={isSubmitting || createMutation.isPending || updateMutation?.isPending}
                className="px-4 py-2 bg-green-600 hover:bg-green-700 disabled:opacity-50 text-white rounded-xl font-medium transition-colors"
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
                className="px-4 py-2 bg-muted hover:bg-muted/80 text-foreground rounded-xl font-medium transition-colors"
              >
                Cancel
              </button>
            </div>

            {/* Error Messages */}
            {(createMutation.isError || updateMutation?.isError) && (
              <div className="p-3 bg-red-100 dark:bg-red-900/50 text-red-700 dark:text-red-400 rounded-xl">
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
        <div className="text-center py-10 text-muted-foreground">Loading...</div>
      ) : problems.length === 0 ? (
        <div className="text-center py-10 text-muted-foreground">
          No problems available. Create one above.
        </div>
      ) : (
        <div className="overflow-x-auto">
          <table className="min-w-full bg-card rounded-xl shadow">
            <thead className="bg-muted">
              <tr>
                <th className="px-4 py-3 text-left text-sm font-medium text-foreground">Name</th>
                <th className="px-4 py-3 text-left text-sm font-medium text-foreground">Difficulty</th>
                <th className="px-4 py-3 text-left text-sm font-medium text-foreground">Time</th>
                <th className="px-4 py-3 text-left text-sm font-medium text-foreground">Memory</th>
                <th className="px-4 py-3 text-left text-sm font-medium text-foreground">Points</th>
                <th className="px-4 py-3 text-left text-sm font-medium text-foreground">Actions</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-200 dark:divide-gray-700">
              {problems.map((problem) => (
                <tr key={problem.id} className="hover:bg-muted/60">
                  <td className="px-4 py-3 text-sm text-foreground">
                    {problem.name}
                  </td>
                  <td className="px-4 py-3 text-sm">
                    <span className={`px-2 py-1 rounded text-xs font-medium ${difficultyColors[problem.difficulty as keyof typeof difficultyColors] || 'bg-muted text-foreground'}`}>
                      {problem.difficulty}
                    </span>
                  </td>
                  <td className="px-4 py-3 text-sm text-foreground">
                    {problem.time_limit}s
                  </td>
                  <td className="px-4 py-3 text-sm text-foreground">
                    {(problem.memory_limit / 1024).toFixed(0)} MB
                  </td>
                  <td className="px-4 py-3 text-sm text-foreground">
                    {problem.points}
                  </td>
                  <td className="px-4 py-3 text-sm">
                    <div className="flex gap-2">
                      <button
                        onClick={() => handleEdit(problem)}
                        className="text-primary hover:text-primary   font-medium"
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
                            className="text-muted-foreground hover:text-foreground dark:text-muted-foreground  font-medium"
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
                        className="text-muted-foreground hover:text-foreground dark:text-muted-foreground  font-medium"
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
          className="text-muted-foreground hover:text-foreground dark:text-muted-foreground "
        >
          ← Back to Admin Dashboard
        </button>
      </div>
    </div>
  )
}