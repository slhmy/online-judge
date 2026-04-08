// Auth error codes matching backend AuthErrorCode
export enum ErrorCode {
  INVALID_CREDENTIALS = 'INVALID_CREDENTIALS',
  EMAIL_EXISTS = 'EMAIL_EXISTS',
  VALIDATION_ERROR = 'VALIDATION_ERROR',
  UNAUTHORIZED = 'UNAUTHORIZED',
  OAUTH_NOT_CONFIGURED = 'OAUTH_NOT_CONFIGURED',
  OAUTH_STATE_EXPIRED = 'OAUTH_STATE_EXPIRED',
  OAUTH_FAILED = 'OAUTH_FAILED',
  TOKEN_INVALID = 'TOKEN_INVALID',
  DATABASE_ERROR = 'DATABASE_ERROR',
  INTERNAL_ERROR = 'INTERNAL_ERROR',
  NETWORK_ERROR = 'NETWORK_ERROR',
  TIMEOUT_ERROR = 'TIMEOUT_ERROR',
}

// Auth error response from backend
export interface AuthErrorResponse {
  error_code: ErrorCode
  message: string
  field?: string
}

// Error type classification for display styling
export enum ErrorType {
  VALIDATION = 'validation',    // Input validation errors (yellow)
  AUTH = 'auth',               // Authentication errors (red)
  SYSTEM = 'system',           // System/internal errors (gray)
  NETWORK = 'network',         // Network connectivity errors (orange)
}

// Chinese error messages for each error code
export const authErrorMessages: Record<ErrorCode, string> = {
  [ErrorCode.INVALID_CREDENTIALS]: '邮箱或密码错误',
  [ErrorCode.EMAIL_EXISTS]: '该邮箱已被注册',
  [ErrorCode.VALIDATION_ERROR]: '输入验证失败',
  [ErrorCode.UNAUTHORIZED]: '未授权访问',
  [ErrorCode.OAUTH_NOT_CONFIGURED]: 'OAuth未配置',
  [ErrorCode.OAUTH_STATE_EXPIRED]: '授权状态已过期，请重新登录',
  [ErrorCode.OAUTH_FAILED]: 'OAuth登录失败',
  [ErrorCode.TOKEN_INVALID]: '登录已过期，请重新登录',
  [ErrorCode.DATABASE_ERROR]: '数据库错误，请稍后重试',
  [ErrorCode.INTERNAL_ERROR]: '系统错误，请稍后重试',
  [ErrorCode.NETWORK_ERROR]: '网络连接失败，请检查网络',
  [ErrorCode.TIMEOUT_ERROR]: '请求超时，请重试',
}

// Error code to error type mapping
export const errorCodeToType: Record<ErrorCode, ErrorType> = {
  [ErrorCode.INVALID_CREDENTIALS]: ErrorType.AUTH,
  [ErrorCode.EMAIL_EXISTS]: ErrorType.VALIDATION,
  [ErrorCode.VALIDATION_ERROR]: ErrorType.VALIDATION,
  [ErrorCode.UNAUTHORIZED]: ErrorType.AUTH,
  [ErrorCode.OAUTH_NOT_CONFIGURED]: ErrorType.SYSTEM,
  [ErrorCode.OAUTH_STATE_EXPIRED]: ErrorType.AUTH,
  [ErrorCode.OAUTH_FAILED]: ErrorType.SYSTEM,
  [ErrorCode.TOKEN_INVALID]: ErrorType.AUTH,
  [ErrorCode.DATABASE_ERROR]: ErrorType.SYSTEM,
  [ErrorCode.INTERNAL_ERROR]: ErrorType.SYSTEM,
  [ErrorCode.NETWORK_ERROR]: ErrorType.NETWORK,
  [ErrorCode.TIMEOUT_ERROR]: ErrorType.NETWORK,
}

// Field name translations for Chinese display
export const fieldTranslations: Record<string, string> = {
  email: '邮箱',
  password: '密码',
  username: '用户名',
  confirm_password: '确认密码',
  refresh_token: '刷新令牌',
}

// Error type styling configuration
export const errorTypeConfig: Record<ErrorType, { color: string; bgColor: string; icon: string }> = {
  [ErrorType.VALIDATION]: {
    color: 'text-yellow-700 dark:text-yellow-300',
    bgColor: 'bg-yellow-100 dark:bg-yellow-900/50',
    icon: '⚠',
  },
  [ErrorType.AUTH]: {
    color: 'text-red-700 dark:text-red-300',
    bgColor: 'bg-red-100 dark:bg-red-900/50',
    icon: '🔒',
  },
  [ErrorType.SYSTEM]: {
    color: 'text-gray-700 dark:text-gray-300',
    bgColor: 'bg-gray-100 dark:bg-gray-700/50',
    icon: '⚙',
  },
  [ErrorType.NETWORK]: {
    color: 'text-orange-700 dark:text-orange-300',
    bgColor: 'bg-orange-100 dark:bg-orange-900/50',
    icon: '📡',
  },
}