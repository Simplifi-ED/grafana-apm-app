import {
  createDataFrame,
  FieldType,
  NodeGraphDataFrameFieldNames as F,
  type DataFrame,
  type DataLink,
  type PreferredVisualisationType,
} from '@grafana/data';
import type { ServiceMapEdge, ServiceMapNode, ServiceMapResponse } from '../api/client';
import type { GraphEdge, GraphNode } from './graphTypes';

const NODE_GRAPH_VIZ: PreferredVisualisationType = 'nodeGraph';

export type NodeLinkMode = 'namespace' | 'serviceSearch' | 'serviceOverview';

export interface ToNodeGraphFramesOptions {
  /** How node clicks should navigate via data links. */
  linkMode: NodeLinkMode;
  /** Plugin base path, e.g. `/a/omnivya-apm-app`. */
  pluginBaseUrl: string;
  /**
   * Namespace segment for `serviceOverview` links
   * (`/services/{namespace}/{service}`). Defaults to `_`.
   */
  namespace?: string;
  /** Optional environment query string value for overview links. */
  environment?: string;
}

/** Convert API map payload into Mermaid-friendly node/edge arrays. */
export function toGraphData(mapData: ServiceMapResponse | null): {
  graphNodes: GraphNode[];
  graphEdges: GraphEdge[];
} {
  if (!mapData) {
    return { graphNodes: [], graphEdges: [] };
  }
  return {
    graphNodes: mapData.nodes.map((n) => ({
      id: n.id,
      title: n.title,
      subtitle: n.subtitle,
      mainStat: n.mainStat,
      secondaryStat: n.secondaryStat,
      errorRate: n.errorRate ?? n.arc__errors ?? 0,
      nodeType: n.nodeType ?? 'service',
      isHub: n.isHub,
      hubDegree: n.hubDegree,
      callerCount: n.callerCount,
    })),
    graphEdges: mapData.edges.map((e) => ({
      id: e.id,
      source: e.source,
      target: e.target,
      mainStat: e.mainStat,
      secondaryStat: e.secondaryStat,
    })),
  };
}

function nodeDataLink(opts: ToNodeGraphFramesOptions): DataLink {
  const base = opts.pluginBaseUrl.replace(/\/$/, '');
  const idVar = `\${__data.fields.${F.id}}`;

  switch (opts.linkMode) {
    case 'namespace':
      return {
        title: 'Open namespace',
        url: `${base}/namespaces/${idVar}`,
      };
    case 'serviceSearch':
      return {
        title: 'Find service',
        url: `${base}/services?q=${idVar}`,
      };
    case 'serviceOverview': {
      const ns = encodeURIComponent(opts.namespace || '_');
      const env = opts.environment ? `?environment=${encodeURIComponent(opts.environment)}` : '';
      return {
        title: 'Open service',
        url: `${base}/services/${ns}/${idVar}${env}`,
      };
    }
    default: {
      const _exhaustive: never = opts.linkMode;
      return _exhaustive;
    }
  }
}

function nodesFrame(nodes: ServiceMapNode[], link: DataLink): DataFrame {
  const links = [link];
  return createDataFrame({
    name: 'nodes',
    refId: 'nodes',
    meta: { preferredVisualisationType: NODE_GRAPH_VIZ },
    fields: [
      { name: F.id, type: FieldType.string, values: nodes.map((n) => n.id), config: { links } },
      { name: F.title, type: FieldType.string, values: nodes.map((n) => n.title), config: { links } },
      {
        name: F.subTitle,
        type: FieldType.string,
        values: nodes.map((n) => n.subtitle ?? ''),
      },
      {
        name: F.mainStat,
        type: FieldType.string,
        values: nodes.map((n) => n.mainStat ?? ''),
      },
      {
        name: F.secondaryStat,
        type: FieldType.string,
        values: nodes.map((n) => n.secondaryStat ?? ''),
      },
      {
        name: `${F.arc}errors`,
        type: FieldType.number,
        values: nodes.map((n) => n.arc__errors ?? n.errorRate ?? 0),
        config: { color: { mode: 'fixed', fixedColor: 'red' } },
      },
      {
        name: `${F.arc}ok`,
        type: FieldType.number,
        values: nodes.map((n) => n.arc__ok ?? Math.max(0, 1 - (n.arc__errors ?? n.errorRate ?? 0))),
        config: { color: { mode: 'fixed', fixedColor: 'green' } },
      },
    ],
  });
}

function edgesFrame(edges: ServiceMapEdge[]): DataFrame {
  return createDataFrame({
    name: 'edges',
    refId: 'edges',
    meta: { preferredVisualisationType: NODE_GRAPH_VIZ },
    fields: [
      { name: F.id, type: FieldType.string, values: edges.map((e) => e.id) },
      { name: F.source, type: FieldType.string, values: edges.map((e) => e.source) },
      { name: F.target, type: FieldType.string, values: edges.map((e) => e.target) },
      {
        name: F.mainStat,
        type: FieldType.string,
        values: edges.map((e) => e.mainStat ?? ''),
      },
      {
        name: F.secondaryStat,
        type: FieldType.string,
        values: edges.map((e) => e.secondaryStat ?? ''),
      },
    ],
  });
}

/** Build Grafana Node Graph DataFrames from a `/service-map` response. */
export function toNodeGraphFrames(
  mapData: ServiceMapResponse | null | undefined,
  opts: ToNodeGraphFramesOptions
): DataFrame[] {
  if (!mapData || (mapData.nodes.length === 0 && mapData.edges.length === 0)) {
    return [];
  }
  const link = nodeDataLink(opts);
  return [nodesFrame(mapData.nodes, link), edgesFrame(mapData.edges)];
}
