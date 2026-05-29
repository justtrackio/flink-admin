import { createFileRoute, Link } from '@tanstack/react-router';
import type { ReactNode } from 'react';
import { Alert, Button, Card, Descriptions, Empty, Space, Spin, Table, Tag, Typography } from 'antd';
import { ArrowLeftOutlined, ReloadOutlined } from '@ant-design/icons';
import { useStorageCheckpointMetadata } from '../hooks/useStorageCheckpointMetadata';
import { formatBytes, formatTimestampUtc } from '../utils/format';
import type { StorageCheckpointOperatorSummary } from '../api/schema';

const { Title, Text } = Typography;

export const Route = createFileRoute('/deployments/$namespace/$name/storage/$entryType/$jobId/$entryName')({
  component: StorageCheckpointMetadataComponent,
});

function StorageCheckpointMetadataComponent() {
  const { namespace, name, entryType, jobId, entryName } = Route.useParams();
  const metadata = useStorageCheckpointMetadata(namespace, name, entryType, jobId, entryName);

  return (
    <Space direction="vertical" size="large" style={{ width: '100%' }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', gap: '16px', flexWrap: 'wrap' }}>
        <Space direction="vertical" size={4}>
          <Link to="/deployments/$namespace/$name/storage" params={{ namespace, name }}>
            <Button icon={<ArrowLeftOutlined />} type="link" style={{ padding: 0 }}>
              Back to Storage
            </Button>
          </Link>
          <Space align="center" wrap>
            <Title level={4} style={{ margin: 0 }}>{entryName}</Title>
            <Tag color={entryType === 'checkpoint' ? 'blue' : 'purple'}>{entryType}</Tag>
          </Space>
        </Space>
        <Button icon={<ReloadOutlined />} onClick={metadata.refetch} loading={metadata.isLoading}>
          Refresh
        </Button>
      </div>

      {metadata.error && (
        <Alert
          type="error"
          message="Failed to Load Metadata"
          description={metadata.error}
          showIcon
        />
      )}

      {metadata.isLoading && !metadata.data && (
        <div style={{ textAlign: 'center', padding: '40px' }}>
          <Spin size="large" />
        </div>
      )}

      {metadata.data && (
        <>
          {!metadata.data.metadataExists && (
            <Alert
              type="warning"
              message="Metadata File Missing"
              description="This checkpoint/savepoint directory does not contain a _metadata file."
              showIcon
            />
          )}

          {metadata.data.parseStatus === 'failed' && (
            <Alert
              type="error"
              message="Metadata Parse Failed"
              description={metadata.data.parseError ?? 'The _metadata file could not be parsed.'}
              showIcon
            />
          )}

          <Card title="Summary" size="small">
            <Descriptions column={2} size="small" bordered>
              <Descriptions.Item label="Type">
                <Tag color={metadata.data.type === 'checkpoint' ? 'blue' : 'purple'}>{metadata.data.type}</Tag>
              </Descriptions.Item>
              <Descriptions.Item label="Name">{metadata.data.name}</Descriptions.Item>
              {metadata.data.jobId && (
                <Descriptions.Item label="Job ID">
                  <code style={{ fontSize: '11px' }}>{metadata.data.jobId}</code>
                </Descriptions.Item>
              )}
              <Descriptions.Item label="Parse Status">
                <ParseStatusTag status={metadata.data.parseStatus} />
              </Descriptions.Item>
              <Descriptions.Item label="Checkpoint ID">
                {metadata.data.summary?.checkpointId ?? 'N/A'}
              </Descriptions.Item>
              <Descriptions.Item label="Metadata Version">
                {metadata.data.summary?.version ?? 'N/A'}
              </Descriptions.Item>
              <Descriptions.Item label="Operators">
                {metadata.data.summary?.numOperators ?? 'N/A'}
              </Descriptions.Item>
              <Descriptions.Item label="Referenced State Files">
                {metadata.data.summary?.stateFilePaths.length ?? 'N/A'}
              </Descriptions.Item>
              <Descriptions.Item label="Metadata Size">
                {metadata.data.size !== undefined ? formatBytes(metadata.data.size) : 'N/A'}
              </Descriptions.Item>
              <Descriptions.Item label="Last Modified">
                {metadata.data.lastModified ? formatTimestampUtc(metadata.data.lastModified) : 'N/A'}
              </Descriptions.Item>
            </Descriptions>
          </Card>

          <Card title="S3 Paths" size="small">
            <Descriptions column={1} size="small">
              <Descriptions.Item label="State Directory">
                <code style={{ fontSize: '12px' }}>{metadata.data.path}</code>
              </Descriptions.Item>
              <Descriptions.Item label="Metadata File">
                <code style={{ fontSize: '12px' }}>{metadata.data.metadataPath}</code>
              </Descriptions.Item>
            </Descriptions>
          </Card>

          <Card title="Checkpoint Properties" size="small">
            {metadata.data.summary?.properties ? (
              <Descriptions column={1} size="small" bordered>
                <Descriptions.Item label="Checkpoint Type">
                  {metadata.data.summary.properties.checkpointType || 'N/A'}
                </Descriptions.Item>
                <Descriptions.Item label="Sharing Strategy">
                  {metadata.data.summary.properties.sharingStrategy || 'N/A'}
                </Descriptions.Item>
                <Descriptions.Item label="Source">
                  {metadata.data.summary.properties.source || 'N/A'}
                </Descriptions.Item>
              </Descriptions>
            ) : (
              <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="No checkpoint properties found" />
            )}
          </Card>

          <Card title="Operators" size="small">
            {metadata.data.summary?.operators.length ? (
              <Table
                dataSource={metadata.data.summary.operators}
                columns={operatorColumns}
                rowKey={(record) => record.operatorId}
                size="small"
                pagination={{
                  defaultPageSize: 25,
                  showSizeChanger: true,
                  showTotal: (total) => `Total ${total} operators`,
                }}
              />
            ) : (
              <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="No operators found" />
            )}
          </Card>

          <Card title="Referenced State Files" size="small">
            {metadata.data.summary?.stateFilePaths.length ? (
              <Table
                dataSource={metadata.data.summary.stateFilePaths.map((path, index) => ({ index, path }))}
                columns={[
                  {
                    title: '#',
                    dataIndex: 'index',
                    key: 'index',
                    width: 80,
                    render: (index: number) => index + 1,
                  },
                  {
                    title: 'Path',
                    dataIndex: 'path',
                    key: 'path',
                    ellipsis: true,
                    render: (path: string) => <code style={{ fontSize: '11px' }}>{path}</code>,
                  },
                ]}
                rowKey="path"
                size="small"
                pagination={{
                  defaultPageSize: 25,
                  showSizeChanger: true,
                  showTotal: (total) => `Total ${total} state files`,
                }}
              />
            ) : (
              <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="No referenced state files found" />
            )}
          </Card>
        </>
      )}
    </Space>
  );
}

function ParseStatusTag({ status }: { status: string }) {
  if (status === 'parsed') {
    return <Tag color="success">parsed</Tag>;
  }

  if (status === 'failed') {
    return <Tag color="error">failed</Tag>;
  }

  return <Tag color="warning">missing</Tag>;
}

const operatorColumns = [
  {
    title: 'Name',
    dataIndex: 'name',
    key: 'name',
    width: 240,
    render: (value: string) => value || <Text type="secondary">N/A</Text>,
  },
  {
    title: 'UID',
    dataIndex: 'uid',
    key: 'uid',
    width: 220,
    render: (value: string) => value || <Text type="secondary">N/A</Text>,
  },
  {
    title: 'Operator ID',
    dataIndex: 'operatorId',
    key: 'operatorId',
    width: 260,
    render: (value: string) => <code style={{ fontSize: '11px' }}>{value}</code>,
  },
  {
    title: 'Parallelism',
    dataIndex: 'parallelism',
    key: 'parallelism',
    width: 120,
  },
  {
    title: 'Max Parallelism',
    dataIndex: 'maxParallelism',
    key: 'maxParallelism',
    width: 150,
  },
] satisfies Array<{
  title: string;
  dataIndex: keyof StorageCheckpointOperatorSummary;
  key: string;
  width?: number;
  render?: (value: string) => ReactNode;
}>;
