import BaseNode from './BaseNode';
import type { NodeProps } from 'reactflow';

export default function DataNode(props: NodeProps) {
  return <BaseNode data={props.data} colorClass="bg-nodeData" icon="🗄" />;
}
