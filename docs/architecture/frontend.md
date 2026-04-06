# Frontend Architecture

## Overview

This document outlines the frontend architecture for the Online Judge platform, using Next.js for the web frontend with a Go BFF (Backend for Frontend) layer for API aggregation.

## Architecture Pattern

**BFF (Backend for Frontend)** pattern - A dedicated Go service that aggregates backend microservices and provides a unified API optimized for the frontend.

```
┌─────────────────────────────────────────────────────────────────┐
│                         Next.js Frontend                         │
│  - SSR/SSG for SEO optimization                                  │
│  - React Server Components                                       │
│  - Client-side hydration                                         │
└─────────────────────────────┬───────────────────────────────────┘
                              │ HTTP/WebSocket
                              │
┌─────────────────────────────▼───────────────────────────────────┐
│                         Go BFF Layer                             │
│  - API aggregation from backend services                         │
│  - Authentication proxy                                          │
│  - WebSocket connection manager                                  │
│  - Response transformation                                       │
└─────────────────────────────┬───────────────────────────────────┘
                              │ gRPC
                              │
┌─────────────────────────────▼───────────────────────────────────┐
│                    Backend Microservices                         │
│  (User, Problem, Submission, Contest, Notification)              │
└─────────────────────────────────────────────────────────────────┘
```

## Technology Stack

### Frontend (Next.js)
| Category | Technology | Justification |
|----------|------------|---------------|
| Framework | Next.js 14+ (App Router) | SSR/SSG, Server Components, streaming |
| Language | TypeScript | Type safety, better IDE support |
| Styling | Tailwind CSS + shadcn/ui | Rapid development, consistent design |
| Code Editor | Monaco Editor | VS Code engine, multi-language support |
| State Management | Zustand (client) + Server Actions | Simple global state, server mutations |
| Data Fetching | TanStack Query + Server Actions | Client caching, server mutations |
| Forms | React Hook Form + Zod | Performant forms, schema validation |

### BFF Layer (Go)
| Category | Technology | Justification |
|----------|------------|---------------|
| Language | Go 1.21+ | High performance, concurrent request handling |
| HTTP Framework | Gin or Fiber | Fast routing, middleware support |
| gRPC Client | grpc-go | Connect to backend microservices |
| WebSocket | gorilla/websocket | Real-time communication |
| Cache | Redis | Session storage, response caching |

## Project Structure

### Next.js Frontend
```
frontend/
├── src/
│   ├── app/                       # Next.js App Router
│   │   ├── layout.tsx             # Root layout
│   │   ├── page.tsx               # Home page
│   │   ├── (auth)/                # Auth route group
│   │   │   ├── login/
│   │   │   │   └── page.tsx
│   │   │   └── register/
│   │   │   │   └── page.tsx
│   │   ├── problems/
│   │   │   ├── page.tsx           # Problem list
│   │   │   └── [id]/
│   │   │   │   ├── page.tsx       # Problem detail (SSR)
│   │   │   │   └── submit/
│   │   │   │   │   └── page.tsx   # Submit page
│   │   ├── submissions/
│   │   │   ├── page.tsx           # Submission list
│   │   │   └── [id]/
│   │   │   │   └── page.tsx       # Submission detail
│   │   ├── contests/
│   │   │   ├── page.tsx           # Contest list
│   │   │   └── [id]/
│   │   │   │   └── page.tsx       # Contest detail
│   │   ├── profile/
│   │   │   └── page.tsx           # User profile
│   │   └── api/                   # Next.js API routes (optional)
│   │   │   └── revalidate/
│   │   │   │   └── [tag]/route.ts # Cache revalidation
│   ├── components/
│   │   ├── ui/                    # shadcn/ui components
│   │   ├── editor/                # Code editor components
│   │   ├── problem/               # Problem display components
│   │   ├── submission/            # Submission status components
│   │   ├── contest/               # Contest-related components
│   │   └── layout/                # Layout components
│   ├── hooks/
│   │   ├── useAuth.ts
│   │   ├── useWebSocket.ts
│   │   └── useSubmissionStatus.ts
│   ├── lib/
│   │   ├── bff-client.ts          # BFF API client
│   │   ├── websocket.ts           # WebSocket manager
│   │   └── utils.ts
│   ├── stores/
│   │   ├── authStore.ts           # Client auth state
│   │   └── editorStore.ts         # Editor preferences
│   ├── types/
│   │   ├── problem.ts
│   │   ├── submission.ts
│   │   ├── contest.ts
│   │   └── user.ts
│   └── actions/                   # Server Actions
│   │   ├── auth.ts
│   │   ├── problems.ts
│   │   ├── submissions.ts
│   │   └── contests.ts
├── public/
├── middleware.ts                  # Auth middleware
├── next.config.js
├── tailwind.config.ts
├── tsconfig.json
└── package.json
```

### Go BFF Layer
```
bff/
├── cmd/
│   └── main.go                    # Entry point
├── internal/
│   ├── handler/
│   │   ├── auth.go                # Auth endpoints
│   │   ├── problem.go             # Problem endpoints
│   │   ├── submission.go          # Submission endpoints
│   │   ├── contest.go             # Contest endpoints
│   │   └── user.go                # User endpoints
│   ├── grpc/
│   │   ├── user_client.go         # User service client
│   │   ├── problem_client.go      # Problem service client
│   │   ├── submission_client.go   # Submission service client
│   │   ├── contest_client.go      # Contest service client
│   │   └── notification_client.go # Notification service client
│   ├── websocket/
│   │   ├── manager.go             # WebSocket connection manager
│   │   ├── hub.go                 # Message hub
│   │   └── client.go              # Client connection
│   ├── middleware/
│   │   ├── auth.go                # JWT validation
│   │   ├── cors.go                # CORS handling
│   │   └── ratelimit.go           # Rate limiting
│   ├── aggregator/
│   │   ├── problem.go             # Aggregate problem + test cases
│   │   ├── contest.go             # Aggregate contest + rankings
│   │   └── dashboard.go           # Aggregate user dashboard
│   ├── transformer/
│   │   ├── response.go            # Transform gRPC to HTTP response
│   │   └── request.go             # Transform HTTP to gRPC request
│   └── config/
│   │   └── config.go
├── pkg/
│   ├── cache/
│   │   └── redis.go               # Redis client
│   └── session/
│   │   └── session.go             # Session management
├── gen/                           # Generated gRPC stubs
├── go.mod
├── go.sum
├── Makefile
└── Dockerfile
```

## BFF API Design

### REST Endpoints (Go BFF)
```
POST   /api/v1/auth/register           → User Service.Register
POST   /api/v1/auth/login              → User Service.Login + Session
POST   /api/v1/auth/logout             → User Service.Logout + Session Clear
GET    /api/v1/auth/me                 → User Service.GetMe (cached)

GET    /api/v1/problems                → Problem Service.ListProblems
GET    /api/v1/problems/:id            → Problem Service.GetProblem + TestCases (aggregated)
POST   /api/v1/problems                → Problem Service.CreateProblem (admin)

POST   /api/v1/submissions             → Submission Service.Create + Queue
GET    /api/v1/submissions/:id         → Submission Service.GetSubmission
GET    /api/v1/submissions             → Submission Service.ListSubmissions (filtered)
GET    /api/v1/submissions/:id/result  → Submission Service.GetResult

GET    /api/v1/contests                → Contest Service.ListContests
GET    /api/v1/contests/:id            → Contest Service.GetContest + Problems (aggregated)
GET    /api/v1/contests/:id/rankings   → Contest Service.GetRankings (cached in Redis)
POST   /api/v1/contests/:id/register   → Contest Service.Register

WS     /ws                             → WebSocket for real-time updates
```

### BFF Handler Implementation
```go
// internal/handler/problem.go
package handler

import (
    "net/http"
    
    "github.com/gin-gonic/gin"
    "github.com/online-judge/bff/internal/grpc"
    "github.com/online-judge/bff/internal/aggregator"
    "github.com/online-judge/bff/internal/transformer"
)

type ProblemHandler struct {
    problemClient *grpc.ProblemClient
    aggregator    *aggregator.ProblemAggregator
}

func (h *ProblemHandler) GetProblem(c *gin.Context) {
    problemID := c.Param("id")
    
    // Fetch problem details from gRPC service
    problem, err := h.problemClient.GetProblem(c.Request.Context(), problemID)
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "problem not found"})
        return
    }
    
    // Aggregate with test cases (sample only for non-admin)
    sampleCases, err := h.aggregator.GetSampleTestCases(c.Request.Context(), problemID)
    if err != nil {
        sampleCases = nil
    }
    
    // Transform to frontend-friendly response
    response := transformer.ToProblemResponse(problem, sampleCases)
    c.JSON(http.StatusOK, response)
}
```

### gRPC Aggregation
```go
// internal/aggregator/contest.go
package aggregator

import (
    "context"
    
    contestv1 "github.com/online-judge/backend/gen/go/contest/v1"
    problemv1 "github.com/online-judge/backend/gen/go/problem/v1"
)

type ContestAggregator struct {
    contestClient  contestv1.ContestServiceClient
    problemClient  problemv1.ProblemServiceClient
}

func (a *ContestAggregator) GetContestDetail(ctx context.Context, contestID string) (*ContestDetail, error) {
    // Fetch contest info
    contest, err := a.contestClient.GetContest(ctx, &contestv1.GetContestRequest{Id: contestID})
    if err != nil {
        return nil, err
    }
    
    // Fetch contest problems
    problemsResp, err := a.contestClient.GetContestProblems(ctx, &contestv1.GetContestProblemsRequest{ContestId: contestID})
    if err != nil {
        return nil, err
    }
    
    // Fetch rankings (cached)
    rankings, err := a.contestClient.GetRankings(ctx, &contestv1.GetRankingsRequest{ContestId: contestID})
    
    // Combine into single response
    return &ContestDetail{
        Contest:   contest,
        Problems:  problemsResp.Problems,
        Rankings:  rankings.Entries,
    }, nil
}
```

## WebSocket Manager (BFF)

```go
// internal/websocket/manager.go
package websocket

import (
    "sync"
    
    "github.com/gorilla/websocket"
)

type Manager struct {
    clients    map[string]*Client // userID -> Client
    hubs       map[string]*Hub    // submissionID/contestID -> Hub
    register   chan *Client
    unregister chan *Client
    mu         sync.RWMutex
}

func (m *Manager) Run() {
    for {
        select {
        case client := <-m.register:
            m.mu.Lock()
            m.clients[client.userID] = client
            m.mu.Unlock()
            
        case client := <-m.unregister:
            m.mu.Lock()
            delete(m.clients, client.userID)
            m.mu.Unlock()
            client.conn.Close()
        }
    }
}

// Subscribe to submission updates
func (m *Manager) SubscribeSubmission(userID string, submissionID string) {
    m.mu.Lock()
    if client, ok := m.clients[userID]; ok {
        hub := m.getOrCreateHub(submissionID)
        hub.subscribe(client)
    }
    m.mu.Unlock()
}

// Broadcast judging progress
func (m *Manager) BroadcastSubmissionProgress(submissionID string, progress *JudgeProgress) {
    m.mu.RLock()
    if hub, ok := m.hubs[submissionID]; ok {
        hub.broadcast(progress)
    }
    m.mu.RUnlock()
}
```

## Next.js Frontend Implementation

### Server Component (Problem Detail - SSR)
```typescript
// src/app/problems/[id]/page.tsx
import { bffClient } from '@/lib/bff-client';
import { ProblemDescription } from '@/components/problem/ProblemDescription';
import { CodeEditor } from '@/components/editor/CodeEditor';

interface ProblemPageProps {
  params: { id: string };
}

// Server-side fetch with caching
export default async function ProblemPage({ params }: ProblemPageProps) {
  const problem = await bffClient.getProblem(params.id);
  
  return (
    <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
      <ProblemDescription problem={problem} />
      <CodeEditor 
        problemId={params.id}
        language={problem.defaultLanguage}
        timeLimit={problem.timeLimit}
        memoryLimit={problem.memoryLimit}
      />
    </div>
  );
}

// Generate static pages for published problems
export async function generateStaticParams() {
  const problems = await bffClient.getProblems({ publishedOnly: true });
  return problems.map((p) => ({ id: p.id }));
}
```

### BFF Client (TypeScript)
```typescript
// src/lib/bff-client.ts
import { cache } from 'react';

const BFF_URL = process.env.NEXT_PUBLIC_BFF_URL || 'http://bff:8080';

class BFFClient {
  private baseUrl: string;
  private token: string | null = null;

  constructor(baseUrl: string) {
    this.baseUrl = baseUrl;
  }

  setToken(token: string) {
    this.token = token;
  }

  private async fetch<T>(path: string, options?: RequestInit): Promise<T> {
    const headers: HeadersInit = {
      'Content-Type': 'application/json',
    };
    if (this.token) {
      headers['Authorization'] = `Bearer ${this.token}`;
    }

    const res = await fetch(`${this.baseUrl}${path}`, {
      ...options,
      headers,
    });

    if (!res.ok) {
      throw new Error(`BFF error: ${res.status}`);
    }

    return res.json();
  }

  // Cached server-side fetch for SSR
  getProblem = cache(async (id: string): Promise<Problem> => {
    return this.fetch<Problem>(`/api/v1/problems/${id}`);
  });

  async getProblems(filters?: ProblemFilters): Promise<ProblemList> {
    const params = new URLSearchParams(filters as Record<string, string>);
    return this.fetch<ProblemList>(`/api/v1/problems?${params}`);
  }

  async createSubmission(data: CreateSubmissionRequest): Promise<Submission> {
    return this.fetch<Submission>('/api/v1/submissions', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  async getSubmission(id: string): Promise<Submission> {
    return this.fetch<Submission>(`/api/v1/submissions/${id}`);
  }

  async login(username: string, password: string): Promise<LoginResponse> {
    return this.fetch<LoginResponse>('/api/v1/auth/login', {
      method: 'POST',
      body: JSON.stringify({ username, password }),
    });
  }
}

export const bffClient = new BFFClient(BFF_URL);
```

### WebSocket Hook (Client Component)
```typescript
// src/hooks/useWebSocket.ts
import { useEffect, useRef } from 'react';
import { useAuthStore } from '@/stores/authStore';

interface SubmissionProgress {
  submissionId: string;
  status: 'pending' | 'judging' | 'completed' | 'error';
  progress: number;
  verdict?: string;
  time?: number;
  memory?: number;
}

export function useWebSocket() {
  const wsRef = useRef<WebSocket | null>(null);
  const { token } = useAuthStore();

  useEffect(() => {
    if (!token) return;

    const wsUrl = `${BFF_WS_URL}/ws?token=${token}`;
    wsRef.current = new WebSocket(wsUrl);

    wsRef.current.onopen = () => {
      console.log('WebSocket connected');
    };

    wsRef.current.onerror = (err) => {
      console.error('WebSocket error:', err);
    };

    return () => {
      wsRef.current?.close();
    };
  }, [token]);

  const subscribeToSubmission = (
    submissionId: string,
    onUpdate: (progress: SubmissionProgress) => void
  ) => {
    if (!wsRef.current) return;

    // Send subscription message
    wsRef.current.send(JSON.stringify({
      type: 'subscribe',
      channel: `submission:${submissionId}`,
    }));

    // Handle incoming messages
    wsRef.current.onmessage = (event) => {
      const data = JSON.parse(event.data);
      if (data.channel === `submission:${submissionId}`) {
        onUpdate(data.payload);
      }
    };
  };

  return { subscribeToSubmission };
}
```

### Code Editor Component (Client Component)
```typescript
// src/components/editor/CodeEditor.tsx
'use client';

import { useState } from 'react';
import Editor from '@monaco-editor/react';
import { useWebSocket } from '@/hooks/useWebSocket';
import { useAuthStore } from '@/stores/authStore';
import { bffClient } from '@/lib/bff-client';
import { SubmissionStatus } from '@/components/submission/SubmissionStatus';

interface CodeEditorProps {
  problemId: string;
  language: string;
  timeLimit: number;
  memoryLimit: number;
}

export function CodeEditor({ problemId, language, timeLimit, memoryLimit }: CodeEditorProps) {
  const [code, setCode] = useState('');
  const [currentLang, setCurrentLang] = useState(language);
  const [submissionId, setSubmissionId] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const { subscribeToSubmission } = useWebSocket();
  const { token } = useAuthStore();

  const handleSubmit = async () => {
    if (!token) {
      alert('Please login to submit');
      return;
    }

    setIsSubmitting(true);
    try {
      const submission = await bffClient.createSubmission({
        problemId,
        language: currentLang,
        sourceCode: code,
      });
      
      setSubmissionId(submission.id);
      
      // Subscribe to real-time updates
      subscribeToSubmission(submission.id, (progress) => {
        if (progress.status === 'completed') {
          setIsSubmitting(false);
        }
      });
    } catch (err) {
      setIsSubmitting(false);
      alert('Submission failed');
    }
  };

  return (
    <div className="flex flex-col h-full">
      <div className="flex items-center justify-between p-2 border-b">
        <LanguageSelector 
          value={currentLang}
          onChange={setCurrentLang}
        />
        <div className="flex gap-2">
          <span className="text-sm text-muted-foreground">
            Time: {timeLimit}ms | Memory: {memoryLimit}KB
          </span>
          <button 
            onClick={handleSubmit}
            disabled={isSubmitting}
            className="btn btn-primary"
          >
            {isSubmitting ? 'Submitting...' : 'Submit'}
          </button>
        </div>
      </div>
      
      <Editor
        height="400px"
        language={currentLang}
        theme="vs-dark"
        value={code}
        onChange={setCode}
        options={{
          fontSize: 14,
          minimap: { enabled: false },
          automaticLayout: true,
        }}
      />
      
      {submissionId && (
        <SubmissionStatus submissionId={submissionId} />
      )}
    </div>
  );
}
```

### Auth Middleware (Next.js)
```typescript
// middleware.ts
import { NextRequest, NextResponse } from 'next/server';

const protectedRoutes = ['/profile', '/submit', '/submissions'];
const authRoutes = ['/login', '/register'];

export function middleware(request: NextRequest) {
  const token = request.cookies.get('auth_token')?.value;
  const pathname = request.nextUrl.pathname;

  // Redirect to login if accessing protected route without token
  if (protectedRoutes.some(route => pathname.startsWith(route))) {
    if (!token) {
      return NextResponse.redirect(new URL('/login', request.url));
    }
  }

  // Redirect to home if accessing auth routes with token
  if (authRoutes.some(route => pathname.startsWith(route))) {
    if (token) {
      return NextResponse.redirect(new URL('/', request.url));
    }
  }

  return NextResponse.next();
}

export const config = {
  matcher: ['/profile/:path*', '/submit/:path*', '/submissions/:path*', '/login', '/register'],
};
```

## Supported Languages
```typescript
// src/lib/languages.ts
export const SUPPORTED_LANGUAGES = [
  { id: 'cpp', name: 'C++ 17', monacoId: 'cpp', extension: '.cpp' },
  { id: 'python', name: 'Python 3', monacoId: 'python', extension: '.py' },
  { id: 'java', name: 'Java 17', monacoId: 'java', extension: '.java' },
  { id: 'go', name: 'Go 1.21', monacoId: 'go', extension: '.go' },
  { id: 'rust', name: 'Rust 1.70', monacoId: 'rust', extension: '.rs' },
  { id: 'javascript', name: 'Node.js 18', monacoId: 'javascript', extension: '.js' },
];
```

## Verdict Display
```typescript
// Verdict color mapping for UI
export const VERDICT_CONFIG: Record<Verdict, { color: string; label: string }> = {
  AC: { color: 'bg-green-500', label: 'Accepted' },
  WA: { color: 'bg-red-500', label: 'Wrong Answer' },
  TLE: { color: 'bg-yellow-500', label: 'Time Limit Exceeded' },
  MLE: { color: 'bg-orange-500', label: 'Memory Limit Exceeded' },
  RE: { color: 'bg-purple-500', label: 'Runtime Error' },
  CE: { color: 'bg-blue-500', label: 'Compilation Error' },
  PE: { color: 'bg-pink-500', label: 'Presentation Error' },
  SE: { color: 'bg-gray-500', label: 'System Error' },
};
```

## Data Caching Strategy

### Next.js Cache (Server)
```typescript
// Revalidate problem cache on update
export async function revalidateProblem(problemId: string) {
  await fetch(`/api/revalidate/problem?id=${problemId}`, { method: 'POST' });
}

// Cache tags for targeted revalidation
export const CACHE_TAGS = {
  problems: 'problems',
  problem: (id: string) => `problem:${id}`,
  contests: 'contests',
  contest: (id: string) => `contest:${id}`,
};
```

### Redis Cache (BFF)
```go
// Cache problem details with TTL
func (c *Cache) SetProblem(problemID string, problem *ProblemResponse) error {
    key := fmt.Sprintf("bff:problem:%s", problemID)
    data, _ := json.Marshal(problem)
    return c.client.Set(ctx, key, data, 5*time.Minute).Err()
}

// Cache contest rankings with short TTL
func (c *Cache) SetRankings(contestID string, rankings *RankingsResponse) error {
    key := fmt.Sprintf("bff:contest:%s:rankings", contestID)
    data, _ := json.Marshal(rankings)
    return c.client.Set(ctx, key, data, 10*time.Second).Err()
}
```

## Authentication Flow (Identra)

We use [Identra](https://github.com/poly-workshop/identra) for authentication, supporting multiple auth methods.

### Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                         Next.js Frontend                         │
│  - Login/Register pages                                          │
│  - OAuth redirect handling                                       │
└─────────────────────────────┬───────────────────────────────────┘
                              │
                              │ HTTP
                              │
┌─────────────────────────────▼───────────────────────────────────┐
│                         Go BFF Layer                             │
│  - Proxy auth requests to Identra                               │
│  - Token validation via JWKS                                    │
└─────────────────────────────┬───────────────────────────────────┘
                              │ gRPC/HTTP
                              │
┌─────────────────────────────▼───────────────────────────────────┐
│                    Identra Auth Service                          │
│  - OAuth (GitHub)                                               │
│  - Email Code                                                   │
│  - Password Authentication                                      │
│  - JWT Token Generation                                         │
│  - JWKS Endpoint                                                │
└─────────────────────────────────────────────────────────────────┘
```

### Auth Flow

```
1. Password Login
   User → Next.js → BFF /api/v1/auth/login → Identra gRPC
   Identra validates credentials → Returns JWT → BFF returns token
   
2. OAuth Login (GitHub)
   User → GET /api/v1/auth/oauth/url → Identra returns GitHub URL
   User redirects to GitHub → Authorizes → Callback to frontend
   Frontend → POST /api/v1/auth/oauth/callback → Identra exchanges code
   Identra returns JWT → BFF returns token
   
3. Token Validation
   Request with Bearer token → BFF validates via Identra JWKS
   JWKS endpoint: /.well-known/jwks.json
   Extract user_id from JWT claims → Forward to backend services
   
4. Token Refresh
   Access token expires → Frontend sends refresh_token
   POST /api/v1/auth/refresh → Identra returns new access_token
```

### Frontend Auth Implementation

```typescript
// src/actions/auth.ts
'use server';

import { bffClient } from '@/lib/bff-client';
import { cookies } from 'next/headers';

export async function login(email: string, password: string) {
  const response = await bffClient.login(email, password);
  
  // Set tokens in cookies
  cookies().set('access_token', response.access_token, {
    httpOnly: true,
    secure: true,
    maxAge: response.expires_in,
  });
  
  cookies().set('refresh_token', response.refresh_token, {
    httpOnly: true,
    secure: true,
    maxAge: 7 * 24 * 60 * 60, // 7 days
  });
  
  return { success: true };
}

export async function oauthLogin(code: string, state: string) {
  const response = await bffClient.oauthCallback(code, state);
  
  cookies().set('access_token', response.access_token, {
    httpOnly: true,
    secure: true,
    maxAge: response.expires_in,
  });
  
  cookies().set('refresh_token', response.refresh_token, {
    httpOnly: true,
    secure: true,
    maxAge: 7 * 24 * 60 * 60,
  });
  
  return { success: true };
}

export async function logout() {
  cookies().delete('access_token');
  cookies().delete('refresh_token');
}
```

### BFF Client Auth Methods

```typescript
// src/lib/bff-client.ts (auth methods)
class BFFClient {
  async login(email: string, password: string): Promise<AuthResponse> {
    return this.fetch<AuthResponse>('/api/v1/auth/login', {
      method: 'POST',
      body: JSON.stringify({ email, password }),
    });
  }

  async getOAuthUrl(provider: string = 'github'): Promise<{ url: string; state: string }> {
    return this.fetch(`/api/v1/auth/oauth/url?provider=${provider}`);
  }

  async oauthCallback(code: string, state: string): Promise<AuthResponse> {
    return this.fetch<AuthResponse>('/api/v1/auth/oauth/callback', {
      method: 'POST',
      body: JSON.stringify({ code, state }),
    });
  }

  async refreshToken(refreshToken: string): Promise<{ access_token: string; expires_in: number }> {
    return this.fetch('/api/v1/auth/refresh', {
      method: 'POST',
      body: JSON.stringify({ refresh_token: refreshToken }),
    });
  }

  async getMe(): Promise<UserInfo> {
    return this.fetch<UserInfo>('/api/v1/auth/me');
  }
}

interface AuthResponse {
  access_token: string;
  refresh_token: string;
  expires_in: number;
}

interface UserInfo {
  user_id: string;
  email: string;
}
```

### OAuth Login Page

```typescript
// src/app/login/page.tsx
'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { login, oauthLogin } from '@/actions/auth';
import { bffClient } from '@/lib/bff-client';

export default function LoginPage() {
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const router = useRouter();

  const handlePasswordLogin = async (e: React.FormEvent) => {
    e.preventDefault();
    await login(email, password);
    router.push('/');
  };

  const handleGitHubOAuth = async () => {
    const { url, state } = await bffClient.getOAuthUrl('github');
    // Store state for callback verification
    localStorage.setItem('oauth_state', state);
    window.location.href = url;
  };

  return (
    <div className="flex flex-col gap-4 max-w-md mx-auto">
      <form onSubmit={handlePasswordLogin}>
        <input
          type="email"
          value={email}
          onChange={setEmail}
          placeholder="Email"
        />
        <input
          type="password"
          value={password}
          onChange={setPassword}
          placeholder="Password"
        />
        <button type="submit">Login</button>
      </form>
      
      <button onClick={handleGitHubOAuth}>
        Login with GitHub
      </button>
    </div>
  );
}
```

### OAuth Callback Handler

```typescript
// src/app/oauth/callback/page.tsx
'use client';

import { useEffect } from 'react';
import { useRouter, useSearchParams } from 'next/navigation';
import { oauthLogin } from '@/actions/auth';

export default function OAuthCallbackPage() {
  const searchParams = useSearchParams();
  const router = useRouter();

  useEffect(() => {
    const code = searchParams.get('code');
    const state = searchParams.get('state');
    const storedState = localStorage.getItem('oauth_state');

    if (code && state && state === storedState) {
      oauthLogin(code, state).then(() => {
        localStorage.removeItem('oauth_state');
        router.push('/');
      });
    } else {
      router.push('/login?error=oauth_failed');
    }
  }, [searchParams, router]);

  return <div>Processing OAuth login...</div>;
}
```

## Performance Optimizations

1. **SSR for SEO**: Problem pages rendered server-side for search indexing
2. **BFF Aggregation**: Single request fetches multiple backend resources
3. **Redis Caching**: BFF caches frequently accessed data
4. **Streaming**: Use Next.js streaming for slow operations
5. **Connection Pooling**: BFF maintains gRPC connection pool to services

## Next Steps

1. Initialize Next.js project with App Router
2. Set up Tailwind CSS and shadcn/ui
3. Create Go BFF project structure
4. Implement BFF authentication endpoints
5. Build gRPC clients for backend services
6. Create WebSocket manager in BFF
7. Build frontend pages with SSR
8. Implement code editor with Monaco
9. Add real-time submission status tracking