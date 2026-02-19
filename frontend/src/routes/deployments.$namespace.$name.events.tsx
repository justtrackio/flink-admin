import { createFileRoute } from '@tanstack/react-router';
import { Alert, Button, Space, Spin, Table, Tag, Typography } from 'antd';
import { ReloadOutlined } from '@ant-design/icons';
import { useEvents } from '../hooks/useEvents';
import { formatAge } from '../utils/format';
import type { K8sEvent } from '../api/schema';

const { Title } = Typography;

export const Route = createFileRoute('/deployments/$namespace/$name/events')({
  component: DeploymentEventsComponent,
});

function DeploymentEventsComponent() {
  const { namespace, name } = Route.useParams();
  const events = useEvents(namespace, name);

  const columns = [
    {
      title: 'Time',
      dataIndex: 'eventTime',
      key: 'eventTime',
      width: 110,
      render: (eventTime: string) => (eventTime ? formatAge(eventTime) : '-'),
    },
    {
      title: 'Type',
      dataIndex: 'type',
      key: 'type',
      width: 100,
      render: (type: string) => (
        <Tag color={type === 'Warning' ? 'error' : 'success'}>{type}</Tag>
      ),
    },
    {
      title: 'Reason',
      dataIndex: 'reason',
      key: 'reason',
      width: 180,
    },
    {
      title: 'Message',
      dataIndex: 'note',
      key: 'note',
      render: (note: string) => (
        <span style={{ whiteSpace: 'pre-wrap' }}>{note}</span>
      ),
    },
    {
      title: 'Source',
      dataIndex: 'reportingController',
      key: 'reportingController',
      width: 220,
      ellipsis: true,
    },
    {
      title: 'Action',
      dataIndex: 'action',
      key: 'action',
      width: 160,
      render: (action: string) => action || '-',
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
            Kubernetes Events
          </Title>
          <Button
            icon={<ReloadOutlined />}
            onClick={events.refetch}
            loading={events.isLoading}
          >
            Refresh
          </Button>
        </div>

        {events.error && (
          <Alert
            type="error"
            message="Failed to Load Events"
            description={events.error}
            showIcon
            style={{ marginBottom: '16px' }}
          />
        )}

        {events.isLoading && !events.data && (
          <div style={{ textAlign: 'center', padding: '40px' }}>
            <Spin size="large" />
          </div>
        )}

        {events.data && (
          <Table<K8sEvent>
            dataSource={events.data}
            columns={columns}
            rowKey={(record) => `${record.eventTime}-${record.reason}-${record.note}`}
            size="small"
            pagination={{
              defaultPageSize: 50,
              showSizeChanger: true,
              showTotal: (total) => `Total ${total} events`,
            }}
            locale={{ emptyText: 'No events found for this deployment' }}
          />
        )}
      </div>
    </Space>
  );
}
