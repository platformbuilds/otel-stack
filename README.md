# OTEL Stack Demo — Backend (Go) + React UI (visx timeline + d3-flame-graph)

This bundle includes:
- **server/** — production-ready Go API (PromQL, LogsQL, Traces) with human-friendly trace discovery.
- **ui/** — tiny React demo (drag-and-drop-ready) using **visx** (timeline) and **d3-flame-graph**.
- **ch/** — ClickHouse SQL for **Materialized Views** to accelerate list/suggest queries.

## Quick Start
```bash
# 1) Backend
cd server
cp ../.env.example .env
docker build -t otel-backend .
docker run --rm -it --net=host --env-file .env otel-backend

# or with compose from repo root (if you prefer):
# docker compose up --build

# 2) UI
cd ../ui
npm install
npm run dev -- --host 0.0.0.0
# UI at http://localhost:5173 (uses VITE_API_BASE from package.json proxy or .env)
```

### Configure datasources in `.env`
```
HTTP_ADDR=:8080
PROM_URL=http://localhost:9090
VLOGS_URL=http://localhost:9428
CH_HTTP_URL=http://localhost:8123
CH_USER=default
CH_PASS=
CH_DATABASE=default
DEMO_MODE=true
DEFAULT_ROLE=editor
CORS_ALLOW_ORIGINS=*
```

---

## ClickHouse Materialized Views (speed boost)
Apply these SQL files (adjust DB name if not `default`).

```bash
clickhouse-client -n < ch/10_trace_roots.sql
clickhouse-client -n < ch/20_service_suggest.sql
clickhouse-client -n < ch/21_operation_suggest.sql
clickhouse-client -n < ch/30_attr_values.sql
```

**What they do:**
- `trace_roots` : One row per trace with start time, total duration, root service/op, span count, top service by time.
- `service_suggest` : Hourly counts of services for fast suggestions/autocomplete.
- `operation_suggest` : Hourly counts of operations.
- `attr_values` : Hourly counts of selected attribute values (e.g., `http.method`, `deployment.environment`, `db.system`, `http.route`).

If your exported OTel schema stores attributes differently (Map vs JSON string), swap `JSON_VALUE` for `JSONExtractString` or relevant functions.

---

## Endpoints (high level)
- `POST /api/metrics/query` → PromQL `/api/v1/query_range`
- `POST /api/logs/search` → VictoriaLogs LogsQL (`/select/logsql/query`)
- `POST /api/traces/list` → list traces (uses `trace_roots` MV if available)
- `GET  /api/traces/{traceId}` → Gantt-friendly spans
- `GET  /api/traces/{traceId}/flame` → `{name,value,children[]}` (μs) for **d3-flame-graph**
- `GET  /api/traces/suggest/services|operations|attributes` → fast suggestions (uses MVs)
- `POST /api/traces/search`, `GET /api/traces/recent`, human handles (`/api/handles`, `/api/traces/handle/{handle}`)

---

## UI Demo
- **Finder**: service/op facets + list (calls `/api/traces/list`).
- **Trace View**:
  - **Timeline** tab: visx Gantt (service lanes) using `/api/traces/{id}`.
  - **Flame** tab: d3-flame-graph using `/api/traces/{id}/flame`.

This UI is small and meant as a starter for your drag-and-drop builder.
