import React, { useEffect, useState } from 'react'

type Item = {
  traceId: string
  startTs: string
  durationMs: number
  rootService: string
  rootOperation: string
  status: string
  spanCount: number
  svcBreakdown?: [string, number][]
}

export default function Finder({ onOpen }:{ onOpen:(id:string)=>void }){
  const [service, setService] = useState('')
  const [operation, setOperation] = useState('')
  const [status, setStatus] = useState('')
  const [items, setItems] = useState<Item[]>([])

  async function load(){
    const body = {
      from: Math.floor(Date.now()/1000) - 3600,
      to: Math.floor(Date.now()/1000),
      filters: {
        service: service? [service]: [],
        operation: operation? [operation]: [],
        status: status? [status]: []
      },
      sort: { by: 'duration', order: 'DESC' },
      page: { size: 50 }
    }
    const r = await fetch('/api/traces/list', { method:'POST', headers:{'Content-Type':'application/json'}, body: JSON.stringify(body)})
    const j = await r.json()
    setItems(j.items || [])
  }

  useEffect(()=>{ load() }, [])

  return (
    <div style={{padding:12, border:'1px solid #ddd', borderRadius:12}}>
      <div style={{display:'flex', gap:8, alignItems:'center', marginBottom:8}}>
        <input placeholder='Service' value={service} onChange={e=>setService(e.target.value)} />
        <input placeholder='Operation' value={operation} onChange={e=>setOperation(e.target.value)} />
        <select value={status} onChange={e=>setStatus(e.target.value)}>
          <option value=''>Any status</option>
          <option value='OK'>OK</option>
          <option value='ERROR'>ERROR</option>
        </select>
        <button onClick={load}>Search</button>
      </div>
      <table style={{width:'100%', borderCollapse:'collapse'}}>
        <thead><tr>
          <th align='left'>Start</th><th align='left'>Service</th><th align='left'>Operation</th>
          <th align='right'>Duration (ms)</th><th align='right'>Spans</th><th></th>
        </tr></thead>
        <tbody>
          {items.map((it)=>(
            <tr key={it.traceId}>
              <td>{it.startTs}</td>
              <td>{it.rootService}</td>
              <td>{it.rootOperation}</td>
              <td style={{textAlign:'right'}}>{it.durationMs.toFixed(2)}</td>
              <td style={{textAlign:'right'}}>{it.spanCount}</td>
              <td><button onClick={()=>onOpen(it.traceId)}>Open</button></td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}
