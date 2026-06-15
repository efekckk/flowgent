import BaseNode from './BaseNode';
import type { NodeProps } from 'reactflow';

export default function LLMNode(props: NodeProps) {
  return <BaseNode data={props.data} colorClass="bg-nodeLLM" icon="🤖" />;
}
