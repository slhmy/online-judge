'use client'

import { useState, useCallback } from 'react'
import {
  ErrorCode,
  ErrorType,
  AuthErrorResponse,
  ParsedError,
  authErrorMessages,
  errorCodeToType,
  fieldTranslations,
} from '@/types/auth'

export interface UseAuthErrorReturn {
  error: ParsedError | null
  fieldErrors: Record<string, ParsedError>
  setError: (error: ParsedError | null) => void
  parseError: (error: unknown) => ParsedError
  clearError: () => void
  clearFieldError: (field: string) => void
  hasError: boolean
  hasFieldError: (field: string) => boolean
}

/**
 * Hook for handling authentication errors with structured parsing
 */
export function useAuthError(): UseAuthErrorReturn {
  const [error, setError] = useState<ParsedError | null>(null)
  const [fieldErrors, setFieldErrors] = useState<Record<string, ParsedError>>({})

  /**
   * Parse an unknown error into a structured ParsedError
   */
  const parseError = useCallback((err: unknown): ParsedError => {
    // Handle network/fetch errors (TypeError from fetch failure)
    if (err instanceof TypeError) {
      return {
        code: ErrorCode.NETWORK_ERROR,
        message: authErrorMessages[ErrorCode.NETWORK_ERROR],
        type: ErrorType.NETWORK,
      }
    }

    // Handle AbortError (timeout or cancelled request)
    if (err instanceof Error && err.name === 'AbortError') {
      return {
        code: ErrorCode.TIMEOUT_ERROR,
        message: authErrorMessages[ErrorCode.TIMEOUT_ERROR],
        type: ErrorType.NETWORK,
      }
    }

    // Handle structured API error response
    if (err instanceof Error) {
      // Try to parse JSON error from API response
      try {
        // Check if the error message contains JSON (from fetch response parsing)
        const errorObj = JSON.parse(err.message.replace('BFF error: ', '').replace('API error: ', ''))
        if (errorObj.error_code) {
          const code = errorObj.error_code as ErrorCode
          const field = errorObj.field
          const message = errorObj.message || authErrorMessages[code] || '未知错误'
          const type = errorCodeToType[code] || ErrorType.SYSTEM

          return {
            code,
            message,
            type,
            field,
            fieldLabel: field ? fieldTranslations[field] : undefined,
          }
        }
      } catch {
        // Not JSON, continue with generic error handling
      }

      // Generic error message
      return {
        code: ErrorCode.INTERNAL_ERROR,
        message: err.message || authErrorMessages[ErrorCode.INTERNAL_ERROR],
        type: ErrorType.SYSTEM,
      }
    }

    // Unknown error type
    return {
      code: ErrorCode.INTERNAL_ERROR,
      message: authErrorMessages[ErrorCode.INTERNAL_ERROR],
      type: ErrorType.SYSTEM,
    }
  }, [])

  /**
   * Set error and update field errors if applicable
   */
  const handleError = useCallback((parsedError: ParsedError | null) => {
    setError(parsedError)

    if (parsedError?.field) {
      setFieldErrors((prev) => ({
        ...prev,
        [parsedError.field!]: parsedError,
      }))
    }
  }, [])

  /**
   * Clear all errors
   */
  const clearError = useCallback(() => {
    setError(null)
    setFieldErrors({})
  }, [])

  /**
   * Clear error for a specific field
   */
  const clearFieldError = useCallback((field: string) => {
    setFieldErrors((prev) => {
      const newErrors = { ...prev }
      delete newErrors[field]
      return newErrors
    })

    // Clear global error if it matches the field
    if (error?.field === field) {
      setError(null)
    }
  }, [error])

  /**
   * Check if there's a global error
   */
  const hasError = error !== null

  /**
   * Check if there's an error for a specific field
   */
  const hasFieldError = useCallback(
    (field: string) => fieldErrors[field] !== undefined,
    [fieldErrors]
  )

  return {
    error,
    fieldErrors,
    setError: handleError,
    parseError,
    clearError,
    clearFieldError,
    hasError,
    hasFieldError,
  }
}

/**
 * Parse API response and throw structured error if not ok
 */
export async function parseAuthResponse<T>(response: Response): Promise<T> {
  if (!response.ok) {
    const data = await response.json().catch(() => ({ error: 'Unknown error' }))
    const errorResponse: AuthErrorResponse = {
      error_code: data.error_code || ErrorCode.INTERNAL_ERROR,
      message: data.message || data.error || '请求失败',
      field: data.field,
    }
    throw new Error(JSON.stringify(errorResponse))
  }
  return response.json()
}

/**
 * Create an AbortController with timeout for fetch requests
 */
export function createTimeoutController(timeoutMs: number = 10000): {
  controller: AbortController
  timeoutId: number
} {
  const controller = new AbortController()
  const timeoutId = window.setTimeout(() => controller.abort(), timeoutMs)
  return { controller, timeoutId }
}