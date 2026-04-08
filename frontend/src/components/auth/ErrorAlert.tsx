'use client'

import { ParsedError, ErrorType, errorTypeConfig } from '@/types/auth'
import { cn } from '@/lib/utils'

export interface ErrorAlertProps {
  error: ParsedError | null
  onDismiss?: () => void
  className?: string
  showIcon?: boolean
  showField?: boolean
}

/**
 * Error alert component for displaying authentication errors
 */
export function ErrorAlert({
  error,
  onDismiss,
  className,
  showIcon = true,
  showField = true,
}: ErrorAlertProps) {
  if (!error) return null

  const config = errorTypeConfig[error.type]
  const displayMessage = showField && error.fieldLabel
    ? `${error.fieldLabel}: ${error.message}`
    : error.message

  return (
    <div
      role="alert"
      className={cn(
        'flex items-start gap-3 p-3 rounded-md transition-all',
        config.bgColor,
        config.color,
        className
      )}
    >
      {showIcon && (
        <span className="text-lg flex-shrink-0" aria-hidden="true">
          {config.icon}
        </span>
      )}
      <div className="flex-1 min-w-0">
        <p className="text-sm font-medium break-words">{displayMessage}</p>
        {error.code && (
          <p className="text-xs opacity-70 mt-1">
            错误码: {error.code}
          </p>
        )}
      </div>
      {onDismiss && (
        <button
          type="button"
          onClick={onDismiss}
          className="flex-shrink-0 p-1 hover:opacity-70 transition-opacity"
          aria-label="关闭错误提示"
        >
          <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
            <path
              fillRule="evenodd"
              d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z"
              clipRule="evenodd"
            />
          </svg>
        </button>
      )}
    </div>
  )
}

export interface FieldErrorProps {
  error: ParsedError | null | undefined
  className?: string
}

/**
 * Inline field error component for displaying errors next to input fields
 */
export function FieldError({ error, className }: FieldErrorProps) {
  if (!error) return null

  const config = errorTypeConfig[error.type]

  return (
    <p
      role="alert"
      className={cn(
        'text-sm mt-1 flex items-center gap-1',
        config.color,
        className
      )}
    >
      <span aria-hidden="true">•</span>
      {error.message}
    </p>
  )
}

export interface ErrorListProps {
  errors: Record<string, ParsedError>
  onDismissField?: (field: string) => void
  className?: string
}

/**
 * Component for displaying multiple field errors in a list
 */
export function ErrorList({ errors, onDismissField, className }: ErrorListProps) {
  const errorEntries = Object.entries(errors)
  if (errorEntries.length === 0) return null

  return (
    <div className={cn('space-y-2', className)}>
      {errorEntries.map(([field, error]) => (
        <ErrorAlert
          key={field}
          error={error}
          showField={true}
          onDismiss={onDismissField ? () => onDismissField(field) : undefined}
        />
      ))}
    </div>
  )
}