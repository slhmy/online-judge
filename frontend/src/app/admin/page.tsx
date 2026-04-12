'use client'

import { useEffect, useRef, useState } from 'react'
import { useRouter } from 'next/navigation'
import { useAuthStore } from '@/stores/authStore'

const BFF_URL = process.env.NEXT_PUBLIC_BFF_URL || ''

interface User {
  id: string
  username: string
  email: string
  role: string
  rating: number
  solved_count: number
  submission_count: number
  created_at: string
}

export default function AdminPage() {
  const router = useRouter()
  const { logout } = useAuthStore()
  const [hydrated, setHydrated] = useState(false)
  const [users, setUsers] = useState<User[]>([])
  const [loading, setLoading] = useState(true)
  const initializedRef = useRef(false)

  useEffect(() => {
    setHydrated(useAuthStore.persist.hasHydrated())
    const unsubFinish = useAuthStore.persist.onFinishHydration(() => setHydrated(true))
    return () => {
      unsubFinish()
    }
  }, [])

  useEffect(() => {
    let active = true

    const init = async () => {
      if (!hydrated || initializedRef.current) {
        return
      }
      initializedRef.current = true

      const { isAuthenticated, user } = useAuthStore.getState()

      if (!isAuthenticated || !user) {
        router.replace('/login')
        if (active) {
          setLoading(false)
        }
        return
      }

      try {
        const meRes = await fetch(`${BFF_URL}/api/v1/auth/me`, {
          credentials: 'include',
        })

        if (!meRes.ok) {
          logout()
          router.replace('/login')
          return
        }

        const meData = await meRes.json()
        const currentUser = (meData?.user ?? meData) as Partial<User>
        const currentRole = String(currentUser?.role || '').toLowerCase()

        if (!currentUser || currentRole !== 'admin') {
          router.replace('/')
          return
        }

        if (!currentUser.id || !currentUser.email) {
          logout()
          router.replace('/login')
          return
        }

        await fetchUsers()
      } catch (err) {
        console.error('Failed to initialize admin page:', err)
      } finally {
        if (active) {
          setLoading(false)
        }
      }
    }

    init()

    return () => {
      active = false
    }
  }, [hydrated, router, logout])

  const fetchUsers = async () => {
    try {
      const res = await fetch(`${BFF_URL}/api/v1/admin/users`, {
        credentials: 'include',
      })
      if (res.status === 401) {
        logout()
        router.replace('/login')
        return
      }
      if (res.status === 403) {
        router.replace('/')
        return
      }
      if (res.ok) {
        const data = await res.json()
        setUsers(data.users || [])
      }
    } catch (err) {
      console.error('Failed to fetch users:', err)
    }
  }

  const updateRole = async (userId: string, newRole: string) => {
    try {
      const res = await fetch(`${BFF_URL}/api/v1/admin/users/${userId}/role`, {
        method: 'PUT',
        credentials: 'include',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ role: newRole }),
      })
      if (res.status === 401) {
        logout()
        router.replace('/login')
        return
      }
      if (res.status === 403) {
        router.replace('/')
        return
      }
      if (res.ok) {
        fetchUsers()
      }
    } catch (err) {
      console.error('Failed to update role:', err)
    }
  }

  if (loading) {
    return (
      <div className="px-4 py-6">
        <div className="text-center py-10 text-muted-foreground">Loading...</div>
      </div>
    )
  }

  return (
    <div className="px-4 py-6">
      <h1 className="text-2xl font-bold mb-6 text-foreground">Admin Dashboard</h1>

      <div className="bg-card rounded-xl shadow">
        <div className="p-4 border-b border-border">
          <h2 className="text-lg font-semibold text-foreground">User Management</h2>
        </div>

        <div className="overflow-x-auto">
          <table className="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
            <thead className="bg-muted/40">
              <tr>
                <th className="px-6 py-3 text-left text-xs font-medium text-muted-foreground uppercase tracking-wider">
                  User
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-muted-foreground uppercase tracking-wider">
                  Email
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-muted-foreground uppercase tracking-wider">
                  Role
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-muted-foreground uppercase tracking-wider">
                  Stats
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-muted-foreground uppercase tracking-wider">
                  Actions
                </th>
              </tr>
            </thead>
            <tbody className="bg-card divide-y divide-gray-200 dark:divide-gray-700">
              {users.map((u) => (
                <tr key={u.id}>
                  <td className="px-6 py-4 whitespace-nowrap">
                    <div className="text-sm font-medium text-foreground">{u.username}</div>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap">
                    <div className="text-sm text-muted-foreground">{u.email || '-'}</div>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap">
                    <span className={`px-2 py-1 text-xs rounded-full ${
                      u.role === 'admin'
                        ? 'bg-purple-100 text-purple-800 dark:bg-purple-900 dark:text-purple-200'
                        : 'bg-muted text-foreground'
                    }`}>
                      {u.role}
                    </span>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-muted-foreground">
                    <div>Rating: {u.rating}</div>
                    <div>Solved: {u.solved_count} / Submissions: {u.submission_count}</div>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm">
                    {u.role === 'user' ? (
                      <button
                        onClick={() => updateRole(u.id, 'admin')}
                        className="text-primary hover:text-primary "
                      >
                        Make Admin
                      </button>
                    ) : (
                      <button
                        onClick={() => updateRole(u.id, 'user')}
                        className="text-red-600 hover:text-red-900 dark:text-red-400"
                      >
                        Remove Admin
                      </button>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  )
}