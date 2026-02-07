import { createFileRoute, Link } from '@tanstack/react-router';
import { Alert, Button, Card, Descriptions, Space, Tag, Tabs, Typography } from 'antd';
import { ArrowLeftOutlined } from '@ant-design/icons';
import { useDeployment } from '../hooks/useDeployment';
import { DeploymentStatusTag } from '../components/DeploymentStatusTag';
import { JobStatusTag } from '../components/JobStatusTag';
import { formatAge, formatTimestamp } from '../utils/format';

const { Title } = Typography;

export const Route = createFileRoute('/deployments/$namespace/$name')({
  component: DeploymentOverviewComponent,
});

function DeploymentOverviewComponent() {
  const { namespace, name } = Route.useParams();
  const deployment = useDeployment(namespace, name);

  if (!deployment) {
    return (
      <Card>
        <Space direction="vertical" size="middle" style={{ width: '100%' }}>
          <Link to="/">
            <Button icon={<ArrowLeftOutlined />}>Back to Deployments</Button>
          </Link>
          <Alert
            type="warning"
            message="Deployment Not Found"
            description={`Deployment "${name}" in namespace "${namespace}" was not found or has been deleted.`}
            showIcon
          />
        </Space>
      </Card>
    );
  }

  const { metadata, spec, status } = deployment;

  // Extract checkpoint and savepoint info
  const checkpointTimestamp = status?.jobStatus?.checkpointInfo?.lastPeriodicCheckpointTimestamp;
  const savepointTimestamp = status?.jobStatus?.savepointInfo?.lastPeriodicSavepointTimestamp;
  const savepointHistory = status?.jobStatus?.savepointInfo?.savepointHistory || [];

  return (
    <Space direction="vertical" size="large" style={{ width: '100%' }}>
      {/* Header with Status Summary */}
      <Card>
        <Space direction="vertical" size="middle" style={{ width: '100%' }}>
          <Link to="/">
            <Button icon={<ArrowLeftOutlined />} type="link" style={{ padding: 0 }}>
              Back to Deployments
            </Button>
          </Link>
          <div style={{ display: 'flex', alignItems: 'flex-start', justifyContent: 'space-between', gap: '24px' }}>
            <div style={{ flex: '0 0 auto' }}>
              <Title level={2} style={{ margin: 0, marginBottom: '8px' }}>{metadata.name}</Title>
              <Tag color="blue">{metadata.namespace}</Tag>
            </div>
            <div style={{ flex: '1 1 auto' }}>
              <Descriptions column={2} bordered size="small">
                <Descriptions.Item label="Lifecycle State">
                  {status?.lifecycleState ? (
                    <DeploymentStatusTag status={status.lifecycleState} />
                  ) : (
                    <Tag>N/A</Tag>
                  )}
                </Descriptions.Item>
                <Descriptions.Item label="Job State">
                  <JobStatusTag status={status?.jobStatus?.state} />
                </Descriptions.Item>
                <Descriptions.Item label="Job Manager Status">
                  {status?.jobManagerDeploymentStatus || 'N/A'}
                </Descriptions.Item>
                <Descriptions.Item label="Age">
                  {formatAge(metadata.creationTimestamp)}
                </Descriptions.Item>
                <Descriptions.Item label="Job Start Time">
                  {status?.jobStatus?.startTime || 'N/A'}
                </Descriptions.Item>
                <Descriptions.Item label="Job Update Time">
                  {status?.jobStatus?.updateTime || 'N/A'}
                </Descriptions.Item>
              </Descriptions>
            </div>
          </div>
        </Space>
      </Card>

      {/* Tabs for Details and Checkpoints */}
      <Card>
        <Tabs
          defaultActiveKey="details"
          items={[
            {
              key: 'details',
              label: 'Deployment Details',
              children: (
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
              ),
            },
            {
              key: 'checkpoints',
              label: 'Checkpoints & Savepoints',
              children: (
                <Descriptions column={1} bordered size="small">
                  <Descriptions.Item label="Last Periodic Checkpoint">
                    {checkpointTimestamp ? formatTimestamp(checkpointTimestamp) : 'N/A'}
                  </Descriptions.Item>
                  <Descriptions.Item label="Last Periodic Savepoint">
                    {savepointTimestamp ? formatTimestamp(savepointTimestamp) : 'N/A'}
                  </Descriptions.Item>
                  {savepointHistory.length > 0 && (
                    <Descriptions.Item label="Savepoint History">
                      <Space direction="vertical" size="small" style={{ width: '100%' }}>
                        {savepointHistory.map((savepoint, idx) => (
                          <code key={idx} style={{ fontSize: '12px', display: 'block' }}>
                            {savepoint}
                          </code>
                        ))}
                      </Space>
                    </Descriptions.Item>
                  )}
                </Descriptions>
              ),
            },
          ]}
        />
      </Card>
    </Space>
  );
}
