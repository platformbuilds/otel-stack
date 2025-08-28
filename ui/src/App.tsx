import React, { useState } from 'react'
import Finder from './components/Finder'
import TraceView from './components/TraceView'

export default function App(){
  const [traceId, setTraceId] = useState<string|undefined>(undefined)
  return (
    <div style={{fontFamily:'Inter, system-ui, -apple-system, Segoe UI, Roboto', padding:16}}>
      <h1 style={{marginTop:0}}>OTEL UI Demo</h1>
      <div style={{display:'grid', gap:16, gridTemplateColumns:'1fr'}}>
        <Finder onOpen={setTraceId} />
        {traceId && <TraceView traceId={traceId} />}
      </div>
    </div>
  )
}
