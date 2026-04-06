'use client'

import { useEffect, useState } from 'react'
import { useParams } from 'next/navigation'
import dynamic from 'next/dynamic'

const MonacoEditor = dynamic(() => import('@monaco-editor/react'), { ssr: false })

const BFF_URL = process.env.NEXT_PUBLIC_BFF_URL || 'http://localhost:8080'

const LANGUAGES = [
  { id: 'cpp', name: 'C++ 17', monacoLang: 'cpp' },
  { id: 'python3', name: 'Python 3', monacoLang: 'python' },
  { id: 'java', name: 'Java 17', monacoLang: 'java' },
  { id: 'go', name: 'Go 1.21', monacoLang: 'go' },
  { id: 'rust', name: 'Rust', monacoLang: 'rust' },
  { id: 'nodejs', name: 'Node.js 18', monacoLang: 'javascript' },
]

interface TestCase {
  id: string
  rank: number
  is_sample: boolean
  input_path: string
  output_path: string
  input_content: string
  output_content: string
  description: string
}

interface Problem {
  id: string
  external_id: string
  name: string
  time_limit: number
  memory_limit: number
  difficulty: string
  points: number
}

interface ProblemResponse {
  problem: Problem
  sample_test_cases: TestCase[]
}

export default function ProblemDetailPage() {
  const params = useParams()
  const problemId = params.id as string

  const [problem, setProblem] = useState<Problem | null>(null)
  const [testCases, setTestCases] = useState<TestCase[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const [language, setLanguage] = useState('cpp')
  const [code, setCode] = useState('')
  const [submitting, setSubmitting] = useState(false)

  useEffect(() => {
    async function fetchProblem() {
      try {
        const url = `${BFF_URL}/api/v1/problems/${problemId}`
        console.log('Fetching from:', url)
        const res = await fetch(url)
        if (!res.ok) {
          throw new Error(`HTTP error! status: ${res.status}`)
        }
        const data: ProblemResponse = await res.json()
        console.log('Received data:', data)
        setProblem(data.problem)
        setTestCases(data.sample_test_cases || [])
      } catch (err) {
        console.error('Fetch error:', err)
        setError(err instanceof Error ? err.message : 'Failed to load problem')
      } finally {
        setLoading(false)
      }
    }

    if (problemId) {
      fetchProblem()
    }
  }, [problemId])

  const handleSubmit = async () => {
    if (!code.trim()) {
      alert('Please write some code before submitting')
      return
    }

    setSubmitting(true)
    try {
      const res = await fetch(`${BFF_URL}/api/v1/submissions`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          problem_id: problemId,
          language: language,
          source: code,
        }),
      })

      if (!res.ok) {
        throw new Error(`HTTP error! status: ${res.status}`)
      }

      const data = await res.json()
      alert(`Submission created! ID: ${data.id}\nStatus: Queued for judging`)
    } catch (error) {
      console.error('Submission error:', error)
      alert('Submission failed: ' + (error instanceof Error ? error.message : 'Unknown error'))
    } finally {
      setSubmitting(false)
    }
  }

  if (loading) {
    return (
      <div className="flex h-[calc(100vh-4rem)] items-center justify-center">
        <div className="text-gray-600 dark:text-gray-400">Loading problem...</div>
      </div>
    )
  }

  if (error) {
    return (
      <div className="flex h-[calc(100vh-4rem)] items-center justify-center">
        <div className="text-red-500 dark:text-red-400">Error: {error}</div>
      </div>
    )
  }

  if (!problem) {
    return (
      <div className="flex h-[calc(100vh-4rem)] items-center justify-center">
        <div className="text-gray-600 dark:text-gray-500">Problem not found</div>
      </div>
    )
  }

  return (
    <div className="flex h-[calc(100vh-4rem)]">
      {/* Problem Description */}
      <div className="w-1/2 p-4 overflow-auto border-r border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800">
        <h1 className="text-2xl font-bold mb-2 text-gray-900 dark:text-gray-100">{problem.external_id}. {problem.name}</h1>
        <div className="flex gap-2 mb-4">
          <span className={`px-2 py-1 rounded text-xs font-medium ${
            problem.difficulty === 'easy' ? 'text-green-600 bg-green-100 dark:text-green-400 dark:bg-green-900/50' :
            problem.difficulty === 'medium' ? 'text-yellow-600 bg-yellow-100 dark:text-yellow-400 dark:bg-yellow-900/50' :
            'text-red-600 bg-red-100 dark:text-red-400 dark:bg-red-900/50'
          }`}>
            {problem.difficulty}
          </span>
          <span className="px-2 py-1 rounded text-xs font-medium bg-blue-100 dark:bg-blue-900/50 text-blue-600 dark:text-blue-400">
            {problem.points} points
          </span>
        </div>

        <div className="mb-4 text-sm text-gray-600 dark:text-gray-400">
          <span className="mr-4">Time Limit: {problem.time_limit}s</span>
          <span>Memory Limit: {(problem.memory_limit / 1024).toFixed(0)} MB</span>
        </div>

        <div className="prose prose-invert max-w-none">
          <h2 className="text-gray-700 dark:text-gray-200">Description</h2>
          <p className="text-gray-600 dark:text-gray-300">Problem description will be loaded from the problem statement file.</p>

          <h2 className="text-gray-700 dark:text-gray-200">Sample Test Cases ({testCases.length})</h2>
          {testCases.length > 0 ? (
            testCases.map((tc, idx) => (
              <div key={tc.id} className="mb-6 p-4 bg-gray-100 dark:bg-gray-700/50 rounded-lg border border-gray-200 dark:border-gray-600">
                <h3 className="font-semibold text-gray-800 dark:text-gray-200 mb-2">Sample {idx + 1}</h3>
                {tc.description && (
                  <p className="text-sm text-gray-600 dark:text-gray-400 mb-3">{tc.description}</p>
                )}
                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <label className="text-sm font-medium text-gray-700 dark:text-gray-300 mb-1 block">Input</label>
                    <pre className="bg-gray-200 dark:bg-gray-900 p-3 rounded text-sm text-gray-800 dark:text-gray-200 overflow-auto max-h-40 font-mono">
                      {tc.input_content || 'No input data'}
                    </pre>
                  </div>
                  <div>
                    <label className="text-sm font-medium text-gray-700 dark:text-gray-300 mb-1 block">Output</label>
                    <pre className="bg-gray-200 dark:bg-gray-900 p-3 rounded text-sm text-gray-800 dark:text-gray-200 overflow-auto max-h-40 font-mono">
                      {tc.output_content || 'No output data'}
                    </pre>
                  </div>
                </div>
              </div>
            ))
          ) : (
            <p className="text-gray-500">No sample test cases available.</p>
          )}
        </div>
      </div>

      {/* Code Editor */}
      <div className="w-1/2 flex flex-col bg-gray-50 dark:bg-gray-900">
        <div className="flex items-center justify-between p-2 border-b border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800">
          <select
            value={language}
            onChange={(e) => setLanguage(e.target.value)}
            className="border border-gray-300 dark:border-gray-600 rounded px-2 py-1 bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-200"
          >
            {LANGUAGES.map((lang) => (
              <option key={lang.id} value={lang.id}>
                {lang.name}
              </option>
            ))}
          </select>

          <button
            onClick={handleSubmit}
            disabled={submitting}
            className="bg-blue-600 text-white px-4 py-1.5 rounded hover:bg-blue-700 disabled:bg-gray-600"
          >
            {submitting ? 'Submitting...' : 'Submit'}
          </button>
        </div>

        <div className="flex-1">
          <MonacoEditor
            height="100%"
            language={LANGUAGES.find((l) => l.id === language)?.monacoLang || 'plaintext'}
            theme="vs-dark"
            value={code}
            onChange={(value) => setCode(value || '')}
            options={{
              minimap: { enabled: false },
              fontSize: 14,
              automaticLayout: true,
            }}
          />
        </div>
      </div>
    </div>
  )
}