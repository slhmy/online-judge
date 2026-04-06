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
  'compiler-error': 'text-blue-600 bg-blue-100 dark:text-blue-400 dark:bg-blue-900/50',
  'output-limit': 'text-pink-600 bg-pink-100 dark:text-pink-400 dark:bg-pink-900/50',
  'presentation': 'text-gray-600 bg-gray-100 dark:text-gray-400 dark:bg-gray-900/50',
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
    <div className="border-t border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-900">
      {/* Header */}
      <div className="flex items-center justify-between p-3 border-b border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800">
        <h3 className="font-semibold text-gray-900 dark:text-gray-100">
          Test Run Results
        </h3>
        <div className="flex items-center gap-2">
          {loading && (
            <span className="text-sm text-gray-500 dark:text-gray-400 animate-pulse">
              Running...
            </span>
          )}
          <button
            onClick={onClose}
            className="text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200"
          >
            ✕
          </button>
        </div>
      </div>

      {/* Content */}
      <div className="p-3 max-h-[40vh] overflow-auto">
        {loading && (
          <div className="flex items-center justify-center py-8">
            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
          </div>
        )}

        {!loading && result && (
          <>
            {/* Compilation Error */}
            {result.verdict === 'compiler-error' && result.compile_error && (
              <div className="mb-4 p-4 bg-red-50 dark:bg-red-900/20 rounded-lg border border-red-200 dark:border-red-800">
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
              <div className="mt-4 p-3 bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700">
                <div className="flex items-center justify-between">
                  <span className={`px-2 py-1 rounded text-sm font-medium ${VERDICT_COLORS[result.verdict] || 'text-gray-600 bg-gray-100'}`}>
                    {VERDICT_LABELS[result.verdict] || result.verdict}
                  </span>
                  <div className="text-sm text-gray-600 dark:text-gray-400">
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

  const verdictClass = VERDICT_COLORS[testCase.verdict] || 'text-gray-600 bg-gray-100'
  const verdictLabel = VERDICT_LABELS[testCase.verdict] || testCase.verdict

  return (
    <div className="p-3 bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700">
      {/* Header */}
      <div className="flex items-center justify-between mb-2">
        <div className="flex items-center gap-2">
          <span className="font-medium text-gray-900 dark:text-gray-100">
            Sample {index + 1}
          </span>
          <span className={`px-2 py-0.5 rounded text-xs font-medium ${verdictClass}`}>
            {testCase.pass ? '✓' : '✗'} {verdictLabel}
          </span>
        </div>
        <div className="flex items-center gap-3 text-xs text-gray-500 dark:text-gray-400">
          <span>{testCase.runtime.toFixed(3)}s</span>
          <span>{(testCase.memory / 1024).toFixed(0)} MB</span>
          <button
            onClick={() => setShowDetails(!showDetails)}
            className="text-blue-600 hover:text-blue-800 dark:text-blue-400 dark:hover:text-blue-300"
          >
            {showDetails ? 'Hide' : 'Show'} details
          </button>
        </div>
      </div>

      {/* Details */}
      {showDetails && (
        <div className="grid grid-cols-3 gap-3 mt-3">
          <div>
            <label className="text-xs font-medium text-gray-500 dark:text-gray-400 mb-1 block">
              Input
            </label>
            <pre className="bg-gray-100 dark:bg-gray-900 p-2 rounded text-xs text-gray-800 dark:text-gray-200 overflow-auto max-h-24 font-mono">
              {testCase.input || '(empty)'}
            </pre>
          </div>
          <div>
            <label className="text-xs font-medium text-gray-500 dark:text-gray-400 mb-1 block">
              Expected
            </label>
            <pre className="bg-gray-100 dark:bg-gray-900 p-2 rounded text-xs text-gray-800 dark:text-gray-200 overflow-auto max-h-24 font-mono">
              {testCase.expected || '(empty)'}
            </pre>
          </div>
          <div>
            <label className="text-xs font-medium text-gray-500 dark:text-gray-400 mb-1 block">
              Output
            </label>
            <pre className={`bg-gray-100 dark:bg-gray-900 p-2 rounded text-xs overflow-auto max-h-24 font-mono ${
              testCase.pass ? 'text-gray-800 dark:text-gray-200' : 'text-red-600 dark:text-red-400'
            }`}>
              {testCase.output || '(empty)'}
            </pre>
          </div>
        </div>
      )}
    </div>
  )
}