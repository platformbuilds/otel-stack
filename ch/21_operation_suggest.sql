CREATE TABLE IF NOT EXISTS default.operation_suggest
(
  WindowStart DateTime,
  SpanName LowCardinality(String),
  Cnt UInt64
)
ENGINE = SummingMergeTree()
ORDER BY (WindowStart, SpanName);

CREATE MATERIALIZED VIEW IF NOT EXISTS default.mv_operation_suggest
TO default.operation_suggest AS
SELECT
  toStartOfHour(Timestamp) AS WindowStart,
  SpanName,
  count() AS Cnt
FROM default.otel_traces
GROUP BY WindowStart, SpanName;
