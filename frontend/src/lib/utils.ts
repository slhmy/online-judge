import { clsx, type ClassValue } from 'clsx'
import { twMerge } from 'tailwind-merge'

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}

export function formatTime(seconds: number): string {
  if (seconds < 1) {
    return `${Math.round(seconds * 1000)} ms`
  }
  return `${seconds.toFixed(2)} s`
}

export function formatMemory(kilobytes: number): string {
  if (kilobytes < 1024) {
    return `${kilobytes} KB`
  }
  return `${(kilobytes / 1024).toFixed(2)} MB`
}

export function formatDate(dateString: string): string {
  return new Date(dateString).toLocaleString()
}

export function formatContestTime(startTime: string, currentTime: Date = new Date()): string {
  const start = new Date(startTime)
  const diff = start.getTime() - currentTime.getTime()

  if (diff < 0) {
    return 'Contest started'
  }

  const hours = Math.floor(diff / (1000 * 60 * 60))
  const minutes = Math.floor((diff % (1000 * 60 * 60)) / (1000 * 60))
  const seconds = Math.floor((diff % (1000 * 60)) / 1000)

  return `${hours.toString().padStart(2, '0')}:${minutes.toString().padStart(2, '0')}:${seconds.toString().padStart(2, '0')}`
}