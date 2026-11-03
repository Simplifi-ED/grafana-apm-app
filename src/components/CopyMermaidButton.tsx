import React, { useCallback, useState } from 'react';
import { IconButton } from '@grafana/ui';
import { toMermaidGraph } from '../utils/mermaid';
import type { GraphEdge, GraphNode } from '../utils/graphTypes';

interface CopyMermaidButtonProps {
  nodes: GraphNode[];
  edges: GraphEdge[];
  direction?: 'RIGHT' | 'DOWN';
}

export function CopyMermaidButton({ nodes, edges, direction = 'RIGHT' }: CopyMermaidButtonProps) {
  const [copied, setCopied] = useState(false);

  const handleCopy = useCallback(() => {
    const mermaid = toMermaidGraph(nodes, edges, direction);
    navigator.clipboard.writeText(mermaid).then(() => {
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    });
  }, [nodes, edges, direction]);

  return (
    <IconButton
      name={copied ? 'check' : 'copy'}
      tooltip={copied ? 'Copied!' : 'Copy as Mermaid'}
      size="md"
      onClick={handleCopy}
    />
  );
}
