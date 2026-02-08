import { createFileRoute, Link, Outlet, useNavigate, useLocation } from '@tanstack/react-router';
import { Alert, Button, Card, Descriptions, Space, Tag, Tabs, Typography } from 'antd';
import { ArrowLeftOutlined } from '@ant-design/icons';
import { useDeployment } from '../hooks/useDeployment';
import { DeploymentStatusTag } from '../components/DeploymentStatusTag';
import { JobStatusTag } from '../components/JobStatusTag';
import { formatAge } from '../utils/format';

const { Title } = Typography;

export const Route = createFileRoute('/deployments/$namespace/$name')({
  component: DeploymentOverviewComponent,
});

function DeploymentOverviewComponent() {
  const { namespace, name } = Route.useParams();
  const deployment = useDeployment(namespace, name);
  const navigate = useNavigate();
  const location = useLocation();

  // Determine active tab based on current path
  const getActiveKey = () => {
    const path = location.pathname;
    if (path.endsWith('/checkpoints')) return 'checkpoints';
    if (path.endsWith('/storage')) return 'storage';
    return 'details';
  };

  const activeKey = getActiveKey();

  const handleTabChange = (key: string) => {
    if (key === 'details') {
      navigate({ to: '/deployments/$namespace/$name', params: { namespace, name } });
    } else {
      navigate({ to: `/deployments/$namespace/$name/${key}`, params: { namespace, name } });
    }
  };

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

  const { metadata, status } = deployment;

  const tabItems = [
    {
      key: 'details',
      label: 'Deployment Details',
    },
    {
      key: 'checkpoints',
      label: 'Checkpoints & Savepoints',
    },
    {
      key: 'storage',
      label: 'Storage',
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

      {/* Tabs and Outlet */}
      <Card>
        <Tabs
          activeKey={activeKey}
          items={tabItems}
          onChange={handleTabChange}
        />
        <div style={{ marginTop: '24px' }}>
           <Outlet />
        </div>
      </Card>
    </Space>
  );
}
