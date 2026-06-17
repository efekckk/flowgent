import BaseNode from './BaseNode';
import Icon from '../../ui/Icon';
import type { NodeProps } from 'reactflow';

export default function DataNode(props: NodeProps) {
  return (
    <BaseNode
      data={props.data}
      accentClass="text-nodeData"
      category="data"
      icon={<Icon name="cylinder" size={12} />}
    />
  );
}
