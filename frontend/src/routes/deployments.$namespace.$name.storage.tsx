import { createFileRoute } from '@tanstack/react-router';
import { Alert, Button, Card, Descriptions, Space, Spin, Table, Typography } from 'antd';
import { ReloadOutlined } from '@ant-design/icons';
import { useStorageCheckpoints } from '../hooks/useStorageCheckpoints';
import type { StorageEntry } from '../api/schema';

const { Title } = Typography;

// Helper function to format bytes to human-readable size
function formatBytes(bytes?: number): string {
  if (bytes === undefined || bytes === null) return 'N/A';
  if (bytes === 0) return '0 Bytes';

  const k = 1024;
  const sizes = ['Bytes', 'KB', 'MB', 'GB', 'TB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));

  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
}

// Helper function to format date
function formatDate(dateString?: string): string {
  if (!dateString) return 'N/A';
  
  try {
    const date = new Date(dateString);
    return date.toLocaleString();
  } catch {
    return 'Invalid date';
  }
}

export const Route = createFileRoute('/deployments/$namespace/$name/storage')({
  component: DeploymentStorageComponent,
});

function DeploymentStorageComponent() {
  const { namespace, name } = Route.useParams();
  const storageCheckpoints = useStorageCheckpoints(namespace, name);

  return (
    <Space direction="vertical" size="large" style={{ width: '100%' }}>
      {/* Storage Checkpoints and Savepoints from S3 */}
      <div>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '16px' }}>
          <Title level={4} style={{ margin: 0 }}>Storage Checkpoints & Savepoints</Title>
          <Button 
            icon={<ReloadOutlined />} 
            onClick={storageCheckpoints.refetch}
            loading={storageCheckpoints.isLoading}
          >
            Refresh
          </Button>
        </div>

        {storageCheckpoints.error && (
          <Alert
            type="error"
            message="Failed to Load Storage Data"
            description={storageCheckpoints.error}
            showIcon
            style={{ marginBottom: '16px' }}
          />
        )}

        {storageCheckpoints.isLoading && !storageCheckpoints.data && (
          <div style={{ textAlign: 'center', padding: '40px' }}>
            <Spin size="large" />
          </div>
        )}

        {storageCheckpoints.data && (
          <>
            {/* Configuration Info */}
            {(storageCheckpoints.data.jobId || storageCheckpoints.data.checkpointDir || storageCheckpoints.data.savepointDir) && (
              <Card title="Configuration" size="small" style={{ marginBottom: '16px' }}>
                <Descriptions column={1} size="small">
                  {storageCheckpoints.data.jobId && (
                    <Descriptions.Item label="Current Job ID">
                      <code style={{ fontSize: '12px' }}>{storageCheckpoints.data.jobId}</code>
                    </Descriptions.Item>
                  )}
                  {storageCheckpoints.data.checkpointDir && (
                    <Descriptions.Item label="Checkpoint Directory">
                      <code style={{ fontSize: '12px' }}>{storageCheckpoints.data.checkpointDir}</code>
                    </Descriptions.Item>
                  )}
                  {storageCheckpoints.data.savepointDir && (
                    <Descriptions.Item label="Savepoint Directory">
                      <code style={{ fontSize: '12px' }}>{storageCheckpoints.data.savepointDir}</code>
                    </Descriptions.Item>
                  )}
                </Descriptions>
              </Card>
            )}

            {/* Checkpoints Table */}
            {storageCheckpoints.data.checkpoints && storageCheckpoints.data.checkpoints.length > 0 && (
              <Card title="Checkpoints in Storage" size="small" style={{ marginBottom: '16px' }}>
                <Table
                  dataSource={storageCheckpoints.data.checkpoints}
                  columns={[
                    {
                      title: 'Name',
                      dataIndex: 'name',
                      key: 'name',
                      width: 150,
                    },
                    {
                      title: 'Job ID',
                      dataIndex: 'jobId',
                      key: 'jobId',
                      width: 300,
                      render: (jobId?: string) => jobId ? (
                        <code style={{ fontSize: '11px' }}>{jobId}</code>
                      ) : 'N/A',
                    },
                    {
                      title: 'Last Modified',
                      dataIndex: 'lastModified',
                      key: 'lastModified',
                      width: 180,
                      render: (lastModified?: string) => formatDate(lastModified),
                      sorter: (a: StorageEntry, b: StorageEntry) => {
                        if (!a.lastModified) return 1;
                        if (!b.lastModified) return -1;
                        return new Date(a.lastModified).getTime() - new Date(b.lastModified).getTime();
                      },
                      defaultSortOrder: 'descend',
                    },
                    {
                      title: 'Size',
                      dataIndex: 'size',
                      key: 'size',
                      width: 100,
                      render: (size?: number) => formatBytes(size),
                      sorter: (a: StorageEntry, b: StorageEntry) => (a.size ?? 0) - (b.size ?? 0),
                    },
                    {
                      title: 'S3 Path',
                      dataIndex: 'path',
                      key: 'path',
                      ellipsis: true,
                      render: (path: string) => <code style={{ fontSize: '11px' }}>{path}</code>,
                    },
                  ]}
                  rowKey={(record) => `${record.jobId}-${record.name}`}
                  size="small"
                  pagination={{
                    defaultPageSize: 50,
                    showSizeChanger: true,
                    showTotal: (total) => `Total ${total} checkpoints`,
                  }}
                />
              </Card>
            )}

            {/* Savepoints Table */}
            {storageCheckpoints.data.savepoints && storageCheckpoints.data.savepoints.length > 0 && (
              <Card title="Savepoints in Storage" size="small">
                <Table
                  dataSource={storageCheckpoints.data.savepoints}
                  columns={[
                    {
                      title: 'Name',
                      dataIndex: 'name',
                      key: 'name',
                      width: 200,
                    },
                    {
                      title: 'S3 Path',
                      dataIndex: 'path',
                      key: 'path',
                      ellipsis: true,
                      render: (path: string) => <code style={{ fontSize: '12px' }}>{path}</code>,
                    },
                  ]}
                  rowKey="name"
                  size="small"
                  pagination={{
                    defaultPageSize: 50,
                    showSizeChanger: true,
                    showTotal: (total) => `Total ${total} savepoints`,
                  }}
                />
              </Card>
            )}

            {/* No data message */}
            {(!storageCheckpoints.data.checkpoints || storageCheckpoints.data.checkpoints.length === 0) && 
             (!storageCheckpoints.data.savepoints || storageCheckpoints.data.savepoints.length === 0) && (
              <Alert
                type="info"
                message="No Storage Data Found"
                description={
                  !storageCheckpoints.data.jobId
                    ? 'No active job found. Storage checkpoints are filtered by the current job ID.'
                    : !storageCheckpoints.data.checkpointDir && !storageCheckpoints.data.savepointDir
                    ? 'No checkpoint or savepoint directories configured in flinkConfiguration (execution.checkpointing.dir or execution.checkpointing.savepoint-dir)'
                    : `No checkpoints or savepoints found for the current job (${storageCheckpoints.data.jobId}) in the configured storage directories`
                }
                showIcon
              />
            )}
          </>
        )}
      </div>
    </Space>
  );
}
