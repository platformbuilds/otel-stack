CREATE TABLE IF NOT EXISTS default.trace_roots
(
  TraceId String,
  StartTs DateTime,
  DurationMs Float64,
  RootService LowCardinality(String),
  RootOperation LowCardinality(String),
  Status LowCardinality(String),
  SpanCount UInt32,
  TopService LowCardinality(String),
  TopServiceMs Float64
)
ENGINE = ReplacingMergeTree()
ORDER BY (StartTs, TraceId);

CREATE MATERIALIZED VIEW IF NOT EXISTS default.mv_trace_roots
TO default.trace_roots AS
WITH
  groupArray((ServiceName, sum(Duration))) AS svc_pairs,
  arraySort(x -> -x.2, svc_pairs) AS svc_sorted
SELECT
  TraceId,
  min(Timestamp) AS StartTs,
  sum(Duration)/1e6 AS DurationMs,
  anyIf(ServiceName, ParentSpanId = '') AS RootService,
  anyIf(SpanName, ParentSpanId = '') AS RootOperation,
  any(StatusCode) AS Status,
  uniqExact(SpanId) AS SpanCount,
  if(length(svc_sorted)>0, svc_sorted[1].1, any(ServiceName)) AS TopService,
  if(length(svc_sorted)>0, svc_sorted[1].2/1e6, 0) AS TopServiceMs
FROM default.otel_traces
GROUP BY TraceId;
