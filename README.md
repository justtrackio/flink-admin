# Flink Admin

A web-based administration interface for Apache Flink clusters running on Kubernetes. Provides real-time monitoring of FlinkDeployment CRDs, checkpoint statistics from the Flink REST API, and S3 storage browsing for checkpoints and savepoints.

## Features

- **Real-time deployment monitoring** -- Watches FlinkDeployment CRDs across multiple Kubernetes namespaces via SSE streaming
- **Deployment dashboard** -- Tabular overview with lifecycle state, job state, Flink version, parallelism, JM/TM resources, and age
- **Filtering and sorting** -- URL-persisted filters by namespace and lifecycle state; toggle to show only non-running jobs
- **Deployment detail view** -- Per-deployment metadata, spec (image, entry class, JAR URI, upgrade mode, job args), resource allocations, and status
- **Checkpoint statistics** -- Proxies the Flink REST API for checkpoint counts, history, durations, state sizes, and storage paths
- **S3 storage browser** -- Lists checkpoints and savepoints in S3, validates them by checking for `_metadata` files
- **Flink UI deep links** -- Direct links to the Flink web UI for deployments with active jobs
- **Embedded frontend** -- Production binary embeds the React frontend via `//go:embed`, producing a single self-contained binary

## Tech Stack

### Backend

| Technology | Version | Purpose |
|---|---|---|
| Go | 1.25.0 | Language |
| gosoline | v0.54.8 | Application framework (config, logging, lifecycle) |
| httpserver | v0.1.1 | HTTP server with handler registration |
| k8s.io/client-go | v0.35.0 | Kubernetes client |
| aws-sdk-go-v2/s3 | v1.61.2 | S3 client for storage browsing |

### Frontend

| Technology | Version | Purpose |
|---|---|---|
| React | 19.2.0 | UI framework |
| TypeScript | 5.9.3 | Type-safe JavaScript (strict mode) |
| Vite | 7.2.4 | Build tool and dev server |
| TanStack Router | 1.136.18 | File-based routing |
| TanStack Query | 5.90.10 | Server state caching |
| Ant Design | 5.29.1 | UI component library |

## Project Structure

```
flink-admin/
├── backend/
│   ├── main.go                        # Entry point, module/handler registration
│   ├── config.dist.yml                # Default configuration
│   ├── config.sandbox.yml             # Local dev configuration
│   ├── build/
│   │   ├── Dockerfile                 # Multi-stage production build
│   │   └── Dockerfile.release         # CI release image
│   └── internal/
│       ├── deployment_watcher.go      # K8s FlinkDeployment CRD watcher
│       ├── module_deployment_watcher.go # In-memory cache + SSE fan-out
│       ├── handler_deployments.go     # SSE streaming endpoint
│       ├── handler_checkpoints.go     # Checkpoint statistics endpoint
│       ├── handler_storage_checkpoints.go # S3 storage listing endpoint
│       ├── flink_client.go            # Flink REST API client
│       ├── k8s_service.go             # Kubernetes client wrapper
│       ├── s3_service.go              # S3 client for checkpoint storage
│       ├── flink_k8s_types.go         # FlinkDeployment CRD Go types
│       ├── flink_api_types.go         # Flink REST API response types
│       └── checkpoint/               # Flink _metadata binary parser
└── frontend/
    └── src/
        ├── api/                       # HTTP client and TypeScript types
        ├── components/                # Reusable UI components
        ├── context/                   # React context providers (SSE stream)
        ├── hooks/                     # Custom hooks (SSE, data fetching)
        ├── routes/                    # File-based route definitions
        └── utils/                     # Formatting helpers
```

## Getting Started

### Prerequisites

- go 1.24.0+
- [Bun](https://bun.sh/) (latest)
- Access to a Kubernetes cluster with FlinkDeployment CRDs
- AWS credentials for S3 storage features (optional)

Alternatively, use [mise](https://mise.jdx.dev/) to install all tool versions:

```bash
mise install
```

### Development

Start the backend and frontend in separate terminals:

```bash
# Terminal 1: Backend (port 8082)
cd backend
go mod tidy
go run . -c config.sandbox.yml
```

```bash
# Terminal 2: Frontend (port 5173)
cd frontend
bun install
bun run dev
```

The Vite dev server proxies `/api` requests to `http://localhost:8082`.

### Linting

```bash
# Backend
cd backend
golangci-lint run

# Frontend
cd frontend
bun run lint
```

### Production Build

#### Docker (recommended)

```bash
docker build -f backend/build/Dockerfile -t flink-admin .
```

#### Manual

```bash
# Build frontend
cd frontend
bun install --frozen-lockfile
bun run build

# Copy frontend assets into backend embed directory
cp -r dist ../backend/public

# Build backend (embeds frontend via //go:embed)
cd ../backend
CGO_ENABLED=0 GOOS=linux go build -o flink-admin .
```

## Configuration

### Backend

Configuration is loaded from YAML files. Key options in `config.dist.yml`:

| Key | Default | Description |
|---|---|---|
| `flink.namespaces` | `[...]` | Kubernetes namespaces to watch for FlinkDeployment CRDs |
| `httpserver.default.port` | `8082` | HTTP server port |
| `kube.client_mode` | `in-cluster` | Kubernetes client mode (`in-cluster` or `kube-config`) |
| `kube.context` | - | Kubernetes context (for `kube-config` mode) |
| `cloud.aws.s3.clients.default.region` | `eu-central-1` | AWS S3 region |

For local development, use `config.sandbox.yml` which switches to `kube-config` client mode.

### Frontend

| Variable | Default | Description |
|---|---|---|
| `VITE_API_BASE_URL` | `""` (same-origin) | Override API base URL |

## Architecture

```
  Kubernetes Cluster                          AWS S3
  (FlinkDeployment CRDs)                     (checkpoints/savepoints)
         |                                         |
    K8s Watch API                              AWS SDK
         |                                         |
  +------v-----------------------------------------v------+
  |                    Go Backend                          |
  |                                                        |
  |  DeploymentWatcher ──> In-Memory Cache ──> SSE Fan-Out |
  |  FlinkClient ──────────> Flink REST API Proxy          |
  |  S3Service ────────────> S3 Storage Listing            |
  +----------------------------+---------------------------+
                               |
                          SSE + REST
                               |
                    +----------v-----------+
                    |   React Frontend     |
                    |                      |
                    |  / ──── Dashboard    |
                    |  /deployments/:ns/:n |
                    |    ├── Details       |
                    |    ├── Checkpoints   |
                    |    └── Storage       |
                    +----------------------+
```

- **No database** -- pure API proxy pattern. The backend watches K8s CRDs and proxies Flink REST API calls directly.
- **SSE streaming** -- The frontend uses a custom SSE parser via `fetch` + `ReadableStream` for real-time deployment updates with heartbeat-based liveness detection and automatic reconnection.
- **Embedded frontend** -- The Go binary embeds the compiled frontend assets via `//go:embed`, serving everything from a single binary in production.

