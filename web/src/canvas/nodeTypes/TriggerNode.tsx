import BaseNode from './BaseNode';
import type { NodeProps } from 'reactflow';

export default function TriggerNode(props: NodeProps) {
  return <BaseNode data={props.data} colorClass="bg-nodeTrigger" icon="⚡" />;
}
