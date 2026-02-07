# Flink Admin

A web-based administration interface for Apache Flink clusters, built with the same architecture and patterns as lakehouse-admin.

## Architecture

### Backend
- **Framework:** Go with gosoline (justtrackio/gosoline)
- **HTTP Server:** gosoline-project/httpserver
- **Structure:** Handlers → Services → Flink REST API Client

### Frontend
- **Framework:** React 19 + TypeScript
- **Routing:** TanStack Router
- **State Management:** TanStack Query
- **UI Library:** Ant Design
- **Build Tool:** Vite

## Features

### Current (v1.0 - Read-only + Basic Actions)
- ✅ Multi-cluster management
- ✅ Cluster overview (slots, task managers, job counts)
- ✅ Job listing with status filtering
- ✅ Job cancellation
- ✅ Real-time metrics refresh

### Future Enhancements
- Job deployment (JAR upload)
- Savepoint management (create, restore)
- Job history tracking (with MySQL database)
- Background health monitoring module
- Job configuration viewer
- Log aggregation

## Directory Structure

```
flink-admin/
├── backend/
│   ├── config.dist.yml          # Configuration with cluster definitions
│   ├── main.go                  # Application entry point
│   ├── types.go                 # Domain types
│   ├── flink_client.go          # Flink REST API client
│   ├── service_flink.go         # Business logic layer
│   ├── handler_clusters.go      # Cluster endpoints
│   ├── handler_jobs.go          # Job endpoints
│   └── handler_overview.go      # Overview endpoints
└── frontend/
    ├── src/
    │   ├── api/
    │   │   ├── client.ts        # HTTP client
    │   │   └── schema.ts        # API types & functions
    │   ├── components/
    │   │   ├── JobStatusTag.tsx
    │   │   └── MessageProvider.tsx
    │   ├── routes/
    │   │   ├── __root.tsx
    │   │   ├── index.tsx
    │   │   ├── clusters.$clusterName.tsx
    │   │   ├── clusters.$clusterName.jobs.tsx
    │   │   └── clusters.$clusterName.overview.tsx
    │   └── utils/
    │       └── format.ts
    └── package.json
```

## Configuration

### Backend (`backend/config.dist.yml`)

```yaml
flink:
  clusters:
    production:
      url: http://flink-production:8081
      name: Production
    staging:
      url: http://flink-staging:8081
      name: Staging
```

### Frontend (`frontend/.env`)

```
VITE_API_BASE_URL=http://localhost:8082  # Optional, defaults to same-origin
```

## Development

### Backend

```bash
cd backend
go mod tidy
go run .
```

The backend will start on port 8082 (configurable in config.dist.yml).

### Frontend

```bash
cd frontend
bun install
bun run dev
```

The frontend dev server will start on port 5173 with proxy to backend on 8082.

## Building for Production

### Backend

```bash
cd backend
go build -o flink-admin .
```

### Frontend

```bash
cd frontend
bun run build
# Copy dist/ contents to backend/public/ for embedding
cp -r dist/* ../backend/public/
```

The backend embeds the frontend build, serving it at the root path.

## API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/clusters` | List all configured clusters |
| GET | `/api/clusters/:cluster` | Get cluster info (overview + config) |
| GET | `/api/clusters/:cluster/overview` | Get cluster overview |
| GET | `/api/clusters/:cluster/jobs` | List jobs for cluster |
| GET | `/api/clusters/:cluster/jobs/:jobId` | Get job details |
| DELETE | `/api/clusters/:cluster/jobs/:jobId` | Cancel a running job |

## Design Patterns

This project mirrors lakehouse-admin's architecture:

1. **Handler Pattern:** HTTP handlers with constructor injection
2. **Service Layer:** Business logic separated from HTTP concerns  
3. **Client Pattern:** External API client (Flink REST API) via appctx provider
4. **No Database (Yet):** Direct API proxy pattern, database can be added later for history
5. **Frontend Routing:** File-based routing with TanStack Router
6. **State Management:** Server state via TanStack Query, minimal local state

## Testing

Connect to a Flink cluster by updating `backend/config.dist.yml` with your cluster URL, then start both backend and frontend.

Example with local Flink:
```yaml
flink:
  clusters:
    local:
      url: http://localhost:8081
      name: Local
```

## License

Same as lakehouse-admin.
