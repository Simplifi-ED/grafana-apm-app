export interface LayoutEdge {
  source: string;
  target: string;
}

export interface Point {
  x: number;
  y: number;
}

export interface LayeredLayoutOptions {
  /** Horizontal distance between layers (callers left, callees right). */
  layerGap?: number;
  /** Vertical distance between nodes within a layer. */
  nodeGap?: number;
  /** Barycenter sweeps used to reduce edge crossings. */
  sweeps?: number;
}

const DEFAULTS: Required<LayeredLayoutOptions> = { layerGap: 260, nodeGap: 140, sweeps: 8 };

/**
 * Left-to-right layered (Sugiyama-style) layout, centred on the origin.
 *
 * Grafana's Node Graph renders `fixedx`/`fixedy` positions without running its
 * layout web worker, whose chunk fails to load from app plugins in Grafana 13.3.
 */
export function layeredLayout(
  nodeIds: string[],
  edges: LayoutEdge[],
  options: LayeredLayoutOptions = {}
): Map<string, Point> {
  const { layerGap, nodeGap, sweeps } = { ...DEFAULTS, ...options };
  const positions = new Map<string, Point>();
  if (nodeIds.length === 0) {
    return positions;
  }

  const index = new Map(nodeIds.map((id, i) => [id, i]));
  const dagEdges = acyclicEdges(nodeIds, edges, index);

  const preds = new Map<string, string[]>(nodeIds.map((id) => [id, []]));
  const succs = new Map<string, string[]>(nodeIds.map((id) => [id, []]));
  for (const e of dagEdges) {
    preds.get(e.target)!.push(e.source);
    succs.get(e.source)!.push(e.target);
  }

  const layerOf = assignLayers(nodeIds, preds, succs);
  const layerCount = Math.max(...layerOf.values()) + 1;
  const layers: string[][] = Array.from({ length: layerCount }, () => []);
  for (const id of nodeIds) {
    layers[layerOf.get(id)!].push(id);
  }

  orderLayers(layers, preds, succs, index, sweeps);

  const xOffset = ((layerCount - 1) * layerGap) / 2;
  layers.forEach((layer, l) => {
    const yOffset = ((layer.length - 1) * nodeGap) / 2;
    layer.forEach((id, row) => {
      positions.set(id, { x: l * layerGap - xOffset, y: row * nodeGap - yOffset });
    });
  });
  return positions;
}

/** Drop self-loops, dangling edges and DFS back-edges so the graph is a DAG. */
function acyclicEdges(nodeIds: string[], edges: LayoutEdge[], index: Map<string, number>): LayoutEdge[] {
  const adjacency = new Map<string, LayoutEdge[]>(nodeIds.map((id) => [id, []]));
  for (const e of edges) {
    if (e.source !== e.target && index.has(e.source) && index.has(e.target)) {
      adjacency.get(e.source)!.push(e);
    }
  }

  const state = new Map<string, 'visiting' | 'done'>();
  const kept: LayoutEdge[] = [];
  for (const root of nodeIds) {
    if (state.has(root)) {
      continue;
    }
    const stack: Array<{ id: string; next: number }> = [{ id: root, next: 0 }];
    state.set(root, 'visiting');
    while (stack.length > 0) {
      const frame = stack[stack.length - 1];
      const out = adjacency.get(frame.id)!;
      if (frame.next >= out.length) {
        state.set(frame.id, 'done');
        stack.pop();
        continue;
      }
      const edge = out[frame.next++];
      const targetState = state.get(edge.target);
      if (targetState === 'visiting') {
        continue;
      }
      kept.push(edge);
      if (targetState === undefined) {
        state.set(edge.target, 'visiting');
        stack.push({ id: edge.target, next: 0 });
      }
    }
  }
  return kept;
}

/** Longest-path layering from sources, then pull sinks next to their callers. */
function assignLayers(
  nodeIds: string[],
  preds: Map<string, string[]>,
  succs: Map<string, string[]>
): Map<string, number> {
  const layerOf = new Map<string, number>();
  const indegree = new Map(nodeIds.map((id) => [id, preds.get(id)!.length]));
  const queue = nodeIds.filter((id) => indegree.get(id) === 0);
  for (const id of queue) {
    layerOf.set(id, 0);
  }
  for (let i = 0; i < queue.length; i++) {
    const id = queue[i];
    for (const next of succs.get(id)!) {
      layerOf.set(next, Math.max(layerOf.get(next) ?? 0, layerOf.get(id)! + 1));
      indegree.set(next, indegree.get(next)! - 1);
      if (indegree.get(next) === 0) {
        queue.push(next);
      }
    }
  }

  // Sources sit right before their nearest callee instead of all stacking at layer 0.
  for (const id of [...queue].reverse()) {
    const out = succs.get(id)!;
    if (preds.get(id)!.length === 0 && out.length > 0) {
      layerOf.set(id, Math.min(...out.map((t) => layerOf.get(t)!)) - 1);
    }
  }
  return layerOf;
}

/** Barycenter heuristic: alternate down/up sweeps ordering each layer by neighbour positions. */
function orderLayers(
  layers: string[][],
  preds: Map<string, string[]>,
  succs: Map<string, string[]>,
  index: Map<string, number>,
  sweeps: number
): void {
  const rowOf = new Map<string, number>();
  const syncRows = (layer: string[]) => layer.forEach((id, row) => rowOf.set(id, row));
  layers.forEach(syncRows);

  const reorder = (layer: string[], neighbours: Map<string, string[]>) => {
    const weight = new Map<string, number>();
    for (const id of layer) {
      const adj = neighbours.get(id)!;
      weight.set(id, adj.length > 0 ? adj.reduce((sum, n) => sum + rowOf.get(n)!, 0) / adj.length : rowOf.get(id)!);
    }
    layer.sort((a, b) => weight.get(a)! - weight.get(b)! || index.get(a)! - index.get(b)!);
    syncRows(layer);
  };

  for (let s = 0; s < sweeps; s++) {
    if (s % 2 === 0) {
      for (let l = 1; l < layers.length; l++) {
        reorder(layers[l], preds);
      }
    } else {
      for (let l = layers.length - 2; l >= 0; l--) {
        reorder(layers[l], succs);
      }
    }
  }
}
