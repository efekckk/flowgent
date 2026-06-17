import BaseNode from './BaseNode';
import Icon from '../../ui/Icon';
import type { NodeProps } from 'reactflow';

export default function CodeNode(props: NodeProps) {
  return (
    <BaseNode
      data={props.data}
      accentClass="text-nodeCode"
      category="code"
      icon={<Icon name="braces" size={12} />}
    />
  );
}
