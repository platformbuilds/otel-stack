CREATE TABLE IF NOT EXISTS default.service_suggest
(
  WindowStart DateTime,
  ServiceName LowCardinality(String),
  Cnt UInt64
)
ENGINE = SummingMergeTree()
ORDER BY (WindowStart, ServiceName);

CREATE MATERIALIZED VIEW IF NOT EXISTS default.mv_service_suggest
TO default.service_suggest AS
SELECT
  toStartOfHour(Timestamp) AS WindowStart,
  ServiceName,
  count() AS Cnt
FROM default.otel_traces
GROUP BY WindowStart, ServiceName;
