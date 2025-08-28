import React, { useEffect, useMemo, useRef, useState } from 'react'
import Flame from './flame/Flame'
import Timeline from './timeline/Timeline'

type Span = {
  spanId: string
  parentSpanId?: string
  name: string
  service: string
  startUnixNanos: number
  endUnixNanos: number
}

export default function TraceView({ traceId }:{ traceId:string }){
  const [spans, setSpans] = useState<Span[]>([])
  const [flame, setFlame] = useState<any>(null)
  const [tab, setTab] = useState<'timeline'|'flame'>('timeline')

  useEffect(()=>{
    (async()=>{
      const r = await fetch(`/api/traces/${traceId}`); const j = await r.json(); setSpans(j.spans||[])
      const rf = await fetch(`/api/traces/${traceId}/flame?groupBy=service_operation&mode=self`); setFlame(await rf.json())
    })()
  }, [traceId])

  return (
    <div style={{padding:12, border:'1px solid #ddd', borderRadius:12}}>
      <div style={{display:'flex', gap:8, alignItems:'center', marginBottom:8}}>
        <strong>Trace:</strong> <code>{traceId}</code>
        <span style={{marginLeft:'auto'}}>
          <button onClick={()=>setTab('timeline')} disabled={tab==='timeline'}>Timeline</button>
          <button onClick={()=>setTab('flame')} disabled={tab==='flame'}>Flame</button>
        </span>
      </div>
      {tab==='timeline' ? <Timeline spans={spans} /> : <Flame data={flame} />}
    </div>
  )
}
