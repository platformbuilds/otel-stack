-- Attribute value dictionary for common keys (extend as needed)
CREATE TABLE IF NOT EXISTS default.attr_values
(
  WindowStart DateTime,
  Key LowCardinality(String),
  Val LowCardinality(String),
  Cnt UInt64
)
ENGINE = SummingMergeTree()
ORDER BY (WindowStart, Key, Val);

CREATE MATERIALIZED VIEW IF NOT EXISTS default.mv_attr_values
TO default.attr_values AS
WITH keys AS (
  SELECT array('http.method','deployment.environment','db.system','http.route') AS k
)
SELECT
  toStartOfHour(t.Timestamp) AS WindowStart,
  k AS Key,
  JSON_VALUE(t.SpanAttributes, concat('$.', k)) AS Val,
  count() AS Cnt
FROM default.otel_traces t
ARRAY JOIN (SELECT * FROM keys).k AS k
WHERE Val != ''
GROUP BY WindowStart, Key, Val;
