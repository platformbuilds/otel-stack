import React, { useEffect, useMemo, useState } from "react";

type Span = {
  spanId: string;
  parentSpanId?: string;
  name: string;
  service: string;
  startUnixNanos: number;
  endUnixNanos: number;
  attributes?: Record<string, string>;
  statusCode?: string;
  statusMessage?: string;
};

type FlameNode = {
  name: string;
  value: number;
  children?: FlameNode[];
};

type TraceResponse = {
  traceId: string;
  spans: Span[];
};

export default function TraceView({ traceId }: { traceId: string }) {
  const id = useMemo(() => (traceId || "").toLowerCase(), [traceId]);
  const [spans, setSpans] = useState<Span[]>([]);
  const [flame, setFlame] = useState<FlameNode | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let abort = false;
    setError(null);

    (async () => {
      try {
        const r = await fetch(`/api/traces/${id}`);
        if (!r.ok) throw new Error(`GET /api/traces/${id}: ${r.status}`);
        const j: TraceResponse = await r.json();
        if (!abort) setSpans(j.spans || []);
      } catch (e: any) {
        if (!abort) setError(e.message ?? String(e));
      }
    })();

    return () => {
      abort = true;
    };
  }, [id]);

  useEffect(() => {
    let abort = false;
    setError(null);

    (async () => {
      try {
        const r = await fetch(
          `/api/traces/${id}/flame?groupBy=service_operation&mode=total`
        );
        if (!r.ok) throw new Error(`GET /api/traces/${id}/flame: ${r.status}`);
        const j: FlameNode = await r.json();
        if (!abort) setFlame(j);
      } catch (e: any) {
        if (!abort) setError(e.message ?? String(e));
      }
    })();

    return () => {
      abort = true;
    };
  }, [id]);

  return (
    <div className="trace-view">
      <div style={{ marginBottom: 12 }}>
        <strong>Trace:</strong> {id}
        {error && (
          <span style={{ color: "crimson", marginLeft: 12 }}>
            Error: {error}
          </span>
        )}
      </div>

      <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 16 }}>
        <div>
          <h4>Timeline Spans ({spans.length})</h4>
          <ul
            style={{
              maxHeight: 360,
              overflow: "auto",
              border: "1px solid #eee",
              padding: 8,
            }}
          >
            {spans.map((s) => (
              <li key={s.spanId}>
                <code>{s.service}</code> — {s.name}
              </li>
            ))}
          </ul>
        </div>

        <div>
          <h4>Flame JSON</h4>
          <pre
            style={{
              maxHeight: 360,
              overflow: "auto",
              border: "1px solid #eee",
              padding: 8,
            }}
          >
            {flame ? JSON.stringify(flame, null, 2) : "Loading…"}
          </pre>
        </div>
      </div>
    </div>
  );
}