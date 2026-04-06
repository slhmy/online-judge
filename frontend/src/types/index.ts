// Types for the Online Judge platform

export interface Problem {
  id: string
  externalId: string
  name: string
  timeLimit: number
  memoryLimit: number
  difficulty: 'easy' | 'medium' | 'hard'
  points: number
  allowSubmit: boolean
}

export interface TestCase {
  id: string
  problemId: string
  rank: number
  isSample: boolean
  input: string
  output: string
}

export type Verdict =
  | 'correct'
  | 'wrong-answer'
  | 'timelimit'
  | 'memory-limit'
  | 'run-error'
  | 'compiler-error'
  | 'output-limit'
  | 'presentation'

export interface Submission {
  id: string
  userId: string
  problemId: string
  languageId: string
  submitTime: string
}

export interface JudgingRun {
  id: string
  judgingId: string
  testCaseId: string
  rank: number
  runtime: number
  wallTime?: number
  memory: number
  verdict: Verdict
  outputRunPath?: string
  outputDiffPath?: string
  outputErrorPath?: string
}

export interface Judging {
  id: string
  submissionId: string
  verdict: Verdict
  maxRuntime: number
  maxMemory: number
  verified: boolean
  compileSuccess?: boolean
  compileOutput?: string
  startTime?: string
  endTime?: string
}

export interface Language {
  id: string
  name: string
  timeFactor: number
  extensions: string[]
}

export interface Contest {
  id: string
  name: string
  shortName: string
  startTime: string
  endTime: string
  freezeTime?: string
  public: boolean
}

export interface ScoreboardEntry {
  rank: number
  teamId: string
  teamName: string
  numSolved: number
  totalTime: number
  problems: ProblemScore[]
}

export interface ProblemScore {
  problemShortName: string
  numPending: number
  numCorrect: number
  time: number
  isPending: boolean
}

// Verdict display config with icons
export const VERDICT_CONFIG: Record<Verdict, { color: string; label: string; icon: string; bgColor: string }> = {
  correct: { color: 'bg-green-500', label: 'Accepted', icon: '✓', bgColor: 'bg-green-50' },
  'wrong-answer': { color: 'bg-red-500', label: 'Wrong Answer', icon: '✗', bgColor: 'bg-red-50' },
  timelimit: { color: 'bg-yellow-500', label: 'Time Limit Exceeded', icon: '⏱', bgColor: 'bg-yellow-50' },
  'memory-limit': { color: 'bg-orange-500', label: 'Memory Limit Exceeded', icon: '💾', bgColor: 'bg-orange-50' },
  'run-error': { color: 'bg-purple-500', label: 'Runtime Error', icon: '⚠', bgColor: 'bg-purple-50' },
  'compiler-error': { color: 'bg-blue-500', label: 'Compilation Error', icon: '⚙', bgColor: 'bg-blue-50' },
  'output-limit': { color: 'bg-pink-500', label: 'Output Limit Exceeded', icon: '📋', bgColor: 'bg-pink-50' },
  presentation: { color: 'bg-gray-500', label: 'Presentation Error', icon: '📝', bgColor: 'bg-gray-50' },
}

// Supported languages
export const SUPPORTED_LANGUAGES: Language[] = [
  { id: 'cpp', name: 'C++ 17', timeFactor: 1.0, extensions: ['.cpp', '.cc', '.cxx'] },
  { id: 'python3', name: 'Python 3', timeFactor: 2.0, extensions: ['.py'] },
  { id: 'java', name: 'Java 17', timeFactor: 1.5, extensions: ['.java'] },
  { id: 'go', name: 'Go 1.21', timeFactor: 1.0, extensions: ['.go'] },
  { id: 'rust', name: 'Rust 1.70', timeFactor: 1.0, extensions: ['.rs'] },
  { id: 'nodejs', name: 'Node.js 18', timeFactor: 2.0, extensions: ['.js', '.ts'] },
]

// Test Run types
export interface TestRunRequest {
  problemId: string
  language: string
  source: string
}

export interface TestRunTestCaseResult {
  test_case_id: string
  rank: number
  verdict: string
  input: string
  expected: string
  output: string
  runtime: number
  memory: number
  pass: boolean
}

export interface TestRunResult {
  id: string
  status: string
  verdict: string
  runtime: number
  memory: number
  compile_error?: string
  test_cases: TestRunTestCaseResult[]
}