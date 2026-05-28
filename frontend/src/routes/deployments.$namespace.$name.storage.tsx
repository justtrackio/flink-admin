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
  const stateEntries = [...(storageCheckpoints.data?.stateEntries ?? [])].sort((a, b) => {
    const aTime = a.lastModified ? new Date(a.lastModified).getTime() : 0;
    const bTime = b.lastModified ? new Date(b.lastModified).getTime() : 0;

    return bTime - aTime;
  });

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
            {(storageCheckpoints.data.checkpointDir || storageCheckpoints.data.savepointDir) && (
              <Card title="Configuration" size="small" style={{ marginBottom: '16px' }}>
                <Descriptions column={1} size="small">
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

            {/* State Entries Table */}
            {stateEntries.length > 0 && (
              <Card title="Checkpoints & Savepoints in Storage" size="small">
                <Table
                  dataSource={stateEntries}
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
                      ) : '-',
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
                  rowKey={(record) => `${record.type}-${record.jobId ?? ''}-${record.name}-${record.path}`}
                  size="small"
                  pagination={{
                    defaultPageSize: 50,
                    showSizeChanger: true,
                    showTotal: (total) => `Total ${total} state entries`,
                  }}
                />
              </Card>
            )}

            {/* No data message */}
            {stateEntries.length === 0 && (
              <Alert
                type="info"
                message="No Storage Data Found"
                description={
                  !storageCheckpoints.data.checkpointDir && !storageCheckpoints.data.savepointDir
                    ? 'No checkpoint or savepoint directories configured in flinkConfiguration (execution.checkpointing.dir or execution.checkpointing.savepoint-dir)'
                    : 'No checkpoints or savepoints found in the configured storage directories'
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
