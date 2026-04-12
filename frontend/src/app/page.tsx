import Link from 'next/link'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'

export default function Home() {
  return (
    <main className="flex min-h-screen flex-col items-center justify-center p-6 md:p-12">
      <div className="z-10 w-full max-w-5xl items-center justify-center text-sm">
        <h1 className="mb-8 text-center font-heading text-4xl font-bold text-foreground">
          Online Judge Platform
        </h1>

        <div className="mb-8 grid grid-cols-1 gap-6 md:grid-cols-3">
          <Link href="/problems" className="block">
            <Card className="h-full transition-colors hover:bg-muted/40">
              <CardHeader>
                <CardTitle>Problems</CardTitle>
                <CardDescription>Browse and solve programming problems</CardDescription>
              </CardHeader>
              <CardContent className="text-sm text-primary">Explore</CardContent>
            </Card>
          </Link>

          <Link href="/contests" className="block">
            <Card className="h-full transition-colors hover:bg-muted/40">
              <CardHeader>
                <CardTitle>Contests</CardTitle>
                <CardDescription>Participate in programming contests</CardDescription>
              </CardHeader>
              <CardContent className="text-sm text-primary">Join</CardContent>
            </Card>
          </Link>

          <Link href="/submissions" className="block">
            <Card className="h-full transition-colors hover:bg-muted/40">
              <CardHeader>
                <CardTitle>Submissions</CardTitle>
                <CardDescription>View your submission history</CardDescription>
              </CardHeader>
              <CardContent className="text-sm text-primary">Track</CardContent>
            </Card>
          </Link>
        </div>

        <div className="text-center">
          <Button nativeButton={false} render={<Link href="/login" />}>Get Started</Button>
        </div>
      </div>
    </main>
  )
}