'use client'

import { useEffect, useState, useCallback, useRef } from 'react'
import { useParams, useRouter } from 'next/navigation'
import { useAuthStore } from '@/stores/authStore'
import { useTestCases, useCreateTestCase, useUpdateTestCase, useDeleteTestCase, useToggleTestCaseSample, useBatchUploadTestCases } from '@/hooks/useApi'

const BFF_URL = process.env.NEXT_PUBLIC_BFF_URL || ''

interface TestCase {
  id: string
  problem_id: string
  rank: number
  is_sample: boolean
  input_path: string
  output_path: string
  input_content: string
  output_content: string
  description: string
  is_interactive: boolean
}

interface TestCaseResponse {
  test_cases: TestCase[]
}

export default function AdminTestCasesPage() {
  const params = useParams()
  const router = useRouter()
  const problemId = params.id as string
  const { user, isAuthenticated } = useAuthStore()
  const [hydrated, setHydrated] = useState(false)

  const [showUploadModal, setShowUploadModal] = useState(false)
  const [showEditModal, setShowEditModal] = useState(false)
  const [showPreviewModal, setShowPreviewModal] = useState(false)
  const [editingTestCase, setEditingTestCase] = useState<TestCase | null>(null)
  const [previewTestCase, setPreviewTestCase] = useState<TestCase | null>(null)
  const [previewType, setPreviewType] = useState<'input' | 'output'>('input')
  const [deleteConfirmId, setDeleteConfirmId] = useState<string | null>(null)

  // Batch upload state
  const [uploadMode, setUploadMode] = useState<'zip' | 'files'>('zip')
  const [zipFile, setZipFile] = useState<File | null>(null)
  const [inputFiles, setInputFiles] = useState<File[]>([])
  const [outputFiles, setOutputFiles] = useState<File[]>([])
  const [defaultIsSample, setDefaultIsSample] = useState(false)
  const [uploading, setUploading] = useState(false)

  // Single test case create state
  const [newRank, setNewRank] = useState(1)
  const [newIsSample, setNewIsSample] = useState(false)
  const [newInputContent, setNewInputContent] = useState('')
  const [newOutputContent, setNewOutputContent] = useState('')
  const [newDescription, setNewDescription] = useState('')
  const [showCreateForm, setShowCreateForm] = useState(false)

  const { data, isLoading, error } = useTestCases(problemId) as {
    data: TestCaseResponse | undefined
    isLoading: boolean
    error: Error | null
  }
  const testCases: TestCase[] = data?.test_cases || []

  const createMutation = useCreateTestCase()
  const updateMutation = useUpdateTestCase()
  const deleteMutation = useDeleteTestCase()
  const toggleSampleMutation = useToggleTestCaseSample()
  const batchUploadMutation = useBatchUploadTestCases()

  const fileInputRef = useRef<HTMLInputElement>(null)
  const inputFileInputRef = useRef<HTMLInputElement>(null)
  const outputFileInputRef = useRef<HTMLInputElement>(null)

  useEffect(() => {
    setHydrated(useAuthStore.persist.hasHydrated())
    const unsubFinish = useAuthStore.persist.onFinishHydration(() => setHydrated(true))
    return () => {
      unsubFinish()
    }
  }, [])

  useEffect(() => {
    // Check if user is admin
    if (!hydrated) {
      return
    }

    if (!isAuthenticated || user?.role !== 'admin') {
      router.push('/')
      return
    }
  }, [hydrated, isAuthenticated, user, router])

  // Auto-set new rank based on existing test cases
  useEffect(() => {
    if (testCases.length > 0) {
      const maxRank = Math.max(...testCases.map(tc => tc.rank))
      setNewRank(maxRank + 1)
    }
  }, [testCases])

  const handleToggleSample = async (testCase: TestCase) => {
    try {
      await toggleSampleMutation.mutateAsync(testCase.id)
    } catch (err) {
      console.error('Failed to toggle sample:', err)
    }
  }

  const handleDelete = async (id: string) => {
    try {
      await deleteMutation.mutateAsync(id)
      setDeleteConfirmId(null)
    } catch (err) {
      console.error('Failed to delete test case:', err)
    }
  }

  const handleCreate = async () => {
    if (!newInputContent.trim() || !newOutputContent.trim()) {
      alert('Input and output content are required')
      return
    }

    try {
      await createMutation.mutateAsync({
        problemId,
        rank: newRank,
        is_sample: newIsSample,
        input_content: newInputContent,
        output_content: newOutputContent,
        description: newDescription,
      })
      setShowCreateForm(false)
      setNewInputContent('')
      setNewOutputContent('')
      setNewDescription('')
      setNewIsSample(false)
    } catch (err) {
      console.error('Failed to create test case:', err)
    }
  }

  const handleUpdate = async () => {
    if (!editingTestCase) return

    try {
      await updateMutation.mutateAsync({
        testCaseId: editingTestCase.id,
        rank: editingTestCase.rank,
        is_sample: editingTestCase.is_sample,
        description: editingTestCase.description,
      })
      setShowEditModal(false)
      setEditingTestCase(null)
    } catch (err) {
      console.error('Failed to update test case:', err)
    }
  }

  const handleBatchUpload = async () => {
    if (uploadMode === 'zip' && !zipFile) {
      alert('Please select a ZIP file')
      return
    }
    if (uploadMode === 'files' && inputFiles.length === 0) {
      alert('Please select input files')
      return
    }

    setUploading(true)
    try {
      const formData = new FormData()
      if (uploadMode === 'zip') {
        formData.append('zip_file', zipFile!)
      } else {
        for (const file of inputFiles) {
          formData.append('input_files', file)
        }
        for (const file of outputFiles) {
          formData.append('output_files', file)
        }
      }
      formData.append('is_sample', defaultIsSample.toString())

      await batchUploadMutation.mutateAsync({
        problemId,
        formData,
      })

      setShowUploadModal(false)
      setZipFile(null)
      setInputFiles([])
      setOutputFiles([])
      if (fileInputRef.current) fileInputRef.current.value = ''
      if (inputFileInputRef.current) inputFileInputRef.current.value = ''
      if (outputFileInputRef.current) outputFileInputRef.current.value = ''
    } catch (err) {
      console.error('Failed to batch upload:', err)
      alert('Failed to upload test cases')
    } finally {
      setUploading(false)
    }
  }

  const openPreview = (testCase: TestCase, type: 'input' | 'output') => {
    setPreviewTestCase(testCase)
    setPreviewType(type)
    setShowPreviewModal(true)
  }

  const truncateContent = (content: string, maxLength: number = 100) => {
    if (!content) return '(no content)'
    if (content.length <= maxLength) return content
    return content.substring(0, maxLength) + '...'
  }

  if (!hydrated || !isAuthenticated || user?.role !== 'admin') {
    return null
  }

  return (
    <div className="px-4 py-6">
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-bold text-gray-900 dark:text-gray-100">Admin: Test Case Management</h1>
        <div className="flex gap-2">
          <button
            onClick={() => setShowCreateForm(!showCreateForm)}
            className="px-4 py-2 bg-green-600 hover:bg-green-700 text-white rounded-lg font-medium transition-colors"
          >
            {showCreateForm ? 'Cancel' : 'Add Test Case'}
          </button>
          <button
            onClick={() => setShowUploadModal(true)}
            className="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg font-medium transition-colors"
          >
            Batch Upload
          </button>
          <button
            onClick={() => router.push(`/admin/problems`)}
            className="px-4 py-2 bg-gray-200 hover:bg-gray-300 dark:bg-gray-700 dark:hover:bg-gray-600 text-gray-700 dark:text-gray-200 rounded-lg font-medium transition-colors"
          >
            Back to Problems
          </button>
        </div>
      </div>

      {/* Single Test Case Create Form */}
      {showCreateForm && (
        <div className="bg-white dark:bg-gray-800 rounded-lg shadow p-6 mb-8">
          <h2 className="text-lg font-semibold mb-4 text-gray-900 dark:text-gray-100">Create New Test Case</h2>

          <div className="space-y-4">
            <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Rank</label>
                <input
                  type="number"
                  value={newRank}
                  onChange={(e) => setNewRank(parseInt(e.target.value) || 1)}
                  className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100"
                  min="1"
                />
              </div>
              <div className="flex items-center">
                <label className="flex items-center gap-2 text-sm font-medium text-gray-700 dark:text-gray-300">
                  <input
                    type="checkbox"
                    checked={newIsSample}
                    onChange={(e) => setNewIsSample(e.target.checked)}
                    className="rounded border-gray-300 dark:border-gray-600"
                  />
                  Sample Test Case (visible to users)
                </label>
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Description (optional)</label>
                <input
                  type="text"
                  value={newDescription}
                  onChange={(e) => setNewDescription(e.target.value)}
                  className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100"
                  placeholder="e.g., 'Edge case with empty input'"
                />
              </div>
            </div>

            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Input Content</label>
                <textarea
                  value={newInputContent}
                  onChange={(e) => setNewInputContent(e.target.value)}
                  className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 font-mono"
                  rows={6}
                  placeholder="Enter test input..."
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Output Content</label>
                <textarea
                  value={newOutputContent}
                  onChange={(e) => setNewOutputContent(e.target.value)}
                  className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 font-mono"
                  rows={6}
                  placeholder="Enter expected output..."
                />
              </div>
            </div>

            <div className="flex gap-3">
              <button
                onClick={handleCreate}
                disabled={createMutation.isPending}
                className="px-4 py-2 bg-green-600 hover:bg-green-700 disabled:opacity-50 text-white rounded-lg font-medium transition-colors"
              >
                {createMutation.isPending ? 'Creating...' : 'Create Test Case'}
              </button>
              <button
                onClick={() => setShowCreateForm(false)}
                className="px-4 py-2 bg-gray-200 hover:bg-gray-300 dark:bg-gray-700 dark:hover:bg-gray-600 text-gray-700 dark:text-gray-200 rounded-lg font-medium transition-colors"
              >
                Cancel
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Test Case List */}
      {error && (
        <div className="text-center py-10 text-red-400">
          Error loading test cases: {error.message}
        </div>
      )}

      {isLoading ? (
        <div className="text-center py-10 text-gray-600 dark:text-gray-400">Loading...</div>
      ) : testCases.length === 0 ? (
        <div className="text-center py-10 text-gray-500 dark:text-gray-400">
          No test cases. Add one above or use batch upload.
        </div>
      ) : (
        <div className="overflow-x-auto">
          <table className="min-w-full bg-white dark:bg-gray-800 rounded-lg shadow">
            <thead className="bg-gray-100 dark:bg-gray-700">
              <tr>
                <th className="px-4 py-3 text-left text-sm font-medium text-gray-700 dark:text-gray-300">Rank</th>
                <th className="px-4 py-3 text-left text-sm font-medium text-gray-700 dark:text-gray-300">Sample</th>
                <th className="px-4 py-3 text-left text-sm font-medium text-gray-700 dark:text-gray-300">Input Preview</th>
                <th className="px-4 py-3 text-left text-sm font-medium text-gray-700 dark:text-gray-300">Output Preview</th>
                <th className="px-4 py-3 text-left text-sm font-medium text-gray-700 dark:text-gray-300">Description</th>
                <th className="px-4 py-3 text-left text-sm font-medium text-gray-700 dark:text-gray-300">Actions</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-200 dark:divide-gray-700">
              {testCases.map((tc) => (
                <tr key={tc.id} className="hover:bg-gray-50 dark:hover:bg-gray-700/50">
                  <td className="px-4 py-3 text-sm font-medium text-gray-900 dark:text-gray-100">
                    {tc.rank}
                  </td>
                  <td className="px-4 py-3 text-sm">
                    <button
                      onClick={() => handleToggleSample(tc)}
                      disabled={toggleSampleMutation.isPending}
                      className={`px-3 py-1 rounded-full text-xs font-medium transition-colors ${
                        tc.is_sample
                          ? 'bg-green-100 dark:bg-green-900/50 text-green-700 dark:text-green-400 hover:bg-green-200'
                          : 'bg-gray-100 dark:bg-gray-700 text-gray-600 dark:text-gray-400 hover:bg-gray-200'
                      }`}
                    >
                      {tc.is_sample ? 'Sample' : 'Hidden'}
                    </button>
                  </td>
                  <td className="px-4 py-3 text-sm text-gray-700 dark:text-gray-300 font-mono">
                    <div className="flex items-center gap-2">
                      <span className="truncate max-w-[200px]">{truncateContent(tc.input_content)}</span>
                      <button
                        onClick={() => openPreview(tc, 'input')}
                        className="text-blue-600 hover:text-blue-800 dark:text-blue-400 dark:hover:text-blue-300 text-xs"
                      >
                        View
                      </button>
                    </div>
                  </td>
                  <td className="px-4 py-3 text-sm text-gray-700 dark:text-gray-300 font-mono">
                    <div className="flex items-center gap-2">
                      <span className="truncate max-w-[200px]">{truncateContent(tc.output_content)}</span>
                      <button
                        onClick={() => openPreview(tc, 'output')}
                        className="text-blue-600 hover:text-blue-800 dark:text-blue-400 dark:hover:text-blue-300 text-xs"
                      >
                        View
                      </button>
                    </div>
                  </td>
                  <td className="px-4 py-3 text-sm text-gray-700 dark:text-gray-300">
                    {tc.description || '-'}
                  </td>
                  <td className="px-4 py-3 text-sm">
                    <div className="flex gap-2">
                      <button
                        onClick={() => {
                          setEditingTestCase(tc)
                          setShowEditModal(true)
                        }}
                        className="text-blue-600 hover:text-blue-800 dark:text-blue-400 dark:hover:text-blue-300 font-medium"
                      >
                        Edit
                      </button>
                      {deleteConfirmId === tc.id ? (
                        <div className="flex gap-1">
                          <button
                            onClick={() => handleDelete(tc.id)}
                            className="text-red-600 hover:text-red-800 dark:text-red-400 dark:hover:text-red-300 font-medium"
                          >
                            Confirm
                          </button>
                          <button
                            onClick={() => setDeleteConfirmId(null)}
                            className="text-gray-600 hover:text-gray-800 dark:text-gray-400 dark:hover:text-gray-300 font-medium"
                          >
                            Cancel
                          </button>
                        </div>
                      ) : (
                        <button
                          onClick={() => setDeleteConfirmId(tc.id)}
                          className="text-red-600 hover:text-red-800 dark:text-red-400 dark:hover:text-red-300 font-medium"
                        >
                          Delete
                        </button>
                      )}
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {/* Batch Upload Modal */}
      {showUploadModal && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-white dark:bg-gray-800 rounded-lg shadow-lg p-6 max-w-lg w-full mx-4">
            <h2 className="text-lg font-semibold mb-4 text-gray-900 dark:text-gray-100">Batch Upload Test Cases</h2>

            <div className="space-y-4">
              {/* Upload Mode Selection */}
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">Upload Mode</label>
                <div className="flex gap-4">
                  <button
                    onClick={() => setUploadMode('zip')}
                    className={`px-4 py-2 rounded-lg font-medium transition-colors ${
                      uploadMode === 'zip'
                        ? 'bg-blue-600 text-white'
                        : 'bg-gray-200 dark:bg-gray-700 text-gray-700 dark:text-gray-200'
                    }`}
                  >
                    ZIP File
                  </button>
                  <button
                    onClick={() => setUploadMode('files')}
                    className={`px-4 py-2 rounded-lg font-medium transition-colors ${
                      uploadMode === 'files'
                        ? 'bg-blue-600 text-white'
                        : 'bg-gray-200 dark:bg-gray-700 text-gray-700 dark:text-gray-200'
                    }`}
                  >
                    Separate Files
                  </button>
                </div>
              </div>

              {/* ZIP Upload */}
              {uploadMode === 'zip' && (
                <div>
                  <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                    ZIP File (contains numbered pairs like 1.in, 1.out)
                  </label>
                  <input
                    ref={fileInputRef}
                    type="file"
                    accept=".zip"
                    onChange={(e) => setZipFile(e.target.files?.[0] || null)}
                    className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100"
                  />
                  <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">
                    ZIP should contain files named like: 1.in, 1.out, 2.in, 2.out, etc.
                  </p>
                </div>
              )}

              {/* Separate Files Upload */}
              {uploadMode === 'files' && (
                <div className="space-y-3">
                  <div>
                    <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                      Input Files (named like 1.in, 2.in, etc.)
                    </label>
                    <input
                      ref={inputFileInputRef}
                      type="file"
                      multiple
                      accept=".in,.txt"
                      onChange={(e) => setInputFiles(Array.from(e.target.files || []))}
                      className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100"
                    />
                    {inputFiles.length > 0 && (
                      <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">
                        Selected: {inputFiles.map(f => f.name).join(', ')}
                      </p>
                    )}
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                      Output Files (named like 1.out, 2.out, etc.)
                    </label>
                    <input
                      ref={outputFileInputRef}
                      type="file"
                      multiple
                      accept=".out,.txt"
                      onChange={(e) => setOutputFiles(Array.from(e.target.files || []))}
                      className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100"
                    />
                    {outputFiles.length > 0 && (
                      <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">
                        Selected: {outputFiles.map(f => f.name).join(', ')}
                      </p>
                    )}
                  </div>
                </div>
              )}

              {/* Default is_sample */}
              <div className="flex items-center">
                <label className="flex items-center gap-2 text-sm font-medium text-gray-700 dark:text-gray-300">
                  <input
                    type="checkbox"
                    checked={defaultIsSample}
                    onChange={(e) => setDefaultIsSample(e.target.checked)}
                    className="rounded border-gray-300 dark:border-gray-600"
                  />
                  Mark all as sample test cases (visible to users)
                </label>
              </div>

              {/* Actions */}
              <div className="flex gap-3 justify-end">
                <button
                  onClick={() => setShowUploadModal(false)}
                  className="px-4 py-2 bg-gray-200 hover:bg-gray-300 dark:bg-gray-700 dark:hover:bg-gray-600 text-gray-700 dark:text-gray-200 rounded-lg font-medium transition-colors"
                >
                  Cancel
                </button>
                <button
                  onClick={handleBatchUpload}
                  disabled={uploading || batchUploadMutation.isPending}
                  className="px-4 py-2 bg-blue-600 hover:bg-blue-700 disabled:opacity-50 text-white rounded-lg font-medium transition-colors"
                >
                  {uploading || batchUploadMutation.isPending ? 'Uploading...' : 'Upload'}
                </button>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Edit Modal */}
      {showEditModal && editingTestCase && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-white dark:bg-gray-800 rounded-lg shadow-lg p-6 max-w-lg w-full mx-4">
            <h2 className="text-lg font-semibold mb-4 text-gray-900 dark:text-gray-100">Edit Test Case #{editingTestCase.rank}</h2>

            <div className="space-y-4">
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <div>
                  <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Rank</label>
                  <input
                    type="number"
                    value={editingTestCase.rank}
                    onChange={(e) => setEditingTestCase({ ...editingTestCase, rank: parseInt(e.target.value) || 1 })}
                    className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100"
                    min="1"
                  />
                </div>
                <div className="flex items-center">
                  <label className="flex items-center gap-2 text-sm font-medium text-gray-700 dark:text-gray-300">
                    <input
                      type="checkbox"
                      checked={editingTestCase.is_sample}
                      onChange={(e) => setEditingTestCase({ ...editingTestCase, is_sample: e.target.checked })}
                      className="rounded border-gray-300 dark:border-gray-600"
                    />
                    Sample Test Case
                  </label>
                </div>
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Description</label>
                <input
                  type="text"
                  value={editingTestCase.description || ''}
                  onChange={(e) => setEditingTestCase({ ...editingTestCase, description: e.target.value })}
                  className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100"
                />
              </div>

              <div className="flex gap-3 justify-end">
                <button
                  onClick={() => {
                    setShowEditModal(false)
                    setEditingTestCase(null)
                  }}
                  className="px-4 py-2 bg-gray-200 hover:bg-gray-300 dark:bg-gray-700 dark:hover:bg-gray-600 text-gray-700 dark:text-gray-200 rounded-lg font-medium transition-colors"
                >
                  Cancel
                </button>
                <button
                  onClick={handleUpdate}
                  disabled={updateMutation.isPending}
                  className="px-4 py-2 bg-blue-600 hover:bg-blue-700 disabled:opacity-50 text-white rounded-lg font-medium transition-colors"
                >
                  {updateMutation.isPending ? 'Saving...' : 'Save'}
                </button>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Preview Modal */}
      {showPreviewModal && previewTestCase && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-white dark:bg-gray-800 rounded-lg shadow-lg p-6 max-w-2xl w-full mx-4">
            <div className="flex items-center justify-between mb-4">
              <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">
                Test Case #{previewTestCase.rank} - {previewType === 'input' ? 'Input' : 'Output'}
              </h2>
              <div className="flex gap-2">
                <button
                  onClick={() => setPreviewType('input')}
                  className={`px-3 py-1 rounded-lg font-medium transition-colors ${
                    previewType === 'input'
                      ? 'bg-blue-600 text-white'
                      : 'bg-gray-200 dark:bg-gray-700 text-gray-700 dark:text-gray-200'
                  }`}
                >
                  Input
                </button>
                <button
                  onClick={() => setPreviewType('output')}
                  className={`px-3 py-1 rounded-lg font-medium transition-colors ${
                    previewType === 'output'
                      ? 'bg-blue-600 text-white'
                      : 'bg-gray-200 dark:bg-gray-700 text-gray-700 dark:text-gray-200'
                  }`}
                >
                  Output
                </button>
              </div>
            </div>

            <div className="bg-gray-100 dark:bg-gray-900 rounded-lg p-4 max-h-[400px] overflow-auto">
              <pre className="font-mono text-sm text-gray-800 dark:text-gray-200 whitespace-pre-wrap">
                {previewType === 'input' ? previewTestCase.input_content : previewTestCase.output_content}
              </pre>
            </div>

            <div className="flex justify-end mt-4">
              <button
                onClick={() => {
                  setShowPreviewModal(false)
                  setPreviewTestCase(null)
                }}
                className="px-4 py-2 bg-gray-200 hover:bg-gray-300 dark:bg-gray-700 dark:hover:bg-gray-600 text-gray-700 dark:text-gray-200 rounded-lg font-medium transition-colors"
              >
                Close
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}