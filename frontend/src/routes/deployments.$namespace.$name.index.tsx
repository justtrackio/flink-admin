import { createFileRoute } from '@tanstack/react-router';
import { Descriptions, Table } from 'antd';
import { useMemo } from 'react';
import { useDeployment } from '../hooks/useDeployment';

export const Route = createFileRoute('/deployments/$namespace/$name/')({
  component: DeploymentDetailsComponent,
});

function DeploymentDetailsComponent() {
  const { namespace, name } = Route.useParams();
  const deployment = useDeployment(namespace, name);

  const jobArgsTableData = useMemo(() => {
    const args = deployment?.spec.job.args ?? [];
    const rows: Array<{ key: string; argKey: string; argValue: string }> = [];
    for (let idx = 0; idx < args.length; idx += 2) {
      const argKey = args[idx];
      const argValue = args[idx + 1] ?? '';
      rows.push({ key: `${idx}-${argKey}`, argKey, argValue });
    }
    return rows;
  }, [deployment?.spec.job.args]);

  if (!deployment) {
    return null; // Parent handles "not found"
  }

  const { spec } = deployment;

  const jobArgs = spec.job.args ?? [];

  const jobArgsColumns = [
    {
      title: 'Key',
      dataIndex: 'argKey',
      key: 'argKey',
      render: (value: string) => <code style={{ fontSize: '12px' }}>{value}</code>,
    },
    {
      title: 'Value',
      dataIndex: 'argValue',
      key: 'argValue',
      render: (value: string) => <code style={{ fontSize: '12px' }}>{value}</code>,
    },
  ];

  return (
    <Descriptions column={2} bordered size="small">
      <Descriptions.Item label="Image" span={2}>
        <code style={{ fontSize: '12px' }}>{spec.image}</code>
      </Descriptions.Item>
      <Descriptions.Item label="Flink Version" span={2}>
        {spec.flinkVersion}
      </Descriptions.Item>
      <Descriptions.Item label="Parallelism" span={2}>
        {spec.job.parallelism}
      </Descriptions.Item>
      <Descriptions.Item label="Entry Class" span={2}>
        <code style={{ fontSize: '12px' }}>{spec.job.entryClass}</code>
      </Descriptions.Item>
      <Descriptions.Item label="JAR URI" span={2}>
        <code style={{ fontSize: '12px' }}>{spec.job.jarURI}</code>
      </Descriptions.Item>
      <Descriptions.Item label="Upgrade Mode" span={2}>
        {spec.job.upgradeMode}
      </Descriptions.Item>
      <Descriptions.Item label="Job State (Spec)" span={2}>
        {spec.job.state}
      </Descriptions.Item>
      {jobArgs.length > 0 && (
        <Descriptions.Item label="Job Args" span={2}>
          <Table
            columns={jobArgsColumns}
            dataSource={jobArgsTableData}
            pagination={false}
            size="small"
          />
        </Descriptions.Item>
      )}
      <Descriptions.Item label="Job Manager Resources" span={2}>
        {spec.jobManager.resource.cpu} CPU / {spec.jobManager.resource.memory}
        {' '}({spec.jobManager.replicas} {spec.jobManager.replicas === 1 ? 'replica' : 'replicas'})
      </Descriptions.Item>
      <Descriptions.Item label="Task Manager Resources" span={2}>
        {spec.taskManager.resource.cpu} CPU / {spec.taskManager.resource.memory}
      </Descriptions.Item>
    </Descriptions>
  );
}
