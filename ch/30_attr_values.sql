CREATE TABLE default.attr_values
(
    WindowStart   DateTime,
    Key           String,
    Val           String,
    Cnt           UInt64
)
ENGINE = SummingMergeTree
ORDER BY (WindowStart, Key, Val);

CREATE MATERIALIZED VIEW default.mv_attr_values
TO default.attr_values
AS
WITH ['http.method','deployment.environment','db.system','http.route'] AS keys
SELECT
    toStartOfHour(Timestamp)                                      AS WindowStart,
    k                                                             AS Key,
    v                                                             AS Val,
    count()                                                       AS Cnt
FROM default.otel_traces
ARRAY JOIN mapKeys(SpanAttributes)   AS k,
           mapValues(SpanAttributes) AS v
WHERE k IN keys AND v IS NOT NULL AND v != ''
GROUP BY WindowStart, Key, Val;