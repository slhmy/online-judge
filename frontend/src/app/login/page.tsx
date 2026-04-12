'use client'

import { useState } from 'react'
import { useRouter } from 'next/navigation'
import Link from 'next/link'
import { useAuthStore } from '@/stores/authStore'
import { useAuthError, parseAuthResponse, createTimeoutController } from '@/hooks/useAuthError'
import { ErrorAlert, FieldError } from '@/components/auth/ErrorAlert'
import { cn } from '@/lib/utils'

export default function LoginPage() {
  const router = useRouter()
  const { login } = useAuthStore()

  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [loading, setLoading] = useState(false)
  const [githubLoading, setGithubLoading] = useState(false)

  const {
    error,
    fieldErrors,
    parseError,
    setError,
    clearError,
    clearFieldError,
    hasFieldError,
  } = useAuthError()

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setLoading(true)
    clearError()

    const { controller, timeoutId } = createTimeoutController()

    try {
      const res = await fetch('/api/v1/auth/login', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email, password }),
        signal: controller.signal,
      })

      window.clearTimeout(timeoutId)

      const data = await parseAuthResponse<{ user: any }>(res)

      login(data.user)
      router.push('/')
    } catch (err) {
      setError(parseError(err))
    } finally {
      setLoading(false)
    }
  }

  const handleGitHubLogin = async () => {
    setGithubLoading(true)
    clearError()

    try {
      const res = await fetch('/api/v1/auth/oauth/url?provider=github')
      const data = await parseAuthResponse<{ authorization_url?: string; url?: string }>(res)
      const redirectURL = data.authorization_url || data.url

      if (!redirectURL) {
        throw new Error('OAuth URL missing in response')
      }

      window.location.href = redirectURL
    } catch (err) {
      setError(parseError(err))
      setGithubLoading(false)
    }
  }

  return (
    <div className="min-h-[80vh] flex items-center justify-center">
      <div className="bg-white dark:bg-gray-800 p-8 rounded-lg shadow-md w-full max-w-md">
        <h1 className="text-2xl font-bold text-center mb-6 text-gray-900 dark:text-gray-100">Login</h1>

        <ErrorAlert error={error} onDismiss={clearError} className="mb-4" />

        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
              Email
            </label>
            <input
              type="email"
              value={email}
              onChange={(e) => {
                setEmail(e.target.value)
                clearFieldError('email')
              }}
              className={cn(
                'w-full border rounded-md px-3 py-2 bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 focus:outline-none focus:ring-2 focus:ring-blue-500',
                hasFieldError('email')
                  ? 'border-red-500 dark:border-red-400'
                  : 'border-gray-300 dark:border-gray-600'
              )}
              required
            />
            <FieldError error={fieldErrors['email']} />
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
              Password
            </label>
            <input
              type="password"
              value={password}
              onChange={(e) => {
                setPassword(e.target.value)
                clearFieldError('password')
              }}
              className={cn(
                'w-full border rounded-md px-3 py-2 bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 focus:outline-none focus:ring-2 focus:ring-blue-500',
                hasFieldError('password')
                  ? 'border-red-500 dark:border-red-400'
                  : 'border-gray-300 dark:border-gray-600'
              )}
              required
            />
            <FieldError error={fieldErrors['password']} />
          </div>

          <button
            type="submit"
            disabled={loading}
            className="w-full bg-blue-600 text-white py-2 rounded-md hover:bg-blue-700 disabled:bg-gray-400 transition-colors"
          >
            {loading ? 'Logging in...' : 'Login'}
          </button>
        </form>

        <div className="mt-4">
          <button
            onClick={handleGitHubLogin}
            disabled={githubLoading}
            className="w-full bg-gray-800 dark:bg-gray-700 text-white py-2 rounded-md hover:bg-gray-900 dark:hover:bg-gray-600 disabled:bg-gray-400 transition-colors flex items-center justify-center gap-2"
          >
            <svg className="w-5 h-5" fill="currentColor" viewBox="0 0 24 24">
              <path d="M12 0c-6.626 0-12 5.373-12 12 0 5.302 3.438 9.8 8.207 11.387.599.111.793-.261.793-.577v-2.234c-3.338.726-4.033-1.416-4.033-1.416-.546-1.387-1.333-1.756-1.333-1.756-1.089-.745.083-.729.083-.729 1.205.084 1.839 1.237 1.839 1.237 1.07 1.834 2.807 1.304 3.492.997.107-.775.418-1.305.762-1.604-2.665-.305-5.467-1.334-5.467-5.931 0-1.311.469-2.381 1.236-3.221-.124-.303-.535-1.524.117-3.176 0 0 1.008-.322 3.301 1.23.957-.266 1.983-.399 3.003-.404 1.02.005 2.047.138 3.006.404 2.291-1.552 3.297-1.23 3.297-1.23.653 1.653.242 2.874.118 3.176.77.84 1.235 1.911 1.235 3.221 0 4.609-2.807 5.624-5.479 5.921.43.372.823 1.102.823 2.222v3.293c0 .319.192.694.801.576 4.765-1.589 8.199-6.086 8.199-11.386 0-6.627-5.373-12-12-12z"/>
            </svg>
            {githubLoading ? 'Redirecting...' : 'Login with GitHub'}
          </button>
        </div>

        <p className="mt-6 text-center text-gray-600 dark:text-gray-400">
          Don&apos;t have an account?{' '}
          <Link href="/register" className="text-blue-600 dark:text-blue-400 hover:underline">
            Register
          </Link>
        </p>
      </div>
    </div>
  )
}