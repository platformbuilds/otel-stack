-- Target table for results
CREATE TABLE default.attr_values
(
    WindowStart   DateTime,
    Key           String,
    Val           String,
    Cnt           UInt64
)
ENGINE = SummingMergeTree
ORDER BY (WindowStart, Key, Val);

-- MV definition
CREATE MATERIALIZED VIEW default.mv_attr_values
TO default.attr_values
AS
WITH
    -- List of interesting attribute keys
    ['http.method','deployment.environment','db.system','http.route'] AS keys
SELECT
    toStartOfHour(Timestamp)                          AS WindowStart,
    k                                                 AS Key,
    JSON_VALUE(SpanAttributes, concat('$.', k))       AS Val,
    count()                                           AS Cnt
FROM default.otel_traces
-- expand the list of keys (array join makes one row per element)
ARRAY JOIN keys AS k
WHERE Val IS NOT NULL
GROUP BY WindowStart, Key, Val;