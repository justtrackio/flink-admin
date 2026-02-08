import { createFileRoute } from '@tanstack/react-router';
import { Alert, Button, Card, Col, Row, Spin, Statistic, Table, Tag, Typography, Space } from 'antd';
import { ReloadOutlined } from '@ant-design/icons';
import { useCheckpoints } from '../hooks/useCheckpoints';
import { formatTimestamp, formatBytes, formatDuration } from '../utils/format';
import type { FlinkCheckpointDetail } from '../api/schema';

const { Title } = Typography;

export const Route = createFileRoute('/deployments/$namespace/$name/checkpoints')({
  component: DeploymentCheckpointsComponent,
});

function DeploymentCheckpointsComponent() {
  const { namespace, name } = Route.useParams();
  const checkpoints = useCheckpoints(namespace, name);

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
  );
}
