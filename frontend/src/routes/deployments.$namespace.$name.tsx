import { createFileRoute, Link, Outlet, useLocation, useNavigate } from '@tanstack/react-router';
import { Alert, Badge, Button, Card, Col, Descriptions, Row, Space, Tabs, Tag, Typography } from 'antd';
import { ArrowLeftOutlined, HomeOutlined } from '@ant-design/icons';
import { useDeployment } from '../hooks/useDeployment';
import { useEvents } from '../hooks/useEvents';
import { useExceptions } from '../hooks/useExceptions';
import { DeploymentStatusTag } from '../components/DeploymentStatusTag';
import { JobStatusTag } from '../components/JobStatusTag';
import { formatAge } from '../utils/format';

const { Title } = Typography;

interface DeploymentSearchParams {
  fromNamespace?: string;
  fromLifecycleState?: string;
  fromShowNotRunning?: boolean;
}

export const Route = createFileRoute('/deployments/$namespace/$name')({
  component: DeploymentOverviewComponent,
  validateSearch: (search: Record<string, unknown>): DeploymentSearchParams => ({
    fromNamespace: typeof search.fromNamespace === 'string' ? search.fromNamespace : undefined,
    fromLifecycleState: typeof search.fromLifecycleState === 'string' ? search.fromLifecycleState : undefined,
    fromShowNotRunning: search.fromShowNotRunning === true || search.fromShowNotRunning === 'true' ? true : undefined,
  }),
});

function DeploymentOverviewComponent() {
  const { namespace, name } = Route.useParams();
  const searchParams = Route.useSearch();
  const deployment = useDeployment(namespace, name);
  const events = useEvents(namespace, name);
  const exceptions = useExceptions(namespace, name);
  const navigate = useNavigate();
  const location = useLocation();

  const returnSearch = {
    namespace: searchParams.fromNamespace,
    lifecycleState: searchParams.fromLifecycleState,
    showNotRunning: searchParams.fromShowNotRunning,
  };

  // Determine active tab based on current path
  const getActiveKey = () => {
    const path = location.pathname;
    if (path.endsWith('/checkpoints')) return 'checkpoints';
    if (path.endsWith('/storage')) return 'storage';
    if (path.endsWith('/events')) return 'events';
    if (path.endsWith('/exceptions')) return 'exceptions';
    return 'details';
  };

  const activeKey = getActiveKey();

  const handleTabChange = (key: string) => {
    if (key === 'details') {
      navigate({ to: '/deployments/$namespace/$name', params: { namespace, name }, search: searchParams });
      return;
    }

    navigate({ to: `/deployments/$namespace/$name/${key}`, params: { namespace, name }, search: searchParams });
  };

  if (!deployment) {
    return (
      <Card>
        <Space direction="vertical" size="middle" style={{ width: '100%' }}>
          <Link to="/" search={returnSearch}>
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

  const warningCount = events.data?.filter((e) => e.type !== 'Normal').length ?? 0;
  const exceptionCount = exceptions.data?.exceptionHistory.entries.length ?? 0;

  const tabItems = [
    {
      key: 'details',
      label: 'Deployment Details',
    },
    {
      key: 'exceptions',
      label: (
        <span>
          Exceptions
          {exceptionCount > 0 && (
            <Badge count={exceptionCount} size="small" style={{ marginLeft: 6 }} />
          )}
        </span>
      ),
    },
    {
      key: 'events',
      label: (
        <span>
          Events
          {warningCount > 0 && (
            <Badge count={warningCount} size="small" style={{ marginLeft: 6 }} />
          )}
        </span>
      ),
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
          <Link to="/" search={returnSearch}>
            <Button icon={<ArrowLeftOutlined />} type="link" style={{ padding: 0 }}>
              Back to Deployments
            </Button>
          </Link>
          <div style={{ display: 'flex', alignItems: 'flex-start', justifyContent: 'space-between', gap: '24px' }}>
            <div style={{ flex: '0 0 auto' }}>
              <Row>
                <Col span={24}>
                  <Space align="center" style={{ marginBottom: '8px' }}>
                    <Title level={2} style={{ margin: 0 }}>{metadata.name}</Title>
                    {spec.ingress?.template && status?.jobStatus?.jobId && (
                  <a
                    href={`https://${spec.ingress.template}/#/job/running/${status.jobStatus.jobId}/overview`}
                    target="_blank"
                    rel="noopener noreferrer"
                    aria-label={`Open Flink UI for ${metadata.name}`}
                  >
                    <HomeOutlined style={{ fontSize: '32px' }} />
                  </a>
                    )}
                  </Space>
                </Col>
              </Row>
              <Row>
                <Col span={24}>
                  <Tag color="blue">{metadata.namespace}</Tag>
                </Col>
              </Row>
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
