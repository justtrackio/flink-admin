import { createFileRoute } from '@tanstack/react-router';
import { Descriptions, Table } from 'antd';
import { useMemo } from 'react';
import { useDeployment } from '../hooks/useDeployment';

interface ConfigurationTableRow {
  key: string;
  configKey: string;
  configValue: string;
}

function isConfigurationObject(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
}

function formatConfigurationValue(value: unknown): string {
  if (value === null || value === undefined) {
    return '';
  }

  if (typeof value === 'string') {
    return value;
  }

  if (typeof value === 'number' || typeof value === 'boolean') {
    return String(value);
  }

  return JSON.stringify(value) ?? '';
}

function flattenConfigurationEntries(
  configuration: Record<string, unknown>,
  prefix = ''
): ConfigurationTableRow[] {
  const rows: ConfigurationTableRow[] = [];

  for (const [configKey, configValue] of Object.entries(configuration)) {
    const nestedKey = prefix ? `${prefix}.${configKey}` : configKey;

    if (isConfigurationObject(configValue)) {
      rows.push(...flattenConfigurationEntries(configValue, nestedKey));
      continue;
    }

    rows.push({
      key: nestedKey,
      configKey: nestedKey,
      configValue: formatConfigurationValue(configValue),
    });
  }

  return rows;
}

export const Route = createFileRoute('/deployments/$namespace/$name/')({
  component: DeploymentDetailsComponent,
});

function DeploymentDetailsComponent() {
  const { namespace, name } = Route.useParams();
  const deployment = useDeployment(namespace, name);

  const jobArgsTableData = useMemo(() => {
    const args = deployment?.spec.job.args ?? [];
    const rows: Array<{ key: string; argKey: string; argValue: string }> = [];
    for (let idx = 0; idx < args.length; idx += 2) {
      const argKey = args[idx];
      const argValue = args[idx + 1] ?? '';
      rows.push({ key: `${idx}-${argKey}`, argKey, argValue });
    }
    return rows;
  }, [deployment?.spec.job.args]);

  const flinkConfigurationTableData = useMemo(() => {
    return flattenConfigurationEntries(deployment?.spec.flinkConfiguration ?? {}).sort((first, second) =>
      first.configKey.localeCompare(second.configKey)
    );
  }, [deployment?.spec.flinkConfiguration]);

  if (!deployment) {
    return null; // Parent handles "not found"
  }

  const { spec } = deployment;

  const jobArgs = spec.job.args ?? [];
  const hasFlinkConfiguration = flinkConfigurationTableData.length > 0;

  const jobArgsColumns = [
    {
      title: 'Key',
      dataIndex: 'argKey',
      key: 'argKey',
      render: (value: string) => <code style={{ fontSize: '12px' }}>{value}</code>,
    },
    {
      title: 'Value',
      dataIndex: 'argValue',
      key: 'argValue',
      render: (value: string) => <code style={{ fontSize: '12px' }}>{value}</code>,
    },
  ];

  const flinkConfigurationColumns = [
    {
      title: 'Key',
      dataIndex: 'configKey',
      key: 'configKey',
      render: (value: string) => <code style={{ fontSize: '12px' }}>{value}</code>,
    },
    {
      title: 'Value',
      dataIndex: 'configValue',
      key: 'configValue',
      render: (value: string) => <code style={{ fontSize: '12px' }}>{value}</code>,
    },
  ];

  return (
    <Descriptions column={2} bordered size="small">
      <Descriptions.Item label="Image" span={2}>
        <code style={{ fontSize: '12px' }}>{spec.image}</code>
      </Descriptions.Item>
      <Descriptions.Item label="Flink Version" span={2}>
        {spec.flinkVersion}
      </Descriptions.Item>
      <Descriptions.Item label="Parallelism" span={2}>
        {spec.job.parallelism}
      </Descriptions.Item>
      <Descriptions.Item label="Entry Class" span={2}>
        <code style={{ fontSize: '12px' }}>{spec.job.entryClass}</code>
      </Descriptions.Item>
      <Descriptions.Item label="JAR URI" span={2}>
        <code style={{ fontSize: '12px' }}>{spec.job.jarURI}</code>
      </Descriptions.Item>
      <Descriptions.Item label="Upgrade Mode" span={2}>
        {spec.job.upgradeMode}
      </Descriptions.Item>
      <Descriptions.Item label="Job State (Spec)" span={2}>
        {spec.job.state}
      </Descriptions.Item>
      {jobArgs.length > 0 && (
        <Descriptions.Item label="Job Args" span={2}>
          <Table
            columns={jobArgsColumns}
            dataSource={jobArgsTableData}
            pagination={false}
            size="small"
          />
        </Descriptions.Item>
      )}
      {hasFlinkConfiguration && (
        <Descriptions.Item label="Flink Configuration" span={2}>
          <Table
            columns={flinkConfigurationColumns}
            dataSource={flinkConfigurationTableData}
            pagination={false}
            size="small"
          />
        </Descriptions.Item>
      )}
      <Descriptions.Item label="Job Manager Resources" span={2}>
        {spec.jobManager.resource.cpu} CPU / {spec.jobManager.resource.memory}
        {' '}({spec.jobManager.replicas} {spec.jobManager.replicas === 1 ? 'replica' : 'replicas'})
      </Descriptions.Item>
      <Descriptions.Item label="Task Manager Resources" span={2}>
        {spec.taskManager.resource.cpu} CPU / {spec.taskManager.resource.memory}
      </Descriptions.Item>
    </Descriptions>
  );
}
