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