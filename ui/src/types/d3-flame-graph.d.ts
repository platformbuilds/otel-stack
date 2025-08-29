declare module 'd3-flame-graph' {
  import { Selection } from 'd3-selection'
  import { HierarchyNode } from 'd3-hierarchy'

  export interface FlameGraph {
    (sel: Selection<any, any, any, any>): void
    width(n: number): this
    height(n: number): this
    label(fn: (d: HierarchyNode<any>) => string): this
    value(fn: (d: HierarchyNode<any>) => number): this
    // you can add more here if you use them (minFrameSize, tooltip, etc.)
  }

  export function flamegraph(): FlameGraph
}