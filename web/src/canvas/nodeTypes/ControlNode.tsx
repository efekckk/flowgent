import BaseNode from './BaseNode';
import Icon from '../../ui/Icon';
import type { NodeProps } from 'reactflow';

export default function ControlNode(props: NodeProps) {
  const showFalse = props.data.tool === 'core.if';
  return (
    <BaseNode
      data={props.data}
      accentClass="text-nodeControl"
      category="control"
      icon={<Icon name="branch" size={12} />}
      showFalseHandle={showFalse}
    />
  );
}
