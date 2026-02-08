import { createFileRoute } from '@tanstack/react-router';
import { Descriptions, Space, Tag } from 'antd';
import { useDeployment } from '../hooks/useDeployment';

export const Route = createFileRoute('/deployments/$namespace/$name/')({
  component: DeploymentDetailsComponent,
});

function DeploymentDetailsComponent() {
  const { namespace, name } = Route.useParams();
  const deployment = useDeployment(namespace, name);

  if (!deployment) {
    return null; // Parent handles "not found"
  }

  const { spec } = deployment;

  return (
    <Descriptions column={2} bordered size="small">
      <Descriptions.Item label="Image" span={2}>
        <code style={{ fontSize: '12px' }}>{spec.image}</code>
      </Descriptions.Item>
      <Descriptions.Item label="Flink Version">
        {spec.flinkVersion}
      </Descriptions.Item>
      <Descriptions.Item label="Parallelism">
        {spec.job.parallelism}
      </Descriptions.Item>
      <Descriptions.Item label="Entry Class" span={2}>
        <code style={{ fontSize: '12px' }}>{spec.job.entryClass}</code>
      </Descriptions.Item>
      <Descriptions.Item label="JAR URI" span={2}>
        <code style={{ fontSize: '12px' }}>{spec.job.jarURI}</code>
      </Descriptions.Item>
      <Descriptions.Item label="Upgrade Mode">
        {spec.job.upgradeMode}
      </Descriptions.Item>
      <Descriptions.Item label="Job State (Spec)">
        {spec.job.state}
      </Descriptions.Item>
      {spec.job.args && spec.job.args.length > 0 && (
        <Descriptions.Item label="Job Args" span={2}>
          <Space wrap>
            {spec.job.args.map((arg, idx) => (
              <Tag key={idx}>{arg}</Tag>
            ))}
          </Space>
        </Descriptions.Item>
      )}
      <Descriptions.Item label="Job Manager Resources">
        {spec.jobManager.resource.cpu} CPU / {spec.jobManager.resource.memory}
        {' '}({spec.jobManager.replicas} {spec.jobManager.replicas === 1 ? 'replica' : 'replicas'})
      </Descriptions.Item>
      <Descriptions.Item label="Task Manager Resources">
        {spec.taskManager.resource.cpu} CPU / {spec.taskManager.resource.memory}
      </Descriptions.Item>
    </Descriptions>
  );
}
