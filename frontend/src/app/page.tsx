import Link from 'next/link'

export default function Home() {
  return (
    <main className="flex min-h-screen flex-col items-center justify-center p-24">
      <div className="z-10 max-w-5xl w-full items-center justify-center font-mono text-sm">
        <h1 className="text-4xl font-bold text-center mb-8 text-gray-900 dark:text-gray-100">
          Online Judge Platform
        </h1>

        <div className="grid grid-cols-1 md:grid-cols-3 gap-6 mb-8">
          <Link
            href="/problems"
            className="group rounded-lg border border-gray-300 dark:border-gray-700 bg-white dark:bg-gray-800 px-5 py-4 transition-colors hover:border-gray-400 dark:hover:border-gray-600 hover:bg-gray-50 dark:hover:bg-gray-700"
          >
            <h2 className="mb-3 text-2xl font-semibold text-gray-900 dark:text-gray-100">
              Problems →
            </h2>
            <p className="m-0 max-w-[30ch] text-sm text-gray-600 dark:text-gray-400">
              Browse and solve programming problems
            </p>
          </Link>

          <Link
            href="/contests"
            className="group rounded-lg border border-gray-300 dark:border-gray-700 bg-white dark:bg-gray-800 px-5 py-4 transition-colors hover:border-gray-400 dark:hover:border-gray-600 hover:bg-gray-50 dark:hover:bg-gray-700"
          >
            <h2 className="mb-3 text-2xl font-semibold text-gray-900 dark:text-gray-100">
              Contests →
            </h2>
            <p className="m-0 max-w-[30ch] text-sm text-gray-600 dark:text-gray-400">
              Participate in programming contests
            </p>
          </Link>

          <Link
            href="/submissions"
            className="group rounded-lg border border-gray-300 dark:border-gray-700 bg-white dark:bg-gray-800 px-5 py-4 transition-colors hover:border-gray-400 dark:hover:border-gray-600 hover:bg-gray-50 dark:hover:bg-gray-700"
          >
            <h2 className="mb-3 text-2xl font-semibold text-gray-900 dark:text-gray-100">
              Submissions →
            </h2>
            <p className="m-0 max-w-[30ch] text-sm text-gray-600 dark:text-gray-400">
              View your submission history
            </p>
          </Link>
        </div>

        <div className="text-center">
          <Link
            href="/login"
            className="rounded-full bg-blue-600 text-white px-6 py-3 font-medium hover:bg-blue-700 transition-colors"
          >
            Get Started
          </Link>
        </div>
      </div>
    </main>
  )
}