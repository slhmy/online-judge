import type { Config } from 'tailwindcss'

const config: Config = {
  content: [
    './src/pages/**/*.{js,ts,jsx,tsx,mdx}',
    './src/components/**/*.{js,ts,jsx,tsx,mdx}',
    './src/app/**/*.{js,ts,jsx,tsx,mdx}',
  ],
  darkMode: 'class',
  safelist: [
    // Verdict colors (used dynamically)
    'bg-green-500',
    'bg-red-500',
    'bg-yellow-500',
    'bg-orange-500',
    'bg-purple-500',
    'bg-blue-500',
    'bg-pink-500',
    'bg-gray-500',
  ],
  theme: {
    extend: {
      colors: {
        verdict: {
          correct: 'rgb(34 197 94)',
          'wrong-answer': 'rgb(239 68 68)',
          'timelimit': 'rgb(234 179 8)',
          'memory-limit': 'rgb(249 115 22)',
          'run-error': 'rgb(168 85 247)',
          'compiler-error': 'rgb(59 130 246)',
          'output-limit': 'rgb(236 72 153)',
          'presentation': 'rgb(107 114 128)',
        },
      },
    },
  },
  plugins: [
    require('@tailwindcss/typography'),
  ],
}
export default config