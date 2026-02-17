import { HomeOutlined } from '@ant-design/icons';
import { createFileRoute, Link, useNavigate } from '@tanstack/react-router';
import { Alert, Badge, Button, Card, Space, Table, Tag, Typography } from 'antd';
import type { TableProps } from 'antd';
import type { ColumnsType, FilterValue } from 'antd/es/table/interface';
import { useDeploymentStreamContext } from '../context/useDeploymentStreamContext';
import type { FlinkDeployment } from '../api/schema';
import { DeploymentStatusTag } from '../components/DeploymentStatusTag';
import { JobStatusTag } from '../components/JobStatusTag';
import { formatAge, formatImageTag } from '../utils/format';
import { useMemo } from 'react';

const { Title, Paragraph } = Typography;

interface IndexSearchParams {
  namespace?: string;
  lifecycleState?: string;
  showNotRunning?: boolean;
}

export const Route = createFileRoute('/')({
  component: IndexComponent,
  validateSearch: (search: Record<string, unknown>): IndexSearchParams => ({
    namespace: typeof search.namespace === 'string' ? search.namespace : undefined,
    lifecycleState: typeof search.lifecycleState === 'string' ? search.lifecycleState : undefined,
    showNotRunning: search.showNotRunning === true || search.showNotRunning === 'true' ? true : undefined,
  }),
});

function IndexComponent() {
  const { deployments, isConnected, error, retry } = useDeploymentStreamContext();
  const { namespace, lifecycleState, showNotRunning } = Route.useSearch();
  const navigate = useNavigate({ from: '/' });

  const tableFilters: Record<string, FilterValue | null> = {
    namespace: namespace ? [namespace] : null,
    lifecycleState: lifecycleState ? [lifecycleState] : null,
  };

  // Extract unique namespaces and lifecycle states for filters
  const namespaces = useMemo(() => {
    const unique = new Set(deployments.map((d) => d.metadata.namespace));
    return Array.from(unique).sort();
  }, [deployments]);

  const lifecycleStates = useMemo(() => {
    const unique = new Set(
      deployments
        .map((d) => d.status?.lifecycleState)
        .filter((state): state is string => Boolean(state))
    );
    return Array.from(unique).sort();
  }, [deployments]);

  const notRunningDeployments = useMemo(() => {
    return deployments.filter((deployment) => {
      const jobState = deployment.status?.jobStatus?.state;
      return jobState?.toUpperCase() !== 'RUNNING';
    });
  }, [deployments]);

  const hasNotRunningDeployments = notRunningDeployments.length > 0;

  const effectiveShowNotRunning = Boolean(showNotRunning) && hasNotRunningDeployments;

  const dataSource = useMemo(() => {
    return effectiveShowNotRunning ? notRunningDeployments : deployments;
  }, [deployments, notRunningDeployments, effectiveShowNotRunning]);

  const handleToggleNotRunning = () => {
    if (!showNotRunning) {
      navigate({
        search: {
          showNotRunning: true,
        },
        replace: true,
      });
      return;
    }

    navigate({
      search: {
        namespace,
        lifecycleState,
      },
      replace: true,
    });
  };

  const handleTableChange: TableProps<FlinkDeployment>['onChange'] = (_, filters) => {
    const nextNamespace = Array.isArray(filters.namespace) ? filters.namespace[0] : undefined;
    const nextLifecycleState = Array.isArray(filters.lifecycleState) ? filters.lifecycleState[0] : undefined;

    navigate({
      search: {
        namespace: typeof nextNamespace === 'string' ? nextNamespace : undefined,
        lifecycleState: typeof nextLifecycleState === 'string' ? nextLifecycleState : undefined,
        showNotRunning: showNotRunning || undefined,
      },
      replace: true,
    });
  };

  const columns: ColumnsType<FlinkDeployment> = [
    {
      title: 'Name',
      dataIndex: ['metadata', 'name'],
      key: 'name',
      sorter: (a, b) => a.metadata.name.localeCompare(b.metadata.name),
      defaultSortOrder: 'ascend',
      render: (_, record) => (
        <Space size="small">
          <Link
            to="/deployments/$namespace/$name"
            params={{
              namespace: record.metadata.namespace,
              name: record.metadata.name,
            }}
            search={{
              fromNamespace: namespace,
              fromLifecycleState: lifecycleState,
              fromShowNotRunning: showNotRunning,
            }}
            style={{ fontWeight: 'bold' }}
          >
            {record.metadata.name}
          </Link>
          {record.spec.ingress?.template && record.status?.jobStatus?.jobId && (
            <a
              href={`https://${record.spec.ingress.template}/#/job/running/${record.status.jobStatus.jobId}/overview`}
              target="_blank"
              rel="noopener noreferrer"
              aria-label={`Open Flink UI for ${record.metadata.name}`}
            >
              <HomeOutlined />
            </a>
          )}
        </Space>
      ),
    },
    {
      title: 'Namespace',
      dataIndex: ['metadata', 'namespace'],
      key: 'namespace',
      filters: namespaces.map((ns) => ({ text: ns, value: ns })),
      onFilter: (value, record) => record.metadata.namespace === value,
      filterMultiple: false,
      filteredValue: tableFilters.namespace || null,
      render: (value: string) => (
        <Tag
          color="blue"
          style={{ cursor: 'pointer' }}
          onClick={() => {
            const nextNamespace = namespace === value ? undefined : value;

            navigate({
              search: {
                namespace: nextNamespace,
                lifecycleState,
                showNotRunning: showNotRunning || undefined,
              },
              replace: true,
            });
          }}
        >
          {value}
        </Tag>
      ),
    },
    {
      title: 'Lifecycle State',
      dataIndex: ['status', 'lifecycleState'],
      key: 'lifecycleState',
      filters: lifecycleStates.map((state) => ({ text: state, value: state })),
      onFilter: (value, record) => record.status?.lifecycleState === value,
      filterMultiple: false,
      filteredValue: tableFilters.lifecycleState || null,
      render: (state: string) => state ? <DeploymentStatusTag status={state} /> : <Tag>N/A</Tag>,
    },
    {
      title: 'Job State',
      dataIndex: ['status', 'jobStatus', 'state'],
      key: 'jobState',
      render: (state: string) => <JobStatusTag status={state} />,
    },
    {
      title: 'Flink Version',
      dataIndex: ['spec', 'flinkVersion'],
      key: 'flinkVersion',
    },
    {
      title: 'Image',
      dataIndex: ['spec', 'image'],
      key: 'image',
      render: (image: string) => <code style={{ fontSize: '12px' }}>{formatImageTag(image)}</code>,
    },
    {
      title: 'Parallelism',
      dataIndex: ['spec', 'job', 'parallelism'],
      key: 'parallelism',
      align: 'right',
    },
    {
      title: 'JM Resources',
      key: 'jmResources',
      render: (_, record) => {
        const { cpu, memory } = record.spec.jobManager.resource;
        return `${cpu} CPU / ${memory}`;
      },
    },
    {
      title: 'TM Resources',
      key: 'tmResources',
      render: (_, record) => {
        const { cpu, memory } = record.spec.taskManager.resource;
        return `${cpu} CPU / ${memory}`;
      },
    },
    {
      title: 'Age',
      dataIndex: ['metadata', 'creationTimestamp'],
      key: 'age',
      sorter: (a, b) => new Date(a.metadata.creationTimestamp).getTime() - new Date(b.metadata.creationTimestamp).getTime(),
      render: (timestamp: string) => formatAge(timestamp),
    },
  ];

  return (
    <Card>
      <Space direction="vertical" size="middle" style={{ width: '100%' }}>
        <Space align="center" style={{ justifyContent: 'space-between', width: '100%' }}>
          <div>
            <Title level={2} style={{ margin: 0 }}>Flink Deployments</Title>
            <Paragraph style={{ margin: 0 }}>
              Real-time view of FlinkDeployment resources
            </Paragraph>
          </div>
          <Space>
            <Badge
              status={isConnected ? 'success' : 'error'}
              text={isConnected ? 'Connected' : 'Disconnected'}
            />
            {deployments.length > 0 && (
              <Tag color="blue">{deployments.length} deployment{deployments.length !== 1 ? 's' : ''}</Tag>
            )}
          </Space>
        </Space>

        {error && (
          <Alert
            type="warning"
            message="Connection Issue"
            description={error}
            showIcon
            action={
              <Button size="small" onClick={retry}>
                Reconnect
              </Button>
            }
          />
        )}

        {notRunningDeployments.length > 0 && (
          <Alert
            type={showNotRunning ? 'info' : 'warning'}
            banner
            showIcon
            message={
              showNotRunning
                ? `Showing ${notRunningDeployments.length} not running job${
                    notRunningDeployments.length !== 1 ? 's' : ''
                  }`
                : `${notRunningDeployments.length} job${
                    notRunningDeployments.length !== 1 ? 's' : ''
                  } not running`
            }
            action={
              <Button
                size="small"
                type={showNotRunning ? 'primary' : 'default'}
                onClick={handleToggleNotRunning}
              >
                {showNotRunning ? 'Show All' : 'Show Not Running'}
              </Button>
            }
          />
        )}

        <Table<FlinkDeployment>
          rowKey={(record) => record.metadata.uid}
          columns={columns}
          dataSource={dataSource}
          onChange={handleTableChange}
          pagination={{
            defaultPageSize: 100,
            showSizeChanger: true,
            showTotal: (total) => `Total ${total} deployments`,
          }}
          size="middle"
        />
      </Space>
    </Card>
  );
}
