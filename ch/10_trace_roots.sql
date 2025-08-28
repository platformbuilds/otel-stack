-- Target table that the MV writes into
CREATE TABLE default.trace_roots
(
  TraceId       String,
  StartTs       DateTime,
  DurationMs    Float64,
  RootService   String,
  RootOperation String,
  Status        String,
  SpanCount     UInt32,
  TopService    String,
  TopServiceMs  Float64
)
ENGINE = MergeTree
ORDER BY (TraceId);

-- Materialized view that populates trace_roots
CREATE MATERIALIZED VIEW default.mv_trace_roots
TO default.trace_roots
AS
WITH
  /* Root span info per trace (parent = '') */
  root AS
  (
    SELECT
      TraceId,
      anyIf(ServiceName, ParentSpanId = '') AS RootService,
      anyIf(SpanName,   ParentSpanId = '')  AS RootOperation
    FROM default.otel_traces
    GROUP BY TraceId
  ),
  /* Per-trace per-service total duration (sum done in its own stage) */
  svc_agg AS
  (
    SELECT
      TraceId,
      ServiceName,
      sum(Duration) AS svc_dur   -- adjust units below if needed
    FROM default.otel_traces
    GROUP BY TraceId, ServiceName
  ),
  /* For each trace: array of (service, svc_dur) sorted descending */
  svc_rank AS
  (
    SELECT
      TraceId,
      arraySort(x -> -x.2, groupArray((ServiceName, svc_dur))) AS svc_sorted
    FROM svc_agg
    GROUP BY TraceId
  )
SELECT
  t.TraceId                                 AS TraceId,
  min(t.Timestamp)                          AS StartTs,
  sum(t.Duration)                           AS DurationMs,     -- see units note
  r.RootService                             AS RootService,
  r.RootOperation                           AS RootOperation,
  any(t.StatusCode)                         AS Status,
  uniqExact(t.SpanId)                       AS SpanCount,
  if(length(sv.svc_sorted)>0, sv.svc_sorted[1].1, r.RootService) AS TopService,
  if(length(sv.svc_sorted)>0, sv.svc_sorted[1].2, 0)             AS TopServiceMs
FROM default.otel_traces AS t
LEFT JOIN root     AS r  USING (TraceId)
LEFT JOIN svc_rank AS sv USING (TraceId)
GROUP BY
  TraceId, RootService, RootOperation, svc_sorted;