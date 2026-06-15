import type { Node, Edge } from 'reactflow';
import type { WorkflowDefinition } from '../api/types';

const CATEGORY_BY_PREFIX: Record<string, string> = {
  'http.': 'control',
  'core.if': 'control',
  'core.loop': 'control',
  'core.merge': 'control',
  'core.wait': 'control',
  'core.set': 'data',
  'core.code': 'code',
  'llm.': 'llm',
  'slack.': 'communication',
  'telegram.': 'communication',
  'email.': 'communication',
  'postgres.': 'data',
  'sheets.': 'data',
};

function nodeTypeFor(tool: string): string {
  for (const [prefix, type] of Object.entries(CATEGORY_BY_PREFIX)) {
    if (tool === prefix.replace(/\.$/, '') || tool.startsWith(prefix)) {
      return type;
    }
  }
  return 'control';
}

export function workflowToFlow(def: WorkflowDefinition): { nodes: Node[]; edges: Edge[] } {
  const nodes: Node[] = def.nodes.map((n, i) => ({
    id: n.id,
    type: nodeTypeFor(n.tool),
    position: n.position
      ? { x: n.position[0], y: n.position[1] }
      : { x: 100 + (i % 4) * 220, y: 80 + Math.floor(i / 4) * 140 },
    data: { id: n.id, tool: n.tool, params: n.params },
  }));
  const edges: Edge[] = def.edges.map((e, i) => ({
    id: `e${i}-${e.from}-${e.to}-${e.from_port}-${e.to_port}`,
    source: e.from,
    target: e.to,
    sourceHandle: e.from_port,
    targetHandle: e.to_port,
    label: e.from_port !== 'main' ? e.from_port : undefined,
  }));
  return { nodes, edges };
}
