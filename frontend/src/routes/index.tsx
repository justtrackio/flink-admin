import { HomeOutlined  } from '@ant-design/icons';
import { createFileRoute, Link, useNavigate } from '@tanstack/react-router';
import { Alert, Badge, Button, Card, Space, Table, Tabs, Tag, Typography } from 'antd';
import type { TableProps } from 'antd';
import type { ColumnType, ColumnsType } from 'antd/es/table/interface';
import { useDeploymentStreamContext } from '../context/useDeploymentStreamContext';
import { useAdminMode } from '../context/useAdminMode';
import type { FlinkDeployment } from '../api/schema';
import { DeploymentActionButton } from '../components/DeploymentActionButton';
import { DeploymentStatusTag } from '../components/DeploymentStatusTag';
import { JobStatusTag } from '../components/JobStatusTag';
import { formatAge, formatImageTag } from '../utils/format';
import { useMemo } from 'react';

const { Title } = Typography;

type DeploymentView = 'all' | 'not-running';

interface IndexSearchParams {
  namespace?: string;
  lifecycleState?: string;
  view?: DeploymentView;
}

export const Route = createFileRoute('/')({
  component: IndexComponent,
  validateSearch: (search: Record<string, unknown>): IndexSearchParams => ({
    namespace: typeof search.namespace === 'string' ? search.namespace : undefined,
    lifecycleState: typeof search.lifecycleState === 'string' ? search.lifecycleState : undefined,
    view:
      search.view === 'all' || search.view === 'not-running'
        ? search.view
        : search.showNotRunning === true || search.showNotRunning === 'true'
          ? 'not-running'
          : undefined,
  }),
});

function IndexComponent() {
  const { deployments, isConnected, error, retry } = useDeploymentStreamContext();
  const { isAdminMode } = useAdminMode();
  const { namespace, lifecycleState, view } = Route.useSearch();
  const navigate = useNavigate({ from: '/' });
  const activeView: DeploymentView = view ?? 'all';
  const isNamespaceFilterActive = activeView === 'all';
  const activeNamespace = isNamespaceFilterActive ? namespace : undefined;

  const tableKey = `filters:${activeView}:${activeNamespace ?? 'all'}:${lifecycleState ?? 'all'}`;

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
      const upperJobState = deployment.status?.jobStatus?.state?.toUpperCase();
      return upperJobState !== 'RUNNING' && upperJobState !== 'FINISHED';
    });
  }, [deployments]);

  const hasNotRunningDeployments = notRunningDeployments.length > 0;

  const dataSource = useMemo(() => {
    return activeView === 'not-running' ? notRunningDeployments : deployments;
  }, [activeView, deployments, notRunningDeployments]);

  const handleViewChange = (nextView: string) => {
    navigate({
      search: {
        namespace,
        lifecycleState,
        view: nextView === 'not-running' ? 'not-running' : 'all',
      },
      replace: true,
    });
  };

  const handleTableChange: TableProps<FlinkDeployment>['onChange'] = (_, filters) => {
    const nextNamespace = Array.isArray(filters.namespace) ? filters.namespace[0] : undefined;
    const nextLifecycleState = Array.isArray(filters.lifecycleState) ? filters.lifecycleState[0] : undefined;

    navigate({
      search: {
        namespace:
          activeView === 'all' && typeof nextNamespace === 'string'
            ? nextNamespace
            : namespace,
        lifecycleState: typeof nextLifecycleState === 'string' ? nextLifecycleState : undefined,
        view: activeView,
      },
      replace: true,
    });
  };

  const getRowClassName = (record: FlinkDeployment): string => {
    return record.spec.job.state?.toUpperCase() === 'SUSPENDED' ? 'deployment-row-suspended' : '';
  };

  const actionColumn: ColumnType<FlinkDeployment> = {
    title: 'Action',
    key: 'action',
    align: 'center',
    render: (_, record) => (
      <DeploymentActionButton
        namespace={record.metadata.namespace}
        name={record.metadata.name}
        desiredJobState={record.spec.job.state}
        size="small"
      />
    ),
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
              fromView: activeView,
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
      filters: isNamespaceFilterActive ? namespaces.map((ns) => ({ text: ns, value: ns })) : undefined,
      onFilter: isNamespaceFilterActive ? (value, record) => record.metadata.namespace === value : undefined,
      filterMultiple: false,
      defaultFilteredValue: activeNamespace ? [activeNamespace] : null,
      render: (value: string) => (
        <Tag
          color="blue"
          style={{ cursor: isNamespaceFilterActive ? 'pointer' : 'default' }}
          onClick={() => {
            if (!isNamespaceFilterActive) {
              return;
            }

            const nextNamespace = namespace === value ? undefined : value;

            navigate({
              search: {
                namespace: nextNamespace,
                lifecycleState,
                view: activeView,
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
      defaultFilteredValue: lifecycleState ? [lifecycleState] : null,
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
    ...(isAdminMode ? [actionColumn] : []),
  ];

  return (
    <Card>
      <Space direction="vertical" size="middle" style={{ width: '100%' }}>
        <Space align="center" style={{ justifyContent: 'space-between', width: '100%' }}>
          <div>
            <Title level={2} style={{ margin: 0 }}>Flink Deployments</Title>
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

        <Tabs
          activeKey={activeView}
          onChange={handleViewChange}
          items={[
            {
              key: 'all',
              label: `All Deployments (${deployments.length})`,
            },
            {
              key: 'not-running',
              label: hasNotRunningDeployments
                ? `Not Running (${notRunningDeployments.length})`
                : 'Not Running',
            },
          ]}
        />

        <Table<FlinkDeployment>
          key={tableKey}
          rowKey={(record) => record.metadata.uid}
          rowClassName={getRowClassName}
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
