import BaseNode from './BaseNode';
import Icon from '../../ui/Icon';
import type { NodeProps } from 'reactflow';

export default function LLMNode(props: NodeProps) {
  return (
    <BaseNode
      data={props.data}
      accentClass="text-nodeLLM"
      category="model · llm"
      icon={<Icon name="chip" size={12} />}
    />
  );
}
