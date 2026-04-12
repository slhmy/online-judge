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
  const [contestCount, setContestCount] = useState(0)
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

        await Promise.all([fetchUsers(), fetchContests()])
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

  const fetchContests = async () => {
    try {
      const res = await fetch(`${BFF_URL}/api/v1/contests?page=1&page_size=1`)
      if (!res.ok) return
      const data = await res.json()
      setContestCount(data?.pagination?.total || 0)
    } catch (err) {
      console.error('Failed to fetch contests:', err)
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
        <div className="text-center py-10 text-gray-600 dark:text-gray-400">Loading...</div>
      </div>
    )
  }

  return (
    <div className="px-4 py-6">
      <h1 className="text-2xl font-bold mb-6 text-gray-900 dark:text-gray-100">Admin Dashboard</h1>

      {/* Navigation Cards */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-8">
        <button
          onClick={() => router.push('/admin/problems')}
          className="p-4 bg-blue-600 hover:bg-blue-700 text-white rounded-lg shadow transition-colors"
        >
          <div className="text-lg font-semibold">Problem Management</div>
          <div className="text-sm opacity-80">Create, edit, and delete problems</div>
        </button>
        <button
          onClick={() => router.push('/contests')}
          className="p-4 bg-white dark:bg-gray-800 rounded-lg shadow border border-gray-200 dark:border-gray-700 text-left hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors"
        >
          <div className="text-lg font-semibold text-gray-900 dark:text-gray-100">Contest Overview</div>
          <div className="text-sm text-gray-500 dark:text-gray-400">Total contests: {contestCount}</div>
          <div className="text-sm text-blue-600 dark:text-blue-400 mt-1">Open contest list and scoreboard</div>
        </button>
      </div>

      <div className="bg-white dark:bg-gray-800 rounded-lg shadow">
        <div className="p-4 border-b border-gray-200 dark:border-gray-700">
          <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">User Management</h2>
        </div>

        <div className="overflow-x-auto">
          <table className="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
            <thead className="bg-gray-50 dark:bg-gray-700">
              <tr>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">
                  User
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">
                  Email
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">
                  Role
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">
                  Stats
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">
                  Actions
                </th>
              </tr>
            </thead>
            <tbody className="bg-white dark:bg-gray-800 divide-y divide-gray-200 dark:divide-gray-700">
              {users.map((u) => (
                <tr key={u.id}>
                  <td className="px-6 py-4 whitespace-nowrap">
                    <div className="text-sm font-medium text-gray-900 dark:text-gray-100">{u.username}</div>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap">
                    <div className="text-sm text-gray-500 dark:text-gray-400">{u.email}</div>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap">
                    <span className={`px-2 py-1 text-xs rounded-full ${
                      u.role === 'admin'
                        ? 'bg-purple-100 text-purple-800 dark:bg-purple-900 dark:text-purple-200'
                        : 'bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-200'
                    }`}>
                      {u.role}
                    </span>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-gray-400">
                    <div>Rating: {u.rating}</div>
                    <div>Solved: {u.solved_count} / Submissions: {u.submission_count}</div>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm">
                    {u.role === 'user' ? (
                      <button
                        onClick={() => updateRole(u.id, 'admin')}
                        className="text-blue-600 hover:text-blue-900 dark:text-blue-400"
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