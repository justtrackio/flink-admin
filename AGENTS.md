# Flink Admin - Agent Developer Guide

A web-based administration interface for Apache Flink clusters with Kubernetes integration, built with Go (gosoline) backend and React 19 + TypeScript frontend.

## Build, Lint & Test Commands

### Backend (Go)

**No tests currently exist.** The project uses standard Go tooling:

```bash
# Development
cd backend
go mod tidy                    # Install/update dependencies
go run .                       # Run dev server (port 8082)
go fmt ./...                   # Format code
go vet ./...                   # Static analysis

# Production build
go build -o flink-admin .

# Running single test (when tests exist)
go test -v -run TestFunctionName ./path/to/package
```

**Go Version:** 1.25.0  
**Framework:** gosoline v0.54.8 (github.com/justtrackio/gosoline)

### Frontend (TypeScript/React)

**No tests currently exist.** Uses Bun as package manager:

```bash
# Development
cd frontend
bun install                    # Install dependencies
bun run dev                    # Start Vite dev server (port 5173)
bun run lint                   # Run ESLint
bun run build                  # Production build (vite build && tsc -b)
bun run preview                # Preview production build

# Running single test (when tests exist)
# Add vitest or jest first, then:
# bun test path/to/file.test.ts
```

**Build Tool:** Vite 7.2.4  
**TypeScript:** 5.9.3 (strict mode enabled)  
**Linter:** ESLint 9.39.1 with flat config

---

## Code Style Guidelines

### Backend (Go)

#### Naming Conventions
- **Files:** `snake_case` (e.g., `flink_client.go`, `deployment_watcher.go`)
- **Types:** `PascalCase` (e.g., `FlinkClient`, `DeploymentWatcher`)
- **Functions:** `PascalCase` for exported, `camelCase` for unexported
- **Constants:** `PascalCase` for exported, `camelCase` for unexported
- **Struct tags:** `json:"fieldName,omitempty"`

#### Architecture Patterns

**1. Provider Pattern with appctx**
```go
func ProvideFlinkClient(ctx context.Context, config cfg.Config, logger log.Logger) (*FlinkClient, error) {
    return appctx.Provide(ctx, flinkCtxKey{}, func() (*FlinkClient, error) {
        // Singleton initialization logic
        return &FlinkClient{...}, nil
    })
}
```

**2. Handler Constructor Pattern**
```go
func NewHandlerCheckpoints(ctx context.Context, config cfg.Config, logger log.Logger) (*HandlerCheckpoints, error) {
    client, err := ProvideFlinkClient(ctx, config, logger)
    if err != nil {
        return nil, fmt.Errorf("could not create flink client: %w", err)
    }
    
    return &HandlerCheckpoints{
        logger: logger.WithChannel("handler_checkpoints"),
        client: client,
    }, nil
}
```

**3. HTTP Handlers with httpserver.With**
```go
func (h *HandlerCheckpoints) GetRoutes() []httpserver.RouteDefinition {
    return []httpserver.RouteDefinition{
        httpserver.Get("/checkpoints/:namespace/:name", httpserver.With(h.GetCheckpoints)),
    }
}
```

#### Error Handling
- **Always wrap errors** with context using `fmt.Errorf("description: %w", err)`
- **Early returns** for error conditions
- **Explicit checks:** Always check `if err != nil` immediately
- **No panic:** Avoid panic in production code
- **Constructor errors:** Return `(Type, error)` from all constructors

```go
// Good
if err != nil {
    return nil, fmt.Errorf("failed to connect to kubernetes: %w", err)
}

// Bad - no context
if err != nil {
    return nil, err
}
```

#### Import Organization
```go
import (
    // Standard library first
    "context"
    "fmt"
    "time"
    
    // External dependencies second (alphabetical)
    "github.com/justtrackio/gosoline/pkg/cfg"
    "github.com/justtrackio/gosoline/pkg/log"
    
    // Aliased imports last
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)
```

#### Struct Patterns
```go
type FlinkDeployment struct {
    metav1.TypeMeta   `json:",inline"`      // Embedded types
    metav1.ObjectMeta `json:"metadata"`     // Standard K8s fields
    
    Spec   FlinkDeploymentSpec   `json:"spec"`             // Always pointer receivers
    Status FlinkDeploymentStatus `json:"status,omitempty"` // Use omitempty for optional
}
```

### Frontend (TypeScript/React)

#### Naming Conventions
- **Files:** 
  - Components: `PascalCase.tsx` (e.g., `JobStatusTag.tsx`)
  - Other: `camelCase.ts` (e.g., `format.ts`, `client.ts`)
- **Components:** `PascalCase` function declarations
- **Hooks:** `camelCase` starting with `use` (e.g., `useDeploymentStream`)
- **Interfaces/Types:** `PascalCase` (e.g., `FlinkDeployment`, `ApiError`)
- **Constants:** `UPPER_SNAKE_CASE` (e.g., `API_BASE_URL`, `MIN_BACKOFF_MS`)
- **Variables/Functions:** `camelCase`

#### Component Pattern
```typescript
// Always define prop interface
interface JobStatusTagProps {
  status?: string;
}

// Named function export (not arrow function)
export function JobStatusTag({ status }: JobStatusTagProps) {
  const upperStatus = status?.toUpperCase();
  
  return <Tag color={getColor(upperStatus)}>{upperStatus || 'N/A'}</Tag>;
}
```

#### Custom Hooks Pattern
```typescript
export function useDeploymentStream(): DeploymentStreamState {
  const [deployments, setDeployments] = useState<FlinkDeployment[]>([]);
  const [isConnected, setIsConnected] = useState(false);
  const [error, setError] = useState<string | null>(null);
  
  useEffect(() => {
    // Side effects with cleanup
    return () => {
      // Cleanup logic
    };
  }, [dependencies]);
  
  return { deployments, isConnected, error, retry };
}
```

#### API Client Pattern
```typescript
export class ApiClient {
  private baseUrl: string;
  
  constructor(baseUrl?: string) {
    this.baseUrl = baseUrl || '';
  }
  
  async get<T>(endpoint: string): Promise<T> {
    return this.request<T>(endpoint, { method: 'GET' });
  }
  
  private async request<T>(endpoint: string, options?: RequestInit): Promise<T> {
    const response = await fetch(`${this.baseUrl}${endpoint}`, options);
    
    if (!response.ok) {
      throw { message: errorMessage, status: response.status } as ApiError;
    }
    
    return response.json() as Promise<T>;
  }
}

// Singleton export
export const apiClient = new ApiClient();
```

#### Error Handling
- **Type guards:** `if (err instanceof Error)` or `typeof err === 'object'`
- **Try-catch blocks** for async operations
- **Optional chaining** for nested properties: `deployment.status?.lifecycleState`
- **Nullish coalescing** for defaults: `status ?? 'N/A'`
- **Typed errors:** Define `ApiError` interface with `message` and `status`

```typescript
// Good
try {
  const data = await apiClient.get<FlinkDeployment>(url);
} catch (err) {
  if (err && typeof err === 'object' && 'status' in err) {
    const apiError = err as ApiError;
    setError(`Failed to fetch: ${apiError.message}`);
  }
}

// Good - optional chaining
const state = deployment.status?.lifecycleState ?? 'UNKNOWN';
```

#### Import Organization
```typescript
// React and external libraries first
import { createFileRoute, Link } from '@tanstack/react-router';
import { Alert, Button, Card } from 'antd';
import type { ColumnsType } from 'antd/es/table/interface';

// Relative imports second (grouped by type)
import { useDeploymentStreamContext } from '../context/DeploymentStreamContext';
import { JobStatusTag } from '../components/JobStatusTag';
import type { FlinkDeployment } from '../api/schema';

// IMPORTANT: Use 'type' keyword for type-only imports
import type { FlinkDeployment, ApiError } from '../api/schema';
```

#### TypeScript Types
- **Prefer `interface`** for object shapes
- **Use `type`** for unions, intersections, and primitives
- **Explicit return types** on hooks, API methods, and exported functions
- **Readonly arrays** when appropriate: `ReadonlyArray<T>` or `readonly T[]`
- **Avoid `any`** - use `unknown` and type guards instead

```typescript
// Good - interface for objects
interface FlinkDeployment {
  metadata: ObjectMeta;
  spec: FlinkDeploymentSpec;
  status?: FlinkDeploymentStatus;
}

// Good - type for unions
type DeploymentEventType = 'ADDED' | 'MODIFIED' | 'DELETED';

// Good - explicit return type
export function useDeploymentStream(): DeploymentStreamState {
  // ...
}
```

#### React Patterns
- **Functional components only** (no class components)
- **Hook-based state management** (useState, useEffect, useContext)
- **Context for global state** (e.g., `DeploymentStreamContext`)
- **TanStack Query** for server state caching
- **File-based routing** with TanStack Router
- **useMemo** for expensive computations
- **useCallback** for stable function references in dependencies

---

## Project Structure

```
flink-admin/
├── backend/                      # Go backend
│   ├── config.dist.yml          # Config template with K8s settings
│   ├── main.go                  # Application entry point
│   ├── flink_client.go          # Flink REST API client
│   ├── k8s_service.go           # Kubernetes client wrapper
│   ├── deployment_watcher.go    # FlinkDeployment CRD watcher
│   ├── handler_deployments.go   # SSE streaming endpoint
│   ├── handler_checkpoints.go   # Checkpoint statistics
│   ├── flink_k8s_types.go       # FlinkDeployment CRD types
│   └── flink_api_types.go       # Flink REST API types
└── frontend/                     # React + TypeScript frontend
    ├── src/
    │   ├── api/                 # API client layer
    │   │   ├── client.ts        # HTTP client
    │   │   └── schema.ts        # TypeScript types
    │   ├── components/          # Reusable React components
    │   ├── context/             # React context providers
    │   ├── hooks/               # Custom React hooks (SSE, data fetching)
    │   ├── routes/              # File-based routing
    │   └── utils/               # Utility functions
    ├── eslint.config.js         # ESLint flat config
    ├── tsconfig.json            # TypeScript config (strict mode)
    └── vite.config.ts           # Vite with dev proxy to :8082
```

---

## Key Technologies

**Backend:**
- gosoline v0.54.8 - Application framework with modules/providers
- httpserver v0.1.1 - HTTP server with handler registration
- k8s.io/client-go v0.35.0 - Kubernetes client
- gin-contrib/cors v1.7.3 - CORS middleware

**Frontend:**
- React 19.2.0 + TypeScript 5.9.3
- TanStack Router 1.136.18 (file-based routing)
- TanStack Query 5.90.10 (server state)
- Ant Design 5.29.1 (UI components)
- Vite 7.2.4 (build tool)

---

## Configuration

**Backend (`backend/config.dist.yml`):**
```yaml
kubernetes:
  namespace: flink-deployments  # K8s namespace to watch
```

**Frontend (`frontend/.env`):**
```
VITE_API_BASE_URL=http://localhost:8082  # Optional, defaults to same-origin
```

**Dev Proxy:** Frontend Vite proxies `/api` requests to `http://localhost:8082` (configured in `vite.config.ts`)

---

## Important Notes

1. **No Database:** Pure API proxy pattern - backend calls Flink REST API and watches K8s CRDs directly
2. **SSE Streaming:** Backend streams FlinkDeployment CRD events via Server-Sent Events
3. **Embedded Frontend:** Production builds embed frontend in Go binary via `//go:embed public`
4. **No Tests:** Testing infrastructure not yet established - consider adding vitest (frontend) and standard go test (backend)
5. **Module Pattern:** Backend uses gosoline's module factory pattern for lifecycle management
6. **Custom SSE Parser:** Frontend implements manual SSE parsing instead of EventSource API for better control
