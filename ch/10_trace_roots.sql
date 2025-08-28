-- target table for the MV
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

-- materialized view that populates trace_roots
CREATE MATERIALIZED VIEW default.mv_trace_roots
TO default.trace_roots
AS
WITH
/* Per-trace “root” fields (root span has empty ParentSpanId) */
root AS
(
  SELECT
    TraceId,
    anyIf(ServiceName, ParentSpanId = '')  AS RootService,
    anyIf(SpanName,   ParentSpanId = '')   AS RootOperation
  FROM default.otel_traces
  GROUP BY TraceId
),
/* Per-trace per-service total duration (do the sum in its own query!) */
svc_agg AS
(
  SELECT
    TraceId,
    ServiceName,
    sum(Duration) AS svc_dur           -- NOTE: see “Units” note below
  FROM default.otel_traces
  GROUP BY TraceId, ServiceName
),
/* For each trace: array of (service, svc_dur) sorted descending by svc_dur */
svc_rank AS
(
  SELECT
    TraceId,
    arraySort(x -> -x.2, groupArray((ServiceName, svc_dur))) AS svc_sorted
  FROM svc_agg
  GROUP BY TraceId
)
SELECT
  t.TraceId,
  min(t.Timestamp)                 AS StartTs,
  /* Total trace duration (sum of span durations; same unit as `Duration`) */
  sum(t.Duration)                  AS DurationMs,
  r.RootService,
  r.RootOperation,
  any(t.StatusCode)                AS Status,
  uniqExact(t.SpanId)              AS SpanCount,
  if(length(sv.svc_sorted)>0, sv.svc_sorted[1].1, r.RootService)       AS TopService,
  if(length(sv.svc_sorted)>0, sv.svc_sorted[1].2, 0)                   AS TopServiceMs
FROM default.otel_traces AS t
LEFT JOIN root    AS r  USING (TraceId)
LEFT JOIN svc_rank AS sv USING (TraceId)
GROUP BY
  t.TraceId, r.RootService, r.RootOperation, sv.svc_sorted;