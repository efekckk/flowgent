import BaseNode from './BaseNode';
import type { NodeProps } from 'reactflow';

export default function CommunicationNode(props: NodeProps) {
  return <BaseNode data={props.data} colorClass="bg-nodeComm" icon="✉" />;
}
