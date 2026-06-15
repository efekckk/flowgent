import { describe, it, expect } from 'vitest';
import { workflowToFlow } from './workflowToFlow';

describe('workflowToFlow', () => {
  it('assigns category types based on tool slug', () => {
    const result = workflowToFlow({
      nodes: [
        { id: 'a', tool: 'core.set', params: {} },
        { id: 'b', tool: 'http.request', params: {} },
        { id: 'c', tool: 'slack.send_message', params: {} },
        { id: 'd', tool: 'llm.chat', params: {} },
        { id: 'e', tool: 'core.code', params: {} },
      ],
      edges: [],
    });
    const byId = (id: string) => result.nodes.find((n) => n.id === id);
    expect(byId('a')?.type).toBe('data');
    expect(byId('b')?.type).toBe('control');
    expect(byId('c')?.type).toBe('communication');
    expect(byId('d')?.type).toBe('llm');
    expect(byId('e')?.type).toBe('code');
  });

  it('maps edges including from_port labels', () => {
    const result = workflowToFlow({
      nodes: [
        { id: 'cond', tool: 'core.if', params: {} },
        { id: 'next', tool: 'core.set', params: {} },
      ],
      edges: [
        { from: 'cond', from_port: 'true', to: 'next', to_port: 'main' },
      ],
    });
    expect(result.edges).toHaveLength(1);
    expect(result.edges[0].source).toBe('cond');
    expect(result.edges[0].target).toBe('next');
    expect(result.edges[0].label).toBe('true');
  });
});
