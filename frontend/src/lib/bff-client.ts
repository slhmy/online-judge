const BFF_URL = process.env.NEXT_PUBLIC_BFF_URL || 'http://localhost:8080'

export class BFFClient {
  private baseUrl: string
  private token: string | null = null

  constructor(baseUrl: string = BFF_URL) {
    this.baseUrl = baseUrl
  }

  setToken(token: string) {
    this.token = token
  }

  private async fetch<T>(path: string, options?: RequestInit): Promise<T> {
    const headers: HeadersInit = {
      'Content-Type': 'application/json',
    }
    if (this.token) {
      headers['Authorization'] = `Bearer ${this.token}`
    }

    const res = await fetch(`${this.baseUrl}${path}`, {
      ...options,
      headers,
    })

    if (!res.ok) {
      throw new Error(`BFF error: ${res.status}`)
    }

    return res.json()
  }

  // Problems
  async getProblems(page = 1, pageSize = 20) {
    return this.fetch(`/api/v1/problems?page=${page}&page_size=${pageSize}`)
  }

  async getProblem(id: string) {
    return this.fetch(`/api/v1/problems/${id}`)
  }

  // Submissions
  async createSubmission(data: { problemId: string; languageId: string; sourceCode: string }) {
    return this.fetch('/api/v1/submissions', {
      method: 'POST',
      body: JSON.stringify(data),
    })
  }

  async getSubmission(id: string) {
    return this.fetch(`/api/v1/submissions/${id}`)
  }

  async getSubmissions(page = 1, pageSize = 20) {
    return this.fetch(`/api/v1/submissions?page=${page}&page_size=${pageSize}`)
  }

  async getJudgingRuns(submissionId: string) {
    return this.fetch(`/api/v1/submissions/${submissionId}/runs`)
  }

  async getTestCaseOutput(testCaseId: string, type: 'output' | 'diff' | 'error') {
    return this.fetch(`/api/v1/testcases/${testCaseId}/${type}`)
  }

  async rejudgeSubmission(submissionId: string) {
    return this.fetch(`/api/v1/submissions/${submissionId}/rejudge`, {
      method: 'POST',
    })
  }

  // Contests
  async getContests(page = 1, pageSize = 20) {
    return this.fetch(`/api/v1/contests?page=${page}&page_size=${pageSize}`)
  }

  async getContest(id: string) {
    return this.fetch(`/api/v1/contests/${id}`)
  }

  async getScoreboard(contestId: string) {
    return this.fetch(`/api/v1/contests/${contestId}/scoreboard`)
  }

  async registerContest(contestId: string, data: { team_name: string; affiliation?: string }) {
    return this.fetch(`/api/v1/contests/${contestId}/register`, {
      method: 'POST',
      body: JSON.stringify(data),
    })
  }

  // Auth
  async login(email: string, password: string) {
    return this.fetch('/api/v1/auth/login', {
      method: 'POST',
      body: JSON.stringify({ email, password }),
    })
  }

  async getOAuthUrl(provider = 'github') {
    return this.fetch(`/api/v1/auth/oauth/url?provider=${provider}`)
  }

  async oauthCallback(code: string, state: string) {
    return this.fetch(`/api/v1/auth/oauth/callback?code=${code}&state=${state}`)
  }

  async getMe() {
    return this.fetch('/api/v1/auth/me')
  }

  // Admin - Problem CRUD
  async createProblem(data: {
    external_id: string
    name: string
    time_limit: number
    memory_limit: number
    output_limit?: number
    difficulty: string
    points: number
    description?: string
  }) {
    return this.fetch('/api/v1/problems', {
      method: 'POST',
      body: JSON.stringify(data),
    })
  }

  async updateProblem(id: string, data: {
    name?: string
    time_limit?: number
    memory_limit?: number
    output_limit?: number
    difficulty?: string
    points?: number
    is_published?: boolean
    allow_submit?: boolean
    description?: string
  }) {
    return this.fetch(`/api/v1/problems/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    })
  }

  async deleteProblem(id: string) {
    return this.fetch(`/api/v1/problems/${id}`, {
      method: 'DELETE',
    })
  }

  // Admin - User management
  async getUsers() {
    return this.fetch('/api/v1/admin/users')
  }

  async updateUserRole(userId: string, role: string) {
    return this.fetch(`/api/v1/admin/users/${userId}/role`, {
      method: 'PUT',
      body: JSON.stringify({ role }),
    })
  }

  // User profile
  async getUserProfile(userId: string) {
    return this.fetch(`/api/v1/users/${userId}/profile`)
  }

  async updateUserProfile(userId: string, data: {
    display_name?: string
    avatar_url?: string
    bio?: string
    country?: string
  }) {
    return this.fetch(`/api/v1/users/${userId}/profile`, {
      method: 'PUT',
      body: JSON.stringify(data),
    })
  }

  async getUserStats(userId: string) {
    return this.fetch(`/api/v1/users/${userId}/stats`)
  }

  async getUserSubmissions(userId: string, page = 1, pageSize = 20, verdict?: string, problemId?: string) {
    const params = new URLSearchParams({
      page: String(page),
      page_size: String(pageSize),
    })
    if (verdict) params.append('verdict', verdict)
    if (problemId) params.append('problem_id', problemId)
    return this.fetch(`/api/v1/users/${userId}/submissions?${params.toString()}`)
  }

  // Test Run
  async createTestRun(data: { problemId: string; language: string; source: string }) {
    return this.fetch('/api/v1/test-runs', {
      method: 'POST',
      body: JSON.stringify({
        problem_id: data.problemId,
        language: data.language,
        source: data.source,
      }),
    })
  }

  // Admin - Test Case management
  async getTestCases(problemId: string) {
    return this.fetch(`/api/v1/problems/${problemId}/testcases`)
  }

  async createTestCase(problemId: string, data: {
    rank: number
    is_sample: boolean
    input_content: string
    output_content: string
    description?: string
  }) {
    return this.fetch(`/api/v1/problems/${problemId}/testcases`, {
      method: 'POST',
      body: JSON.stringify(data),
    })
  }

  async updateTestCase(testCaseId: string, data: {
    rank?: number
    is_sample?: boolean
    description?: string
  }) {
    return this.fetch(`/api/v1/testcases/${testCaseId}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    })
  }

  async deleteTestCase(testCaseId: string) {
    return this.fetch(`/api/v1/testcases/${testCaseId}`, {
      method: 'DELETE',
    })
  }

  async toggleTestCaseSample(testCaseId: string) {
    return this.fetch(`/api/v1/testcases/${testCaseId}/toggle-sample`, {
      method: 'PUT',
    })
  }

  async batchUploadTestCases(problemId: string, data: FormData) {
    const headers: HeadersInit = {}
    if (this.token) {
      headers['Authorization'] = `Bearer ${this.token}`
    }

    const res = await fetch(`${this.baseUrl}/api/v1/problems/${problemId}/testcases/batch`, {
      method: 'POST',
      headers,
      body: data,
    })

    if (!res.ok) {
      throw new Error(`BFF error: ${res.status}`)
    }

    return res.json()
  }
}

export const bffClient = new BFFClient()