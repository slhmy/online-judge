'use client'

import { useState } from 'react'
import { useRegisterContest } from '@/hooks/useApi'

interface RegistrationModalProps {
  contestId: string
  contestName: string
  isOpen: boolean
  onClose: () => void
  onSuccess: () => void
}

export function RegistrationModal({
  contestId,
  contestName,
  isOpen,
  onClose,
  onSuccess,
}: RegistrationModalProps) {
  const [teamName, setTeamName] = useState('')
  const [affiliation, setAffiliation] = useState('')
  const [error, setError] = useState<string | null>(null)

  const registerMutation = useRegisterContest(contestId)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError(null)

    if (!teamName.trim()) {
      setError('Team name is required')
      return
    }

    try {
      await registerMutation.mutateAsync({
        team_name: teamName.trim(),
        affiliation: affiliation.trim() || undefined,
      })
      onSuccess()
      onClose()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Registration failed')
    }
  }

  if (!isOpen) return null

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div className="bg-white dark:bg-gray-800 rounded-lg shadow-xl max-w-md w-full mx-4">
        <div className="px-6 py-4 border-b border-gray-200 dark:border-gray-700">
          <h2 className="text-xl font-semibold text-gray-900 dark:text-gray-100">
            Register for {contestName}
          </h2>
        </div>

        <form onSubmit={handleSubmit} className="px-6 py-4 space-y-4">
          {/* Team Name */}
          <div>
            <label
              htmlFor="teamName"
              className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
            >
              Team Name *
            </label>
            <input
              type="text"
              id="teamName"
              value={teamName}
              onChange={(e) => setTeamName(e.target.value)}
              className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 focus:outline-none focus:ring-2 focus:ring-blue-500"
              placeholder="Enter your team name"
              disabled={registerMutation.isPending}
            />
          </div>

          {/* Affiliation */}
          <div>
            <label
              htmlFor="affiliation"
              className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
            >
              Affiliation (optional)
            </label>
            <input
              type="text"
              id="affiliation"
              value={affiliation}
              onChange={(e) => setAffiliation(e.target.value)}
              className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 focus:outline-none focus:ring-2 focus:ring-blue-500"
              placeholder="School/Company/Organization"
              disabled={registerMutation.isPending}
            />
          </div>

          {/* Error message */}
          {error && (
            <div className="px-3 py-2 bg-red-100 dark:bg-red-900 text-red-700 dark:text-red-300 rounded-md text-sm">
              {error}
            </div>
          )}

          {/* Actions */}
          <div className="flex gap-3 justify-end pt-2">
            <button
              type="button"
              onClick={onClose}
              disabled={registerMutation.isPending}
              className="px-4 py-2 text-gray-700 dark:text-gray-300 hover:text-gray-900 dark:hover:text-gray-100 disabled:opacity-50"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={registerMutation.isPending}
              className="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 disabled:bg-blue-400 disabled:cursor-not-allowed"
            >
              {registerMutation.isPending ? 'Registering...' : 'Register'}
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}