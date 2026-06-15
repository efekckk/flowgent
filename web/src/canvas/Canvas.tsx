import { useMemo, useCallback, useState } from 'react';
import ReactFlow, {
  Background, Controls, MiniMap,
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
    <div className="h-full w-full">
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
      >
        <Background />
        <Controls showInteractive={false} />
        <MiniMap pannable zoomable />
      </ReactFlow>
    </div>
  );
}
