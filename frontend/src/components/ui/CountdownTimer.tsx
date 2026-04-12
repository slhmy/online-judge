'use client'

import { useEffect, useState } from 'react'

interface CountdownTimerProps {
  targetTime: string // ISO timestamp
  onEnd?: () => void
  showDays?: boolean
}

export function CountdownTimer({ targetTime, onEnd, showDays = true }: CountdownTimerProps) {
  const [timeLeft, setTimeLeft] = useState(calculateTimeLeft())

  function calculateTimeLeft() {
    const target = new Date(targetTime).getTime()
    const now = Date.now()
    const diff = target - now

    if (diff <= 0) {
      return { days: 0, hours: 0, minutes: 0, seconds: 0, expired: true }
    }

    return {
      days: Math.floor(diff / (1000 * 60 * 60 * 24)),
      hours: Math.floor((diff % (1000 * 60 * 60 * 24)) / (1000 * 60 * 60)),
      minutes: Math.floor((diff % (1000 * 60 * 60)) / (1000 * 60)),
      seconds: Math.floor((diff % (1000 * 60)) / 1000),
      expired: false,
    }
  }

  useEffect(() => {
    const timer = setInterval(() => {
      const newTimeLeft = calculateTimeLeft()
      setTimeLeft(newTimeLeft)

      if (newTimeLeft.expired && onEnd) {
        onEnd()
      }
    }, 1000)

    return () => clearInterval(timer)
  }, [targetTime, onEnd])

  if (timeLeft.expired) {
    return <span className="text-red-600 dark:text-red-400 font-medium">Ended</span>
  }

  const pad = (n: number) => n.toString().padStart(2, '0')

  return (
    <span className="font-mono text-lg text-foreground">
      {showDays && timeLeft.days > 0 && (
        <span className="mr-1">
          <span className="bg-muted px-2 py-1 rounded">{timeLeft.days}</span>
          <span className="text-muted-foreground mx-1">d</span>
        </span>
      )}
      <span className="bg-muted px-2 py-1 rounded">{pad(timeLeft.hours)}</span>
      <span className="text-muted-foreground mx-1">:</span>
      <span className="bg-muted px-2 py-1 rounded">{pad(timeLeft.minutes)}</span>
      <span className="text-muted-foreground mx-1">:</span>
      <span className="bg-muted px-2 py-1 rounded">{pad(timeLeft.seconds)}</span>
    </span>
  )
}