'use client'

import { useEffect, useState } from 'react'
import { useParams } from 'next/navigation'
import dynamic from 'next/dynamic'
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import remarkMath from 'remark-math'
import rehypeKatex from 'rehype-katex'
import TestRunPanel from '@/components/TestRunPanel'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { TestRunResult } from '@/types'
import 'katex/dist/katex.min.css'

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

const DIFFICULTY_BADGE_CLASS: Record<string, string> = {
  easy: 'bg-emerald-100 text-emerald-800 dark:bg-emerald-500/20 dark:text-emerald-300',
  medium: 'bg-amber-100 text-amber-800 dark:bg-amber-500/20 dark:text-amber-300',
  hard: 'bg-red-100 text-red-800 dark:bg-red-500/20 dark:text-red-300',
}

export default function ProblemDetailPage() {
  const params = useParams()
  const problemId = params.id as string

  const [problem, setProblem] = useState<Problem | null>(null)
  const [testCases, setTestCases] = useState<TestCase[]>([])
  const [problemStatement, setProblemStatement] = useState<{ format: string; content: string } | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const [language, setLanguage] = useState('cpp')
  const [code, setCode] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [running, setRunning] = useState(false)
  const [testRunResult, setTestRunResult] = useState<TestRunResult | null>(null)

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

        // Fetch problem statement
        const statementUrl = `${BFF_URL}/api/v1/problems/${problemId}/statement?language=en`
        const statementRes = await fetch(statementUrl)
        if (statementRes.ok) {
          const statementData = await statementRes.json()
          if (statementData && statementData.format && statementData.content) {
            setProblemStatement({ format: statementData.format, content: statementData.content })
          }
        }
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

  const handleRun = async () => {
    if (!code.trim()) {
      alert('Please write some code before running')
      return
    }

    setRunning(true)
    setTestRunResult(null)

    try {
      const res = await fetch(`${BFF_URL}/api/v1/test-runs`, {
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
        const errorText = await res.text()
        throw new Error(`HTTP error! status: ${res.status} - ${errorText}`)
      }

      const data: TestRunResult = await res.json()
      setTestRunResult(data)
    } catch (error) {
      console.error('Test run error:', error)
      alert('Test run failed: ' + (error instanceof Error ? error.message : 'Unknown error'))
    } finally {
      setRunning(false)
    }
  }

  const detailViewportHeight = 'calc(100dvh - var(--app-header-h, 64px) - var(--app-footer-h, 64px) - 3rem)'

  if (loading) {
    return (
      <div className="flex min-h-[40vh] items-center justify-center" style={{ height: detailViewportHeight }}>
        <div className="text-muted-foreground">Loading problem...</div>
      </div>
    )
  }

  if (error) {
    return (
      <div className="flex min-h-[40vh] items-center justify-center" style={{ height: detailViewportHeight }}>
        <div className="text-red-500 dark:text-red-400">Error: {error}</div>
      </div>
    )
  }

  if (!problem) {
    return (
      <div className="flex min-h-[40vh] items-center justify-center" style={{ height: detailViewportHeight }}>
        <div className="text-muted-foreground dark:text-muted-foreground">Problem not found</div>
      </div>
    )
  }

  const hasInlineStatement = Boolean(
    problemStatement &&
      problemStatement.format !== 'pdf' &&
      problemStatement.content &&
      problemStatement.content.trim().length > 0,
  )

  return (
    <div className="flex min-h-0" style={{ height: detailViewportHeight }}>
      {/* Problem Description */}
      <div className="w-1/2 p-4 overflow-auto border-r border-border bg-card">
        <div className="prose max-w-none text-foreground dark:prose-invert prose-headings:text-foreground prose-p:text-foreground prose-li:text-foreground">
          <h1 className="text-2xl font-bold mb-2 text-foreground">{problem.name}</h1>
          <div className="flex gap-2 mb-4">
            <Badge className={DIFFICULTY_BADGE_CLASS[problem.difficulty] || 'bg-muted text-foreground'}>
              {problem.difficulty}
            </Badge>
            <Badge variant="outline" className="bg-primary/10 text-primary border-primary/20">
              {problem.points} points
            </Badge>
          </div>

          <div className="mb-4 text-sm text-muted-foreground">
            <span className="mr-4">Time Limit: {problem.time_limit}s</span>
            <span>Memory Limit: {(problem.memory_limit / 1024).toFixed(0)} MB</span>
          </div>

          {hasInlineStatement ? (
            problemStatement!.format === 'html' ? (
              <div dangerouslySetInnerHTML={{ __html: problemStatement!.content }} />
            ) : (
              <ReactMarkdown
                remarkPlugins={[remarkGfm, remarkMath]}
                rehypePlugins={[rehypeKatex]}
              >
                {problemStatement!.content}
              </ReactMarkdown>
            )
          ) : problemStatement?.format === 'pdf' ? (
            <div className="text-muted-foreground">
              <p>Problem statement is available as a PDF file.</p>
              <a
                href={`${BFF_URL}/api/v1/problems/${problemId}/statement/pdf?language=en`}
                className="text-primary hover:underline"
                target="_blank"
                rel="noopener noreferrer"
              >
                View PDF
              </a>
            </div>
          ) : (
            <p className="text-muted-foreground">No problem statement available.</p>
          )}

          <h2 className="text-foreground">Sample Test Cases ({testCases.length})</h2>
          {testCases.length > 0 ? (
            testCases.map((tc, idx) => (
              <div key={tc.id} className="mb-6 p-4 bg-muted/50 rounded-xl border border-border">
                <h3 className="font-semibold text-foreground mb-2">Sample {idx + 1}</h3>
                {tc.description && (
                  <p className="text-sm text-muted-foreground mb-3">{tc.description}</p>
                )}
                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <label className="text-sm font-medium text-foreground mb-1 block">Input</label>
                    <pre className="bg-muted p-3 rounded text-sm text-foreground overflow-auto max-h-40 font-mono">
                      {tc.input_content || 'No input data'}
                    </pre>
                  </div>
                  <div>
                    <label className="text-sm font-medium text-foreground mb-1 block">Output</label>
                    <pre className="bg-muted p-3 rounded text-sm text-foreground overflow-auto max-h-40 font-mono">
                      {tc.output_content || 'No output data'}
                    </pre>
                  </div>
                </div>
              </div>
            ))
          ) : (
            <p className="text-muted-foreground">No sample test cases available.</p>
          )}
        </div>
      </div>

      {/* Code Editor */}
      <div className="w-1/2 flex flex-col bg-muted/40">
        <div className="flex items-center justify-between p-2 border-b border-border bg-card">
          <select
            value={language}
            onChange={(e) => setLanguage(e.target.value)}
            className="h-9 rounded-[var(--radius)] border border-border bg-background px-3 text-sm text-foreground shadow-sm outline-none focus-visible:border-ring focus-visible:ring-2 focus-visible:ring-ring/40 dark:border-zinc-700 dark:bg-zinc-900 dark:text-zinc-100"
          >
            {LANGUAGES.map((lang) => (
              <option key={lang.id} value={lang.id}>
                {lang.name}
              </option>
            ))}
          </select>

          <div className="flex items-center gap-2">
            <Button
              onClick={handleRun}
              disabled={running}
              size="sm"
              className="min-w-24 !rounded-[var(--radius)] bg-green-600 text-white hover:bg-green-700 dark:bg-emerald-400 dark:text-emerald-950 dark:hover:bg-emerald-300"
            >
              {running ? 'Running...' : 'Run'}
            </Button>
            <Button
              onClick={handleSubmit}
              disabled={submitting}
              size="sm"
              className="min-w-24 !rounded-[var(--radius)] bg-primary text-primary-foreground hover:bg-primary/90 dark:bg-sky-400 dark:text-slate-950 dark:hover:bg-sky-300"
            >
              {submitting ? 'Submitting...' : 'Submit'}
            </Button>
          </div>
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

        {/* Test Run Results Panel */}
        <TestRunPanel
          result={testRunResult}
          loading={running}
          onClose={() => setTestRunResult(null)}
        />
      </div>
    </div>
  )
}