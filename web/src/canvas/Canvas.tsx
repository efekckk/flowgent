import { useMemo, useCallback, useState } from 'react';
import ReactFlow, {
  Background, BackgroundVariant, Controls, MiniMap,
  type Node, type Edge, type NodeMouseHandler,
} from 'reactflow';
import 'reactflow/dist/style.css';
import type { WorkflowDefinition } from '../api/types';
import { workflowToFlow } from './workflowToFlow';
import TriggerNode from './nodeTypes/TriggerNode';
import LLMNode from './nodeTypes/LLMNode';
import CommunicationNode from './nodeTypes/CommunicationNode';
import DataNode from './nodeTypes/DataNode';
import ControlNode from './nodeTypes/ControlNode';
import CodeNode from './nodeTypes/CodeNode';

const nodeTypes = {
  trigger: TriggerNode,
  llm: LLMNode,
  communication: CommunicationNode,
  data: DataNode,
  control: ControlNode,
  code: CodeNode,
};

interface Props {
  definition: WorkflowDefinition;
  onSelectNode?: (nodeId: string | null) => void;
}

export default function Canvas({ definition, onSelectNode }: Props) {
  const { nodes: initialNodes, edges: initialEdges } = useMemo(
    () => workflowToFlow(definition),
    [definition],
  );
  const [nodes] = useState<Node[]>(initialNodes);
  const [edges] = useState<Edge[]>(initialEdges);

  const onNodeClick = useCallback<NodeMouseHandler>((_, node) => {
    onSelectNode?.(node.id);
  }, [onSelectNode]);

  const onPaneClick = useCallback(() => onSelectNode?.(null), [onSelectNode]);

  return (
    <div className="relative h-full w-full bg-ink-800">
      <ReactFlow
        nodes={nodes}
        edges={edges}
        nodeTypes={nodeTypes}
        onNodeClick={onNodeClick}
        onPaneClick={onPaneClick}
        fitView
        nodesDraggable={false}
        nodesConnectable={false}
        edgesFocusable={false}
        elementsSelectable
        proOptions={{ hideAttribution: true }}
        defaultEdgeOptions={{
          type: 'smoothstep',
          animated: false,
          style: { stroke: 'rgba(125, 211, 252, 0.55)', strokeWidth: 1.25 },
        }}
      >
        <Background
          variant={BackgroundVariant.Dots}
          gap={24}
          size={1}
          color="rgba(125, 211, 252, 0.18)"
        />
        <Background
          variant={BackgroundVariant.Lines}
          gap={96}
          lineWidth={0.5}
          color="rgba(125, 211, 252, 0.08)"
        />
        <Controls showInteractive={false} position="bottom-right" />
        <MiniMap pannable zoomable
          nodeColor="#7DD3FC"
          nodeStrokeColor="#0EA5E9"
          nodeBorderRadius={0}
          maskColor="rgba(11, 18, 32, 0.6)"
        />
      </ReactFlow>
      {/* Vignette overlay */}
      <div
        className="pointer-events-none absolute inset-0"
        style={{ background: 'radial-gradient(ellipse at center, transparent 55%, rgba(0,0,0,0.45) 100%)' }}
      />
    </div>
  );
}
