import BaseNode from './BaseNode';
import Icon from '../../ui/Icon';
import type { NodeProps } from 'reactflow';

export default function TriggerNode(props: NodeProps) {
  return (
    <BaseNode
      data={props.data}
      accentClass="text-nodeTrigger"
      category="trigger"
      icon={<Icon name="bolt" size={12} />}
    />
  );
}
