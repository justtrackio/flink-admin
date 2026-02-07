import { createFileRoute, Link } from '@tanstack/react-router';
import { Alert, Button, Card, Descriptions, Space, Tag, Tabs, Typography, Table, Spin, Statistic, Row, Col } from 'antd';
import { ArrowLeftOutlined, ReloadOutlined } from '@ant-design/icons';
import { useDeployment } from '../hooks/useDeployment';
import { useCheckpoints } from '../hooks/useCheckpoints';
import { DeploymentStatusTag } from '../components/DeploymentStatusTag';
import { JobStatusTag } from '../components/JobStatusTag';
import { formatAge, formatTimestamp, formatBytes, formatDuration } from '../utils/format';
import type { FlinkCheckpointDetail } from '../api/schema';

const { Title } = Typography;

export const Route = createFileRoute('/deployments/$namespace/$name')({
  component: DeploymentOverviewComponent,
});

function DeploymentOverviewComponent() {
  const { namespace, name } = Route.useParams();
  const deployment = useDeployment(namespace, name);
  const checkpoints = useCheckpoints(namespace, name);

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

  // Checkpoint history table columns
  const checkpointColumns = [
    {
      title: 'ID',
      dataIndex: 'id',
      key: 'id',
      width: 100,
    },
    {
      title: 'Status',
      dataIndex: 'status',
      key: 'status',
      width: 120,
      render: (status: string) => (
        <Tag color={status === 'COMPLETED' ? 'success' : status === 'FAILED' ? 'error' : 'processing'}>
          {status}
        </Tag>
      ),
    },
    {
      title: 'Type',
      dataIndex: 'checkpoint_type',
      key: 'checkpoint_type',
      width: 150,
    },
    {
      title: 'Trigger Time',
      dataIndex: 'trigger_timestamp',
      key: 'trigger_timestamp',
      width: 200,
      render: (timestamp: number) => formatTimestamp(timestamp),
    },
    {
      title: 'Duration',
      dataIndex: 'end_to_end_duration',
      key: 'end_to_end_duration',
      width: 100,
      render: (duration: number) => formatDuration(duration),
    },
    {
      title: 'State Size',
      dataIndex: 'state_size',
      key: 'state_size',
      width: 100,
      render: (size: number) => formatBytes(size),
    },
    {
      title: 'Subtasks',
      key: 'subtasks',
      width: 100,
      render: (_: unknown, record: FlinkCheckpointDetail) => 
        `${record.num_acknowledged_subtasks} / ${record.num_subtasks}`,
    },
    {
      title: 'Storage Path',
      dataIndex: 'external_path',
      key: 'external_path',
      ellipsis: true,
      render: (path: string) => path ? <code style={{ fontSize: '12px' }}>{path}</code> : '-',
    },
  ];

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
                <Space direction="vertical" size="large" style={{ width: '100%' }}>
                  {/* Checkpoint Statistics from Flink REST API */}
                  <div>
                    <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '16px' }}>
                      <Title level={4} style={{ margin: 0 }}>Checkpoint Statistics</Title>
                      <Button 
                        icon={<ReloadOutlined />} 
                        onClick={checkpoints.refetch}
                        loading={checkpoints.isLoading}
                      >
                        Refresh
                      </Button>
                    </div>

                    {checkpoints.error && (
                      <Alert
                        type="error"
                        message="Failed to Load Checkpoints"
                        description={checkpoints.error}
                        showIcon
                        style={{ marginBottom: '16px' }}
                      />
                    )}

                    {checkpoints.isLoading && !checkpoints.data && (
                      <div style={{ textAlign: 'center', padding: '40px' }}>
                        <Spin size="large" />
                      </div>
                    )}

                    {checkpoints.data && (
                      <>
                        {/* Checkpoint Counts */}
                        <Card title="Counts" size="small" style={{ marginBottom: '16px' }}>
                          <Row gutter={16}>
                            <Col span={4}>
                              <Statistic title="Total" value={checkpoints.data.counts.total} />
                            </Col>
                            <Col span={4}>
                              <Statistic 
                                title="Completed" 
                                value={checkpoints.data.counts.completed}
                                valueStyle={{ color: '#3f8600' }}
                              />
                            </Col>
                            <Col span={4}>
                              <Statistic 
                                title="Failed" 
                                value={checkpoints.data.counts.failed}
                                valueStyle={{ color: '#cf1322' }}
                              />
                            </Col>
                            <Col span={4}>
                              <Statistic 
                                title="In Progress" 
                                value={checkpoints.data.counts.in_progress}
                                valueStyle={{ color: '#1890ff' }}
                              />
                            </Col>
                            <Col span={4}>
                              <Statistic title="Restored" value={checkpoints.data.counts.restored} />
                            </Col>
                          </Row>
                        </Card>

                        {/* Checkpoint History */}
                        {checkpoints.data.history.length > 0 && (
                          <Card title="Checkpoint History" size="small">
                            <Table
                              dataSource={checkpoints.data.history}
                              columns={checkpointColumns}
                              rowKey="id"
                              size="small"
                              pagination={{
                                defaultPageSize: 50,
                                showSizeChanger: true,
                                showTotal: (total) => `Total ${total} checkpoints`,
                              }}
                            />
                          </Card>
                        )}
                      </>
                    )}
                  </div>
                </Space>
              ),
            },
          ]}
        />
      </Card>
    </Space>
  );
}
