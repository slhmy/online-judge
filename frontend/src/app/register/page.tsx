'use client'

import { useState } from 'react'
import { useRouter } from 'next/navigation'
import Link from 'next/link'
import { useAuthStore } from '@/stores/authStore'
import { useAuthError, parseAuthResponse, createTimeoutController } from '@/hooks/useAuthError'
import { ErrorAlert, FieldError } from '@/components/auth/ErrorAlert'
import { cn } from '@/lib/utils'
import { ErrorCode, ParsedError, ErrorType } from '@/types/auth'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'

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
    <div className="flex min-h-[80vh] items-center justify-center">
      <Card className="w-full max-w-md">
        <CardHeader className="space-y-1">
          <CardTitle className="text-center text-2xl">Register</CardTitle>
          <CardDescription className="text-center">
            Create your account to start submitting solutions.
          </CardDescription>
        </CardHeader>
        <CardContent>

        <ErrorAlert error={error} onDismiss={clearError} className="mb-4" />

        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label className="mb-1 block text-sm font-medium text-foreground">
              Username
            </label>
            <Input
              type="text"
              value={username}
              onChange={(e) => {
                setUsername(e.target.value)
                clearFieldError('username')
              }}
              className={cn(
                hasFieldError('username') && 'border-destructive ring-destructive/20'
              )}
              placeholder="Optional"
            />
            <FieldError error={fieldErrors['username']} />
          </div>

          <div>
            <label className="mb-1 block text-sm font-medium text-foreground">
              Email
            </label>
            <Input
              type="email"
              value={email}
              onChange={(e) => {
                setEmail(e.target.value)
                clearFieldError('email')
              }}
              className={cn(
                hasFieldError('email') && 'border-destructive ring-destructive/20'
              )}
              required
            />
            <FieldError error={fieldErrors['email']} />
          </div>

          <div>
            <label className="mb-1 block text-sm font-medium text-foreground">
              Password
            </label>
            <Input
              type="password"
              value={password}
              onChange={(e) => {
                setPassword(e.target.value)
                clearFieldError('password')
              }}
              className={cn(
                hasFieldError('password') && 'border-destructive ring-destructive/20'
              )}
              required
              minLength={6}
            />
            <FieldError error={fieldErrors['password']} />
          </div>

          <div>
            <label className="mb-1 block text-sm font-medium text-foreground">
              Confirm Password
            </label>
            <Input
              type="password"
              value={confirmPassword}
              onChange={(e) => {
                setConfirmPassword(e.target.value)
                clearFieldError('confirm_password')
              }}
              className={cn(
                hasFieldError('confirm_password') && 'border-destructive ring-destructive/20'
              )}
              required
            />
            <FieldError error={fieldErrors['confirm_password']} />
          </div>

          <Button
            type="submit"
            disabled={loading}
            className="w-full"
          >
            {loading ? 'Creating account...' : 'Register'}
          </Button>
        </form>

          <p className="mt-6 text-center text-sm text-muted-foreground">
          Already have an account?{' '}
            <Link href="/login" className="font-medium text-primary hover:underline">
            Login
          </Link>
        </p>
        </CardContent>
      </Card>
    </div>
  )
}