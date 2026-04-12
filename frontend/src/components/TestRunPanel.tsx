'use client'

import { useState } from 'react'
import { TestRunResult, TestRunTestCaseResult } from '@/types'

interface TestRunPanelProps {
  result: TestRunResult | null
  loading: boolean
  onClose: () => void
}

const VERDICT_COLORS: Record<string, string> = {
  'correct': 'text-green-600 bg-green-100 dark:text-green-400 dark:bg-green-900/50',
  'wrong-answer': 'text-red-600 bg-red-100 dark:text-red-400 dark:bg-red-900/50',
  'time-limit': 'text-yellow-600 bg-yellow-100 dark:text-yellow-400 dark:bg-yellow-900/50',
  'memory-limit': 'text-orange-600 bg-orange-100 dark:text-orange-400 dark:bg-orange-900/50',
  'run-error': 'text-purple-600 bg-purple-100 dark:text-purple-400 dark:bg-purple-900/50',
  'compiler-error': 'text-primary bg-primary/20 ',
  'output-limit': 'text-pink-600 bg-pink-100 dark:text-pink-400 dark:bg-pink-900/50',
  'presentation': 'text-muted-foreground bg-muted dark:text-muted-foreground',
}

const VERDICT_LABELS: Record<string, string> = {
  'correct': 'Accepted',
  'wrong-answer': 'Wrong Answer',
  'time-limit': 'Time Limit Exceeded',
  'memory-limit': 'Memory Limit Exceeded',
  'run-error': 'Runtime Error',
  'compiler-error': 'Compilation Error',
  'output-limit': 'Output Limit Exceeded',
  'presentation': 'Presentation Error',
}

export default function TestRunPanel({ result, loading, onClose }: TestRunPanelProps) {
  if (!result && !loading) return null

  const passedCount = result?.test_cases?.filter(tc => tc.pass).length || 0
  const totalCount = result?.test_cases?.length || 0

  return (
    <div className="border-t border-border bg-muted/40">
      {/* Header */}
      <div className="flex items-center justify-between p-3 border-b border-border bg-card">
        <h3 className="font-semibold text-foreground">
          Test Run Results
        </h3>
        <div className="flex items-center gap-2">
          {loading && (
            <span className="text-sm text-muted-foreground animate-pulse">
              Running...
            </span>
          )}
          <button
            onClick={onClose}
            className="text-muted-foreground hover:text-foreground dark:text-muted-foreground "
          >
            ✕
          </button>
        </div>
      </div>

      {/* Content */}
      <div className="p-3 max-h-[40vh] overflow-auto">
        {loading && (
          <div className="flex items-center justify-center py-8">
            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
          </div>
        )}

        {!loading && result && (
          <>
            {/* Compilation Error */}
            {result.verdict === 'compiler-error' && result.compile_error && (
              <div className="mb-4 p-4 bg-red-50 dark:bg-red-900/20 rounded-xl border border-red-200 dark:border-red-800">
                <div className="font-semibold text-red-600 dark:text-red-400 mb-2">
                  Compilation Error
                </div>
                <pre className="text-sm text-red-800 dark:text-red-300 overflow-auto font-mono whitespace-pre-wrap">
                  {result.compile_error}
                </pre>
              </div>
            )}

            {/* Test Case Results */}
            {result.test_cases && result.test_cases.length > 0 && (
              <div className="space-y-3">
                {result.test_cases.map((tc, idx) => (
                  <TestCaseCard key={tc.test_case_id} testCase={tc} index={idx} />
                ))}
              </div>
            )}

            {/* Summary */}
            {result.test_cases && result.test_cases.length > 0 && (
              <div className="mt-4 p-3 bg-card rounded-xl border border-border">
                <div className="flex items-center justify-between">
                  <span className={`px-2 py-1 rounded text-sm font-medium ${VERDICT_COLORS[result.verdict] || 'text-muted-foreground bg-muted'}`}>
                    {VERDICT_LABELS[result.verdict] || result.verdict}
                  </span>
                  <div className="text-sm text-muted-foreground">
                    <span className="mr-4">{passedCount}/{totalCount} passed</span>
                    <span className="mr-4">Time: {result.runtime.toFixed(3)}s</span>
                    <span>Memory: {(result.memory / 1024).toFixed(0)} MB</span>
                  </div>
                </div>
              </div>
            )}
          </>
        )}
      </div>
    </div>
  )
}

function TestCaseCard({ testCase, index }: { testCase: TestRunTestCaseResult; index: number }) {
  const [showDetails, setShowDetails] = useState(false)

  const verdictClass = VERDICT_COLORS[testCase.verdict] || 'text-muted-foreground bg-muted'
  const verdictLabel = VERDICT_LABELS[testCase.verdict] || testCase.verdict

  return (
    <div className="p-3 bg-card rounded-xl border border-border">
      {/* Header */}
      <div className="flex items-center justify-between mb-2">
        <div className="flex items-center gap-2">
          <span className="font-medium text-foreground">
            Sample {index + 1}
          </span>
          <span className={`px-2 py-0.5 rounded text-xs font-medium ${verdictClass}`}>
            {testCase.pass ? '✓' : '✗'} {verdictLabel}
          </span>
        </div>
        <div className="flex items-center gap-3 text-xs text-muted-foreground">
          <span>{testCase.runtime.toFixed(3)}s</span>
          <span>{(testCase.memory / 1024).toFixed(0)} MB</span>
          <button
            onClick={() => setShowDetails(!showDetails)}
            className="text-primary hover:text-primary  "
          >
            {showDetails ? 'Hide' : 'Show'} details
          </button>
        </div>
      </div>

      {/* Details */}
      {showDetails && (
        <div className="grid grid-cols-3 gap-3 mt-3">
          <div>
            <label className="text-xs font-medium text-muted-foreground mb-1 block">
              Input
            </label>
            <pre className="bg-muted p-2 rounded text-xs text-foreground overflow-auto max-h-24 font-mono">
              {testCase.input || '(empty)'}
            </pre>
          </div>
          <div>
            <label className="text-xs font-medium text-muted-foreground mb-1 block">
              Expected
            </label>
            <pre className="bg-muted p-2 rounded text-xs text-foreground overflow-auto max-h-24 font-mono">
              {testCase.expected || '(empty)'}
            </pre>
          </div>
          <div>
            <label className="text-xs font-medium text-muted-foreground mb-1 block">
              Output
            </label>
            <pre className={`bg-muted p-2 rounded text-xs overflow-auto max-h-24 font-mono ${
              testCase.pass ? 'text-foreground' : 'text-red-600 dark:text-red-400'
            }`}>
              {testCase.output || '(empty)'}
            </pre>
          </div>
        </div>
      )}
    </div>
  )
}