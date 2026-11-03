import { NodeGraphDataFrameFieldNames as F } from '@grafana/data';
import type { ServiceMapResponse } from '../api/client';
import { nodeGraphHeight, toGraphData, toNodeGraphFrames } from './nodeGraphFrames';

const sample: ServiceMapResponse = {
  nodes: [
    {
      id: 'api',
      title: 'api',
      subtitle: 'ns-a',
      mainStat: '1.0 req/s',
      secondaryStat: '0.0% errors',
      arc__errors: 0.1,
      arc__ok: 0.9,
      errorRate: 0.1,
      nodeType: 'service',
    },
    {
      id: 'db',
      title: 'db',
      mainStat: '0.5 req/s',
      arc__errors: 0,
      arc__ok: 1,
      errorRate: 0,
      nodeType: 'database',
    },
  ],
  edges: [
    {
      id: 'api->db',
      source: 'api',
      target: 'db',
      mainStat: '0.5 req/s',
      secondaryStat: 'P95: 12ms',
    },
  ],
};

describe('toGraphData', () => {
  it('returns empty arrays for null', () => {
    expect(toGraphData(null)).toEqual({ graphNodes: [], graphEdges: [] });
  });

  it('maps API nodes and edges', () => {
    const { graphNodes, graphEdges } = toGraphData(sample);
    expect(graphNodes).toHaveLength(2);
    expect(graphNodes[0]).toMatchObject({ id: 'api', errorRate: 0.1, nodeType: 'service' });
    expect(graphEdges[0]).toMatchObject({ source: 'api', target: 'db' });
  });
});

describe('toNodeGraphFrames', () => {
  it('returns empty for null or empty maps', () => {
    expect(toNodeGraphFrames(null, { linkMode: 'namespace', pluginBaseUrl: '/a/x' })).toEqual([]);
    expect(toNodeGraphFrames({ nodes: [], edges: [] }, { linkMode: 'namespace', pluginBaseUrl: '/a/x' })).toEqual([]);
  });

  it('builds nodes and edges frames with nodeGraph meta and required fields', () => {
    const frames = toNodeGraphFrames(sample, {
      linkMode: 'namespace',
      pluginBaseUrl: '/a/omnivya-apm-app',
    });
    expect(frames).toHaveLength(2);
    expect(frames[0].meta?.preferredVisualisationType).toBe('nodeGraph');
    expect(frames[1].meta?.preferredVisualisationType).toBe('nodeGraph');

    const nodeNames = frames[0].fields.map((f) => f.name);
    expect(nodeNames).toEqual(
      expect.arrayContaining([F.id, F.title, F.subTitle, F.mainStat, F.secondaryStat, `${F.arc}errors`, `${F.arc}ok`])
    );

    const edgeNames = frames[1].fields.map((f) => f.name);
    expect(edgeNames).toEqual(expect.arrayContaining([F.id, F.source, F.target, F.mainStat, F.secondaryStat]));
  });

  it('precomputes finite fixedx/fixedy so Node Graph skips its layout worker', () => {
    const [nodes] = toNodeGraphFrames(sample, { linkMode: 'namespace', pluginBaseUrl: '/a/x' });
    const xs = nodes.fields.find((f) => f.name === F.fixedX)!.values;
    const ys = nodes.fields.find((f) => f.name === F.fixedY)!.values;
    expect(xs).toHaveLength(2);
    expect([...xs, ...ys].every(Number.isFinite)).toBe(true);
    expect(xs[0]).toBeLessThan(xs[1]);
  });

  it('sizes the graph from the layout span', () => {
    expect(nodeGraphHeight([])).toBe(0);
    const frames = toNodeGraphFrames(sample, { linkMode: 'namespace', pluginBaseUrl: '/a/x' });
    expect(nodeGraphHeight(frames)).toBe(220);
  });

  it('opens node links in the same tab', () => {
    const [nodes] = toNodeGraphFrames(sample, { linkMode: 'namespace', pluginBaseUrl: '/a/x' });
    expect(nodes.fields[0].config.links?.[0].targetBlank).toBe(false);
  });

  it('attaches namespace data-link templates on id', () => {
    const [nodes] = toNodeGraphFrames(sample, {
      linkMode: 'namespace',
      pluginBaseUrl: '/a/omnivya-apm-app/',
    });
    const idField = nodes.fields.find((f) => f.name === F.id)!;
    expect(idField.config.links?.[0].url).toBe(`/a/omnivya-apm-app/namespaces/\${__data.fields.${F.id}}`);
  });

  it('attaches serviceSearch and serviceOverview link templates', () => {
    const [searchNodes] = toNodeGraphFrames(sample, {
      linkMode: 'serviceSearch',
      pluginBaseUrl: '/a/omnivya-apm-app',
    });
    expect(searchNodes.fields[0].config.links?.[0].url).toBe(`/a/omnivya-apm-app/services?q=\${__data.fields.${F.id}}`);

    const [overviewNodes] = toNodeGraphFrames(sample, {
      linkMode: 'serviceOverview',
      pluginBaseUrl: '/a/omnivya-apm-app',
      namespace: 'team-a',
      environment: 'prod',
    });
    expect(overviewNodes.fields[0].config.links?.[0].url).toBe(
      `/a/omnivya-apm-app/services/team-a/\${__data.fields.${F.id}}?environment=prod`
    );
  });
});
