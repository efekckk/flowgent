import BaseNode from './BaseNode';
import type { NodeProps } from 'reactflow';

export default function ControlNode(props: NodeProps) {
  const showFalse = props.data.tool === 'core.if';
  return <BaseNode data={props.data} colorClass="bg-nodeControl" icon="⚙" showFalseHandle={showFalse} />;
}
