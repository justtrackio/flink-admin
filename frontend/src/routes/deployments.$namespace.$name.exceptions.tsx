import { createFileRoute } from '@tanstack/react-router';
import { Alert, Button, Space, Spin, Table, Tag, Typography } from 'antd';
import { ReloadOutlined } from '@ant-design/icons';
import { useExceptions } from '../hooks/useExceptions';
import { formatTimestamp } from '../utils/format';
import type { FlinkExceptionEntry } from '../api/schema';

const { Title, Text } = Typography;

export const Route = createFileRoute('/deployments/$namespace/$name/exceptions')({
  component: DeploymentExceptionsComponent,
});

function StacktraceBlock({ stacktrace }: { stacktrace: string }) {
  return (
    <pre
      style={{
        background: '#1e1e1e',
        color: '#d4d4d4',
        padding: '12px 16px',
        borderRadius: '4px',
        fontSize: '12px',
        lineHeight: '1.5',
        overflowX: 'auto',
        maxHeight: '400px',
        overflowY: 'auto',
        margin: 0,
        whiteSpace: 'pre-wrap',
        wordBreak: 'break-all',
      }}
    >
      {stacktrace}
    </pre>
  );
}

function ConcurrentExceptionsTable({ exceptions }: { exceptions: FlinkExceptionEntry[] }) {
  const columns = [
    {
      title: 'Exception',
      dataIndex: 'exceptionName',
      key: 'exceptionName',
      ellipsis: true,
      render: (name: string) => (
        <Text code style={{ fontSize: '12px' }}>
          {name}
        </Text>
      ),
    },
    {
      title: 'Task',
      dataIndex: 'taskName',
      key: 'taskName',
      width: 220,
      render: (task: string) => task || '-',
    },
    {
      title: 'Location',
      dataIndex: 'location',
      key: 'location',
      width: 200,
      ellipsis: true,
      render: (loc: string) => loc || '-',
    },
  ];

  return (
    <Table<FlinkExceptionEntry>
      dataSource={exceptions}
      columns={columns}
      rowKey={(_, idx) => String(idx)}
      size="small"
      pagination={false}
      expandable={{
        expandedRowRender: (record) => <StacktraceBlock stacktrace={record.stacktrace} />,
        rowExpandable: (record) => Boolean(record.stacktrace),
      }}
    />
  );
}

function DeploymentExceptionsComponent() {
  const { namespace, name } = Route.useParams();
  const exceptions = useExceptions(namespace, name);

  const entries = exceptions.data?.exceptionHistory.entries ?? [];
  const truncated = exceptions.data?.exceptionHistory.truncated ?? false;

  const columns = [
    {
      title: 'Time',
      dataIndex: 'timestamp',
      key: 'timestamp',
      width: 160,
      render: (ts: number) => (ts ? formatTimestamp(ts) : '-'),
    },
    {
      title: 'Exception',
      dataIndex: 'exceptionName',
      key: 'exceptionName',
      ellipsis: true,
      render: (name: string) => (
        <Text code style={{ fontSize: '12px' }}>
          {name}
        </Text>
      ),
    },
    {
      title: 'Task',
      dataIndex: 'taskName',
      key: 'taskName',
      width: 220,
      render: (task: string) => task || '-',
    },
    {
      title: 'Location',
      dataIndex: 'location',
      key: 'location',
      width: 200,
      ellipsis: true,
      render: (loc: string) => loc || '-',
    },
    {
      title: 'Labels',
      key: 'failureLabels',
      width: 160,
      render: (_: unknown, record: FlinkExceptionEntry) => {
        const labels = record.failureLabels;
        if (!labels || Object.keys(labels).length === 0) return '-';
        return (
          <Space size={4} wrap>
            {Object.entries(labels).map(([k, v]) => (
              <Tag key={k} style={{ fontSize: '11px' }}>
                {k}: {v}
              </Tag>
            ))}
          </Space>
        );
      },
    },
  ];

  return (
    <Space direction="vertical" size="large" style={{ width: '100%' }}>
      <div>
        <div
          style={{
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'center',
            marginBottom: '16px',
          }}
        >
          <Title level={4} style={{ margin: 0 }}>
            Job Exceptions
          </Title>
          <Button
            icon={<ReloadOutlined />}
            onClick={exceptions.refetch}
            loading={exceptions.isLoading}
          >
            Refresh
          </Button>
        </div>

        {exceptions.error && (
          <Alert
            type="error"
            message="Failed to Load Exceptions"
            description={exceptions.error}
            showIcon
            style={{ marginBottom: '16px' }}
          />
        )}

        {truncated && (
          <Alert
            type="info"
            message="Exception history is truncated. Showing the most recent 50 exceptions."
            showIcon
            style={{ marginBottom: '16px' }}
          />
        )}

        {exceptions.isLoading && !exceptions.data && (
          <div style={{ textAlign: 'center', padding: '40px' }}>
            <Spin size="large" />
          </div>
        )}

        {exceptions.data && (
          <Table<FlinkExceptionEntry>
            dataSource={entries}
            columns={columns}
            rowKey={(_, idx) => String(idx)}
            size="small"
            pagination={{
              defaultPageSize: 20,
              showSizeChanger: true,
              showTotal: (total) => `Total ${total} exceptions`,
            }}
            locale={{ emptyText: 'No exceptions found for this job' }}
            expandable={{
              expandedRowRender: (record) => (
                <Space direction="vertical" size="middle" style={{ width: '100%' }}>
                  <div>
                    <Text strong>Stacktrace</Text>
                    <div style={{ marginTop: '8px' }}>
                      <StacktraceBlock stacktrace={record.stacktrace} />
                    </div>
                  </div>
                  {record.concurrentExceptions && record.concurrentExceptions.length > 0 && (
                    <div>
                      <Text strong>
                        Concurrent Exceptions ({record.concurrentExceptions.length})
                      </Text>
                      <div style={{ marginTop: '8px' }}>
                        <ConcurrentExceptionsTable exceptions={record.concurrentExceptions} />
                      </div>
                    </div>
                  )}
                </Space>
              ),
              rowExpandable: (record) =>
                Boolean(record.stacktrace) ||
                (record.concurrentExceptions?.length ?? 0) > 0,
            }}
          />
        )}
      </div>
    </Space>
  );
}
