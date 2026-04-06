'use client'

import { useEffect, useState } from 'react'
import { useRouter } from 'next/navigation'
import { useAuthStore } from '@/stores/authStore'

const BFF_URL = process.env.NEXT_PUBLIC_BFF_URL || 'http://localhost:8080'

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
  const { user, isAuthenticated, token } = useAuthStore()
  const [users, setUsers] = useState<User[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    // Check if user is admin
    if (!isAuthenticated || user?.role !== 'admin') {
      router.push('/')
      return
    }

    // Fetch users
    fetchUsers()
  }, [isAuthenticated, user, router])

  const fetchUsers = async () => {
    try {
      const res = await fetch(`${BFF_URL}/api/v1/admin/users`, {
        headers: {
          Authorization: `Bearer ${token}`,
        },
      })
      if (res.ok) {
        const data = await res.json()
        setUsers(data.users || [])
      }
    } catch (err) {
      console.error('Failed to fetch users:', err)
    } finally {
      setLoading(false)
    }
  }

  const updateRole = async (userId: string, newRole: string) => {
    try {
      const res = await fetch(`${BFF_URL}/api/v1/admin/users/${userId}/role`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${token}`,
        },
        body: JSON.stringify({ role: newRole }),
      })
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