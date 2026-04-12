'use client'

import { useState } from 'react'
import { useRouter } from 'next/navigation'
import Link from 'next/link'
import { useAuthStore } from '@/stores/authStore'
import { useAuthError, parseAuthResponse, createTimeoutController } from '@/hooks/useAuthError'
import { ErrorAlert, FieldError } from '@/components/auth/ErrorAlert'
import { cn } from '@/lib/utils'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'

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
    <div className="flex min-h-[80vh] items-center justify-center">
      <Card className="w-full max-w-md">
        <CardHeader className="space-y-1">
          <CardTitle className="text-center text-2xl">Login</CardTitle>
          <CardDescription className="text-center">
            Sign in to continue to Online Judge.
          </CardDescription>
        </CardHeader>
        <CardContent>

        <ErrorAlert error={error} onDismiss={clearError} className="mb-4" />

        <form onSubmit={handleSubmit} className="space-y-4">
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
            />
            <FieldError error={fieldErrors['password']} />
          </div>

          <Button
            type="submit"
            disabled={loading}
            className="w-full"
          >
            {loading ? 'Logging in...' : 'Login'}
          </Button>
        </form>

        <div className="mt-4">
          <Button
            onClick={handleGitHubLogin}
            disabled={githubLoading}
            variant="secondary"
            className="w-full"
          >
            <svg className="size-4" fill="currentColor" viewBox="0 0 24 24" aria-hidden="true">
              <path d="M12 0C5.373 0 0 5.373 0 12c0 5.303 3.438 9.8 8.207 11.387.6.11.793-.261.793-.577v-2.234c-3.338.726-4.033-1.416-4.033-1.416-.546-1.387-1.333-1.756-1.333-1.756-1.089-.745.083-.729.083-.729 1.205.084 1.839 1.237 1.839 1.237 1.07 1.834 2.807 1.304 3.492.997.108-.775.418-1.305.762-1.604-2.665-.304-5.467-1.334-5.467-5.93 0-1.312.469-2.382 1.236-3.222-.124-.303-.535-1.523.117-3.176 0 0 1.008-.322 3.301 1.23A11.5 11.5 0 0 1 12 5.8a11.5 11.5 0 0 1 3.003.404c2.291-1.552 3.297-1.23 3.297-1.23.653 1.653.242 2.873.119 3.176.77.84 1.235 1.91 1.235 3.222 0 4.607-2.805 5.625-5.476 5.921.43.372.823 1.102.823 2.222v3.293c0 .319.192.688.802.576C20.565 21.796 24 17.299 24 12c0-6.627-5.373-12-12-12Z" />
            </svg>
            {githubLoading ? 'Redirecting...' : 'Login with GitHub'}
          </Button>
        </div>

          <p className="mt-6 text-center text-sm text-muted-foreground">
            Don&apos;t have an account?{' '}
            <Link href="/register" className="font-medium text-primary hover:underline">
              Register
            </Link>
          </p>
        </CardContent>
      </Card>
    </div>
  )
}