import BaseNode from './BaseNode';
import Icon from '../../ui/Icon';
import type { NodeProps } from 'reactflow';

export default function CommunicationNode(props: NodeProps) {
  return (
    <BaseNode
      data={props.data}
      accentClass="text-nodeComm"
      category="dispatch"
      icon={<Icon name="signal" size={12} />}
    />
  );
}
