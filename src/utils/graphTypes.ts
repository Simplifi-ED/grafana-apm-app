/** Shared service-map graph shapes used by Mermaid export (and tests). */

export interface GraphNode {
  id: string;
  title: string;
  subtitle?: string;
  mainStat?: string;
  secondaryStat?: string;
  errorRate: number;
  nodeType?: 'service' | 'database' | 'messaging' | 'external';
  namespace?: string;
  isHub?: boolean;
  hubDegree?: number;
  callerCount?: number;
}

export interface GraphEdge {
  id: string;
  source: string;
  target: string;
  mainStat?: string;
  secondaryStat?: string;
}
