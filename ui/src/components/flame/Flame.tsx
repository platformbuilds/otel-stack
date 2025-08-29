import React, { useEffect, useRef } from 'react'
import * as d3 from 'd3'
import { flamegraph } from 'd3-flame-graph'

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

    const chart = flamegraph()
      .width(1000)
      .height(480)
      .label(d => d.data.name)
      .value(d => d.data.value)

    d3.select(ref.current).datum(data).call(chart)
  }, [data])

  return <div ref={ref} />
}