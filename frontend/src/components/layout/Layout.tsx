'use client'

import { useState, useEffect } from 'react'
import Link from 'next/link'
import { usePathname, useRouter } from 'next/navigation'
import { clsx } from 'clsx'
import { useAuthStore } from '@/stores/authStore'
import { MoonIcon, SunIcon, ChevronDownIcon, LogOutIcon, ShieldIcon, UserIcon } from 'lucide-react'
import { Button, buttonVariants } from '@/components/ui/button'
import { Avatar, AvatarFallback } from '@/components/ui/avatar'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { cn } from '@/lib/utils'

export function Layout({ children }: { children: React.ReactNode }) {
  const pathname = usePathname()
  const router = useRouter()
  const { user, isAuthenticated, logout } = useAuthStore()
  const [darkMode, setDarkMode] = useState(true)

  useEffect(() => {
    // Check localStorage for saved preference
    const saved = localStorage.getItem('darkMode')
    if (saved !== null) {
      setDarkMode(saved === 'true')
    }
  }, [])

  useEffect(() => {
    // Apply dark mode class
    if (darkMode) {
      document.documentElement.classList.add('dark')
    } else {
      document.documentElement.classList.remove('dark')
    }
    // Save preference
    localStorage.setItem('darkMode', String(darkMode))
  }, [darkMode])

  const handleLogout = () => {
    logout()
    router.push('/')
  }

  const navLinks = [
    { href: '/', label: 'Home' },
    { href: '/problems', label: 'Problems' },
    { href: '/submissions', label: 'Submissions' },
  ]

  return (
    <div className="min-h-screen bg-background text-foreground">
      <header className="sticky top-0 z-40 border-b bg-background/95 backdrop-blur">
        <nav className="mx-auto flex h-16 w-full max-w-7xl items-center justify-between px-4 sm:px-6 lg:px-8">
          <div className="flex items-center gap-6">
            <div className="flex items-center">
              <Link href="/" className="text-xl font-semibold tracking-tight text-foreground">
                  Online Judge
                </Link>
            </div>
            <div className="hidden items-center gap-1 sm:flex">
                {navLinks.map((link) => (
                  <Link
                    key={link.href}
                    href={link.href}
                    className={clsx(
                      buttonVariants({ variant: pathname === link.href ? 'secondary' : 'ghost', size: 'sm' }),
                      'rounded-md',
                      pathname === link.href
                        ? 'text-foreground'
                        : 'text-muted-foreground'
                    )}
                  >
                    {link.label}
                  </Link>
                ))}
            </div>
          </div>

          <div className="flex items-center gap-2">
            <Button
              variant="ghost"
              size="icon"
                onClick={() => setDarkMode(!darkMode)}
                title="Toggle theme"
              >
                {darkMode ? (
                  <SunIcon />
                ) : (
                  <MoonIcon />
                )}
            </Button>

              {isAuthenticated && user ? (
                <DropdownMenu>
                  <DropdownMenuTrigger
                    render={
                      <Button
                        variant="outline"
                        className="h-9 w-fit justify-start border-zinc-300/70 bg-white px-2 hover:bg-zinc-50 dark:border-zinc-700 dark:bg-zinc-900 dark:hover:bg-zinc-800"
                      />
                    }
                  >
                    <Avatar size="sm">
                      <AvatarFallback>
                        {user.username?.[0]?.toUpperCase() || user.email?.[0]?.toUpperCase() || 'U'}
                      </AvatarFallback>
                    </Avatar>
                    <div className="hidden min-w-0 flex-1 text-left sm:grid">
                      <span className="truncate text-sm font-medium">{user.username || user.email}</span>
                    </div>
                    <ChevronDownIcon className="text-muted-foreground" />
                  </DropdownMenuTrigger>
                  <DropdownMenuContent align="end" className="w-52">
                    <DropdownMenuGroup>
                      <DropdownMenuLabel>My Account</DropdownMenuLabel>
                      <DropdownMenuItem
                        onClick={() => router.push('/profile')}
                      >
                        <UserIcon />
                        Profile
                      </DropdownMenuItem>
                      {user.role === 'admin' && (
                        <DropdownMenuItem onClick={() => router.push('/admin')}>
                          <ShieldIcon />
                          Admin Dashboard
                        </DropdownMenuItem>
                      )}
                    </DropdownMenuGroup>
                    <DropdownMenuSeparator />
                    <DropdownMenuItem variant="destructive" onClick={handleLogout}>
                      <LogOutIcon />
                      Logout
                    </DropdownMenuItem>
                  </DropdownMenuContent>
                </DropdownMenu>
              ) : (
                <div className="flex items-center gap-2">
                  <Link
                    href="/login"
                    className={cn(buttonVariants({ variant: 'ghost', size: 'sm' }), 'text-muted-foreground')}
                  >
                    Login
                  </Link>
                  <Link
                    href="/register"
                    className={buttonVariants({ size: 'sm' })}
                  >
                    Register
                  </Link>
                </div>
              )}
          </div>
        </nav>
      </header>
      <main className="mx-auto w-full max-w-7xl px-4 py-6 sm:px-6 lg:px-8">
        {children}
      </main>
    </div>
  )
}