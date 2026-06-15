import BaseNode from './BaseNode';
import type { NodeProps } from 'reactflow';

export default function CodeNode(props: NodeProps) {
  return <BaseNode data={props.data} colorClass="bg-nodeCode" icon="{ }" />;
}
