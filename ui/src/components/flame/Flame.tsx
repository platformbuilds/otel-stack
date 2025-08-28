import React, { useEffect, useRef } from 'react'
import * as d3 from 'd3'
import { flamegraph } from 'd3-flame-graph'

// Shape of your flame nodes
type FlameNode = {
  name: string
  value: number
  children?: FlameNode[]
}

export default function Flame({ data }: { data: FlameNode }) {
  const ref = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!data || !ref.current) return
    ref.current.innerHTML = ''

    // cast to any so TS doesn’t complain about missing .value()
    const chart = (flamegraph() as any)
      .width(1000)
      .height(480)
      .label((d: any) => d.data.name)
      .value((d: any) => d.data.value)

    d3.select(ref.current).datum(data).call(chart)

  }, [data])

  return <div ref={ref} />
}
