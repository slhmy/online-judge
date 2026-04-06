import type { Metadata } from 'next'
import { Inter } from 'next/font/google'
import './globals.css'
import { Layout } from '@/components/layout/Layout'
import { Providers } from '@/components/providers/Providers'

const inter = Inter({ subsets: ['latin'] })

export const metadata: Metadata = {
  title: 'Online Judge',
  description: 'A modern competitive programming platform',
}

export default function RootLayout({
  children,
}: {
  children: React.ReactNode
}) {
  return (
    <html lang="en">
      <body className={inter.className}>
        <Providers>
          <Layout>{children}</Layout>
        </Providers>
      </body>
    </html>
  )
}