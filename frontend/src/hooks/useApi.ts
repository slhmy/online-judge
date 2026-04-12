import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { bffClient } from '@/lib/bff-client'

// Problems
export function useProblems(page = 1, pageSize = 20) {
  return useQuery({
    queryKey: ['problems', page, pageSize],
    queryFn: () => bffClient.getProblems(page, pageSize),
  })
}

export function useProblem(id: string) {
  return useQuery({
    queryKey: ['problem', id],
    queryFn: () => bffClient.getProblem(id),
    enabled: !!id,
  })
}

// Submissions
export function useSubmissions(page = 1, pageSize = 20) {
  return useQuery({
    queryKey: ['submissions', page, pageSize],
    queryFn: () => bffClient.getSubmissions(page, pageSize),
  })
}

export function useSubmission(id: string) {
  return useQuery({
    queryKey: ['submission', id],
    queryFn: () => bffClient.getSubmission(id),
    enabled: !!id,
  })
}

export function useJudgingRuns(submissionId: string) {
  return useQuery({
    queryKey: ['judgingRuns', submissionId],
    queryFn: () => bffClient.getJudgingRuns(submissionId),
    enabled: !!submissionId,
  })
}

export function useRejudgeSubmission() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (submissionId: string) => bffClient.rejudgeSubmission(submissionId),
    onSuccess: (_, submissionId) => {
      queryClient.invalidateQueries({ queryKey: ['submission', submissionId] })
      queryClient.invalidateQueries({ queryKey: ['judgingRuns', submissionId] })
      queryClient.invalidateQueries({ queryKey: ['submissions'] })
    },
  })
}

export function useCreateSubmission() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (data: { problemId: string; contestId?: string; languageId: string; sourceCode: string }) =>
      bffClient.createSubmission(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['submissions'] })
    },
  })
}

// Contests
export function useContests(page = 1, pageSize = 20) {
  return useQuery({
    queryKey: ['contests', page, pageSize],
    queryFn: () => bffClient.getContests(page, pageSize),
  })
}

export function useContest(id: string) {
  return useQuery({
    queryKey: ['contest', id],
    queryFn: () => bffClient.getContest(id),
    enabled: !!id,
  })
}

export function useScoreboard(contestId: string) {
  return useQuery({
    queryKey: ['scoreboard', contestId],
    queryFn: () => bffClient.getScoreboard(contestId),
    enabled: !!contestId,
    refetchInterval: 5000, // Refresh every 5 seconds during contests
  })
}

export function useRegisterContest(contestId: string) {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (data: { team_name: string; affiliation?: string }) =>
      bffClient.registerContest(contestId, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['contest', contestId] })
      queryClient.invalidateQueries({ queryKey: ['scoreboard', contestId] })
    },
  })
}

// Auth
export function useLogin() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ email, password }: { email: string; password: string }) =>
      bffClient.login(email, password),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['user'] })
    },
  })
}

export function useMe() {
  return useQuery({
    queryKey: ['user'],
    queryFn: () => bffClient.getMe(),
    retry: false,
  })
}

// Admin - Problem CRUD
export function useCreateProblem() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (data: {
      name: string
      time_limit: number
      memory_limit: number
      difficulty: string
      points: number
      description?: string
    }) => bffClient.createProblem(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['problems'] })
    },
  })
}

export function useUpdateProblem(id: string) {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (data: {
      name?: string
      time_limit?: number
      memory_limit?: number
      difficulty?: string
      points?: number
      is_published?: boolean
      allow_submit?: boolean
      description?: string
    }) => bffClient.updateProblem(id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['problems'] })
      queryClient.invalidateQueries({ queryKey: ['problem', id] })
    },
  })
}

export function useDeleteProblem() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (id: string) => bffClient.deleteProblem(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['problems'] })
    },
  })
}

// User profile
export function useUserProfile(userId: string) {
  return useQuery({
    queryKey: ['userProfile', userId],
    queryFn: () => bffClient.getUserProfile(userId),
    enabled: !!userId,
  })
}

export function useUserStats(userId: string) {
  return useQuery({
    queryKey: ['userStats', userId],
    queryFn: () => bffClient.getUserStats(userId),
    enabled: !!userId,
  })
}

export function useUserSubmissions(userId: string, page = 1, pageSize = 20, verdict?: string, problemId?: string) {
  return useQuery({
    queryKey: ['userSubmissions', userId, page, pageSize, verdict, problemId],
    queryFn: () => bffClient.getUserSubmissions(userId, page, pageSize, verdict, problemId),
    enabled: !!userId,
  })
}

export function useUpdateUserProfile(userId: string) {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (data: {
      display_name?: string
      avatar_url?: string
      bio?: string
      country?: string
    }) => bffClient.updateUserProfile(userId, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['userProfile', userId] })
    },
  })
}

// Admin - Test Case CRUD
export function useTestCases(problemId: string) {
  return useQuery({
    queryKey: ['testCases', problemId],
    queryFn: () => bffClient.getTestCases(problemId),
    enabled: !!problemId,
  })
}

export function useCreateTestCase() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (data: { problemId: string; rank: number; is_sample: boolean; input_content: string; output_content: string; description?: string }) =>
      bffClient.createTestCase(data.problemId, {
        rank: data.rank,
        is_sample: data.is_sample,
        input_content: data.input_content,
        output_content: data.output_content,
        description: data.description,
      }),
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({ queryKey: ['testCases', variables.problemId] })
    },
  })
}

export function useUpdateTestCase() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (data: { testCaseId: string; rank?: number; is_sample?: boolean; description?: string }) =>
      bffClient.updateTestCase(data.testCaseId, {
        rank: data.rank,
        is_sample: data.is_sample,
        description: data.description,
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['testCases'] })
    },
  })
}

export function useDeleteTestCase() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (testCaseId: string) => bffClient.deleteTestCase(testCaseId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['testCases'] })
    },
  })
}

export function useToggleTestCaseSample() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (testCaseId: string) => bffClient.toggleTestCaseSample(testCaseId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['testCases'] })
    },
  })
}

export function useBatchUploadTestCases() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (data: { problemId: string; formData: FormData }) =>
      bffClient.batchUploadTestCases(data.problemId, data.formData),
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({ queryKey: ['testCases', variables.problemId] })
    },
  })
}