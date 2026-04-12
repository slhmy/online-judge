'use client'

import Link from 'next/link'
import { usePathname } from 'next/navigation'
import { cn } from '@/lib/utils'
import { buttonVariants } from '@/components/ui/button'

const adminNav = [
  { href: '/admin', label: 'User Management' },
  { href: '/admin/problems', label: 'Problem Management' },
  { href: '/contests', label: 'Contest Overview' },
]

export default function AdminLayout({ children }: { children: React.ReactNode }) {
  const pathname = usePathname()

  const isActive = (href: string) => {
    if (href === '/admin') return pathname === '/admin'
    return pathname.startsWith(href)
  }

  return (
    <div className="grid grid-cols-1 gap-4 lg:grid-cols-[220px_1fr]">
      <aside className="h-fit rounded-xl border border-border bg-card p-3 lg:sticky lg:top-20">
        <div className="px-2 pb-2 text-xs font-medium uppercase tracking-wider text-muted-foreground">
          Admin Menu
        </div>
        <nav className="flex flex-col gap-1">
          {adminNav.map((item) => (
            <Link
              key={item.href}
              href={item.href}
              className={cn(
                buttonVariants({
                  variant: isActive(item.href) ? 'secondary' : 'ghost',
                  size: 'sm',
                }),
                'justify-start'
              )}
            >
              {item.label}
            </Link>
          ))}
        </nav>
      </aside>

      <section>{children}</section>
    </div>
  )
}
