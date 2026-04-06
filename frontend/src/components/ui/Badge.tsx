import { VERDICT_CONFIG, type Verdict } from '@/types'
import { cn } from '@/lib/utils'

interface VerdictBadgeProps {
  verdict: Verdict
  size?: 'sm' | 'md' | 'lg'
  showLabel?: boolean
}

export function VerdictBadge({ verdict, size = 'md', showLabel = true }: VerdictBadgeProps) {
  const config = VERDICT_CONFIG[verdict] || { color: 'bg-gray-500', label: verdict, icon: '?' }

  const sizeClasses = {
    sm: 'text-xs px-1.5 py-0.5',
    md: 'text-sm px-2 py-1',
    lg: 'text-base px-3 py-1.5',
  }

  return (
    <span
      className={cn(
        'rounded font-medium text-white inline-flex items-center gap-1',
        config.color,
        sizeClasses[size]
      )}
    >
      {showLabel ? (
        <>
          <span className="text-xs">{config.icon}</span>
          {config.label}
        </>
      ) : (
        verdict.toUpperCase()
      )}
    </span>
  )
}

interface DifficultyBadgeProps {
  difficulty: 'easy' | 'medium' | 'hard'
}

export function DifficultyBadge({ difficulty }: DifficultyBadgeProps) {
  const colors = {
    easy: 'text-green-700 bg-green-100',
    medium: 'text-yellow-700 bg-yellow-100',
    hard: 'text-red-700 bg-red-100',
  }

  return (
    <span
      className={cn(
        'text-xs font-medium px-2 py-1 rounded capitalize',
        colors[difficulty]
      )}
    >
      {difficulty}
    </span>
  )
}