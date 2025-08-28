import React, { useMemo } from 'react'

type Span = {
  spanId: string
  parentSpanId?: string
  name: string
  service: string
  startUnixNanos: number
  endUnixNanos: number
}

export default function Timeline({ spans }: { spans: Span[] }) {
  const { min, max, byService } = useMemo(() => {
    let min = Number.POSITIVE_INFINITY, max = 0
    const byService: Record<string, Span[]> = {}
    for (const s of spans) {
      min = Math.min(min, s.startUnixNanos)
      max = Math.max(max, s.endUnixNanos)
      if (!byService[s.service]) byService[s.service] = []
      byService[s.service].push(s)
    }
    for (const k of Object.keys(byService)) {
      byService[k].sort((a, b) => a.startUnixNanos - b.startUnixNanos)
    }
    return { min, max, byService }
  }, [spans])

  if (!spans || spans.length === 0) return <div>No spans.</div>

  const width = 1000, laneH = 28, pad = 80
  const total = Math.max(1, max - min) // guard against divide-by-zero
  const x = (ns: number) => pad + (ns - min) / total * (width - pad - 20)

  const services = Object.keys(byService)
  const height = pad / 2 + services.length * laneH + 20

  return (
    <svg width={width} height={height} style={{ background: '#fff' }}>
      {/* axes */}
      <text x={pad} y={20} style={{ fontSize: 12 }}>Start</text>
      <text x={width - 40} y={20} style={{ fontSize: 12, textAnchor: 'end' }}>End</text>

      {/* lanes */}
      {services.map((svc, i) => {
        const y = pad / 2 + i * laneH
        return (
          <g key={svc} transform={`translate(0,${y})`}>
            <text x={6} y={laneH / 2} dominantBaseline="middle" style={{ fontSize: 12 }}>{svc}</text>
            <line x1={pad} y1={laneH / 2} x2={width - 10} y2={laneH / 2} stroke="#eee" />
            {byService[svc].map(s => {
              const x1 = x(s.startUnixNanos), x2 = x(s.endUnixNanos)
              const w = Math.max(2, x2 - x1)
              return (
                <g key={s.spanId}>
                  <title>{s.name}</title>
                  <rect x={x1} y={6} width={w} height={laneH - 12} fill="#89a" rx={4} />
                  <text x={x1 + 4} y={laneH / 2} dominantBaseline="middle" style={{ fontSize: 11, fill: '#fff' }}>
                    {s.name}
                  </text>
                </g>
              )
            })}
          </g>
        )
      })}
    </svg>
  )
}