import { createFileRoute, Link, Outlet, useLocation } from '@tanstack/react-router';
import { Alert, Button, Card, Descriptions, Popconfirm, Space, Spin, Table, Typography } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { ReloadOutlined } from '@ant-design/icons';
import { useStorageCheckpoints } from '../hooks/useStorageCheckpoints';
import { useAdminMode } from '../context/useAdminMode';
import { useMessageApi } from '../context/MessageContext';
import { useDeploymentActions } from '../hooks/useDeploymentActions';
import { formatTimestamp } from '../utils/format';
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

  return formatTimestamp(dateString);
}

export const Route = createFileRoute('/deployments/$namespace/$name/storage')({
  component: DeploymentStorageComponent,
});

function DeploymentStorageComponent() {
  const { namespace, name } = Route.useParams();
  const location = useLocation();

  if (!location.pathname.endsWith('/storage')) {
    return <Outlet />;
  }

  return <DeploymentStorageList namespace={namespace} name={name} />;
}

interface DeploymentStorageListProps {
  namespace: string;
  name: string;
}

function DeploymentStorageList({ namespace, name }: DeploymentStorageListProps) {
  const { isAdminMode } = useAdminMode();
  const storageCheckpoints = useStorageCheckpoints(namespace, name);
  const { recoverDeployment, isLoading, pendingAction, error } = useDeploymentActions(namespace, name);
  const messageApi = useMessageApi();
  const stateEntries = [...(storageCheckpoints.data?.stateEntries ?? [])].sort((a, b) => {
    const aTime = a.lastModified ? new Date(a.lastModified).getTime() : 0;
    const bTime = b.lastModified ? new Date(b.lastModified).getTime() : 0;

    return bTime - aTime;
  });

  const handleRecover = async (entry: StorageEntry) => {
    try {
      await recoverDeployment(entry.path);
      void messageApi.success(`Recovery requested for ${namespace}/${name} from ${entry.name}`);
    } catch (requestError) {
      void messageApi.error(getErrorMessage(requestError, error));
    }
  };

  const columns: ColumnsType<StorageEntry> = [
    {
      title: 'Name',
      dataIndex: 'name',
      key: 'name',
      width: 150,
      render: (entryName: string, record: StorageEntry) => (
        <Link
          to="/deployments/$namespace/$name/storage/$entryType/$jobId/$entryName"
          params={{
            namespace,
            name,
            entryType: record.type,
            jobId: record.jobId || '-',
            entryName,
          }}
        >
          {entryName}
        </Link>
      ),
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
      title: 'Checkpoint ID',
      dataIndex: 'checkpointId',
      key: 'checkpointId',
      width: 140,
      render: (checkpointId?: number) => checkpointId ?? '-',
      sorter: (a: StorageEntry, b: StorageEntry) => (a.checkpointId ?? 0) - (b.checkpointId ?? 0),
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
    ...(isAdminMode ? [{
      title: 'Action',
      key: 'action',
      align: 'center' as const,
      width: 210,
      render: (_: unknown, record: StorageEntry) => (
        <Space size="small">
          <Popconfirm
            title="Recover deployment?"
            description="This will redeploy the job from the selected checkpoint or savepoint."
            okText="Recover"
            cancelText="Cancel"
            onConfirm={() => handleRecover(record)}
            disabled={isLoading}
          >
            <Button
              disabled={isLoading}
              loading={pendingAction === 'recover'}
              size="small"
              type="primary"
            >
              Recover
            </Button>
          </Popconfirm>
        </Space>
      ),
    }] : []),
  ];

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
                  columns={columns}
                  rowKey={getStorageEntryKey}
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

function getStorageEntryKey(entry: StorageEntry): string {
  return `${entry.type}-${entry.jobId ?? ''}-${entry.name}-${entry.path}`;
}

function getErrorMessage(error: unknown, fallback: string | null): string {
  if (error && typeof error === 'object' && 'message' in error && typeof error.message === 'string') {
    return error.message;
  }

  if (error instanceof Error) {
    return error.message;
  }

  return fallback ?? 'Failed to recover deployment';
}
