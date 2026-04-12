'use client'

import { useState } from 'react'
import { useRouter } from 'next/navigation'
import Link from 'next/link'
import { useAuthStore } from '@/stores/authStore'
import { useAuthError, parseAuthResponse, createTimeoutController } from '@/hooks/useAuthError'
import { ErrorAlert, FieldError } from '@/components/auth/ErrorAlert'
import { cn } from '@/lib/utils'
import { ErrorCode, ParsedError, ErrorType } from '@/types/auth'

export default function RegisterPage() {
  const router = useRouter()
  const { login } = useAuthStore()

  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [username, setUsername] = useState('')
  const [loading, setLoading] = useState(false)

  const {
    error,
    fieldErrors,
    parseError,
    setError,
    clearError,
    clearFieldError,
    hasFieldError,
  } = useAuthError()

  const createValidationError = (message: string, field?: string): ParsedError => ({
    code: ErrorCode.VALIDATION_ERROR,
    message,
    type: ErrorType.VALIDATION,
    field,
  })

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    clearError()

    if (password !== confirmPassword) {
      setError(createValidationError('密码不匹配', 'confirm_password'))
      return
    }

    if (password.length < 6) {
      setError(createValidationError('密码长度至少6个字符', 'password'))
      return
    }

    setLoading(true)

    const { controller, timeoutId } = createTimeoutController()

    try {
      const res = await fetch('/api/v1/auth/register', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email, password, username }),
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

  return (
    <div className="min-h-[80vh] flex items-center justify-center">
      <div className="bg-white dark:bg-gray-800 p-8 rounded-lg shadow-md w-full max-w-md">
        <h1 className="text-2xl font-bold text-center mb-6 text-gray-900 dark:text-gray-100">Register</h1>

        <ErrorAlert error={error} onDismiss={clearError} className="mb-4" />

        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
              Username
            </label>
            <input
              type="text"
              value={username}
              onChange={(e) => {
                setUsername(e.target.value)
                clearFieldError('username')
              }}
              className={cn(
                'w-full border rounded-md px-3 py-2 bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 focus:outline-none focus:ring-2 focus:ring-blue-500',
                hasFieldError('username')
                  ? 'border-red-500 dark:border-red-400'
                  : 'border-gray-300 dark:border-gray-600'
              )}
              placeholder="Optional"
            />
            <FieldError error={fieldErrors['username']} />
          </div>

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
              minLength={6}
            />
            <FieldError error={fieldErrors['password']} />
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
              Confirm Password
            </label>
            <input
              type="password"
              value={confirmPassword}
              onChange={(e) => {
                setConfirmPassword(e.target.value)
                clearFieldError('confirm_password')
              }}
              className={cn(
                'w-full border rounded-md px-3 py-2 bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 focus:outline-none focus:ring-2 focus:ring-blue-500',
                hasFieldError('confirm_password')
                  ? 'border-red-500 dark:border-red-400'
                  : 'border-gray-300 dark:border-gray-600'
              )}
              required
            />
            <FieldError error={fieldErrors['confirm_password']} />
          </div>

          <button
            type="submit"
            disabled={loading}
            className="w-full bg-blue-600 text-white py-2 rounded-md hover:bg-blue-700 disabled:bg-gray-400 transition-colors"
          >
            {loading ? 'Creating account...' : 'Register'}
          </button>
        </form>

        <p className="mt-6 text-center text-gray-600 dark:text-gray-400">
          Already have an account?{' '}
          <Link href="/login" className="text-blue-600 dark:text-blue-400 hover:underline">
            Login
          </Link>
        </p>
      </div>
    </div>
  )
}