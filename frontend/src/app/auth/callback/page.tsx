'use client'

import { Suspense, useEffect, useState } from 'react'
import { useRouter, useSearchParams } from 'next/navigation'
import { useAuthStore } from '@/stores/authStore'

function AuthCallbackContent() {
  const router = useRouter()
  const searchParams = useSearchParams()
  const { login } = useAuthStore()

  const [status, setStatus] = useState<'loading' | 'success' | 'error'>('loading')
  const [error, setError] = useState('')

  useEffect(() => {
    const code = searchParams.get('code')
    const state = searchParams.get('state')

    if (!code || !state) {
      setStatus('error')
      setError('Missing authorization code or state')
      return
    }

    // Call the BFF OAuth callback endpoint
    fetch(`/api/v1/auth/oauth/callback?code=${code}&state=${state}`)
      .then(async (res) => {
        if (!res.ok) {
          const data = await res.json()
          throw new Error(data.error || 'OAuth callback failed')
        }
        return res.json()
      })
      .then((data) => {
        login(data.user)
        setStatus('success')
        setTimeout(() => router.push('/'), 1000)
      })
      .catch((err) => {
        setStatus('error')
        setError(err instanceof Error ? err.message : 'OAuth callback failed')
      })
  }, [searchParams, login, router])

  return (
    <div className="min-h-[80vh] flex items-center justify-center">
      <div className="bg-white dark:bg-gray-800 p-8 rounded-lg shadow-md w-full max-w-md text-center">
        {status === 'loading' && (
          <div className="flex flex-col items-center gap-4">
            <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600"></div>
            <p className="text-gray-600 dark:text-gray-400">Completing authentication...</p>
          </div>
        )}

        {status === 'success' && !error && (
          <div className="flex flex-col items-center gap-4">
            <div className="text-green-500 text-5xl">&#10003;</div>
            <p className="text-gray-900 dark:text-gray-100 font-medium">Successfully logged in!</p>
            <p className="text-gray-500 dark:text-gray-400 text-sm">Redirecting...</p>
          </div>
        )}

        {status === 'success' && error && (
          <div className="flex flex-col items-center gap-4">
            <div className="text-green-500 text-5xl">&#10003;</div>
            <p className="text-gray-900 dark:text-gray-100 font-medium">Authentication successful!</p>
            <p className="text-blue-600 dark:text-blue-400">{error}</p>
            <p className="text-gray-500 dark:text-gray-400 text-sm">Redirecting to login...</p>
          </div>
        )}

        {status === 'error' && (
          <div className="flex flex-col items-center gap-4">
            <div className="text-red-500 text-5xl">&#10007;</div>
            <p className="text-gray-900 dark:text-gray-100 font-medium">Authentication failed</p>
            <p className="text-red-600 dark:text-red-400">{error}</p>
            <button
              onClick={() => router.push('/login')}
              className="mt-4 px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 transition-colors"
            >
              Back to Login
            </button>
          </div>
        )}
      </div>
    </div>
  )
}

export default function AuthCallbackPage() {
  return (
    <Suspense fallback={
      <div className="min-h-[80vh] flex items-center justify-center">
        <div className="bg-white dark:bg-gray-800 p-8 rounded-lg shadow-md w-full max-w-md text-center">
          <div className="flex flex-col items-center gap-4">
            <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600"></div>
            <p className="text-gray-600 dark:text-gray-400">Loading...</p>
          </div>
        </div>
      </div>
    }>
      <AuthCallbackContent />
    </Suspense>
  )
}