import { Tag } from 'antd';

interface JobStatusTagProps {
  status?: string;
}

export function JobStatusTag({ status }: JobStatusTagProps) {
  const colorMap: Record<string, string> = {
    RUNNING: 'processing',
    FINISHED: 'success',
    CANCELED: 'default',
    CANCELING: 'warning',
    FAILED: 'error',
    RESTARTING: 'warning',
    CREATED: 'default',
    RECONCILING: 'processing',
    SUSPENDED: 'default',
  };

  const upperStatus = status?.toUpperCase();
  const color = upperStatus ? colorMap[upperStatus] || 'default' : 'default';

  return <Tag color={color}>{upperStatus || 'N/A'}</Tag>;
}
