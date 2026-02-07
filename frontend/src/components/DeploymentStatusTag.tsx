import { Tag } from 'antd';

interface DeploymentStatusTagProps {
  status: string;
}

export function DeploymentStatusTag({ status }: DeploymentStatusTagProps) {
  const colorMap: Record<string, string> = {
    STABLE: 'success',
    DEPLOYED: 'processing',
    ROLLING_BACK: 'error',
    ROLLED_BACK: 'warning',
    UPGRADING: 'processing',
    SUSPENDED: 'default',
  };
  
  const color = colorMap[status.toUpperCase()] || 'default';
  return <Tag color={color}>{status.toUpperCase()}</Tag>;
}
