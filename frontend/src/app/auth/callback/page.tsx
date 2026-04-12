'use client'

import { Suspense, useEffect, useState } from 'react'
import { useRouter, useSearchParams } from 'next/navigation'
import { useAuthStore } from '@/stores/authStore'
import { Card, CardContent } from '@/components/ui/card'
import { Button } from '@/components/ui/button'

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
      <Card className="w-full max-w-md">
        <CardContent className="p-8 text-center">
        {status === 'loading' && (
          <div className="flex flex-col items-center gap-4">
            <div className="h-12 w-12 animate-spin rounded-full border-b-2 border-primary"></div>
            <p className="text-muted-foreground">Completing authentication...</p>
          </div>
        )}

        {status === 'success' && !error && (
          <div className="flex flex-col items-center gap-4">
            <div className="text-5xl text-emerald-500">&#10003;</div>
            <p className="font-medium text-foreground">Successfully logged in!</p>
            <p className="text-sm text-muted-foreground">Redirecting...</p>
          </div>
        )}

        {status === 'success' && error && (
          <div className="flex flex-col items-center gap-4">
            <div className="text-5xl text-emerald-500">&#10003;</div>
            <p className="font-medium text-foreground">Authentication successful!</p>
            <p className="text-primary">{error}</p>
            <p className="text-sm text-muted-foreground">Redirecting to login...</p>
          </div>
        )}

        {status === 'error' && (
          <div className="flex flex-col items-center gap-4">
            <div className="text-5xl text-destructive">&#10007;</div>
            <p className="font-medium text-foreground">Authentication failed</p>
            <p className="text-destructive">{error}</p>
            <Button
              onClick={() => router.push('/login')}
              className="mt-4"
            >
              Back to Login
            </Button>
          </div>
        )}
        </CardContent>
      </Card>
    </div>
  )
}

export default function AuthCallbackPage() {
  return (
    <Suspense fallback={
      <div className="min-h-[80vh] flex items-center justify-center">
        <Card className="w-full max-w-md">
          <CardContent className="p-8 text-center">
          <div className="flex flex-col items-center gap-4">
            <div className="h-12 w-12 animate-spin rounded-full border-b-2 border-primary"></div>
            <p className="text-muted-foreground">Loading...</p>
          </div>
          </CardContent>
        </Card>
      </div>
    }>
      <AuthCallbackContent />
    </Suspense>
  )
}