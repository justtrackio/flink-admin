import { Button, Popconfirm, Tag } from 'antd';
import { useAdminMode } from '../context/useAdminMode';
import { useDeploymentActions } from '../hooks/useDeploymentActions';
import { useMessageApi } from '../context/MessageContext';

interface DeploymentActionButtonProps {
  namespace: string;
  name: string;
  desiredJobState?: string;
  size?: 'small' | 'middle' | 'large';
}

interface ActionConfig {
  action: 'suspend' | 'resume';
  title: string;
  description: string;
  confirmLabel: string;
  buttonLabel: string;
  danger: boolean;
}

function getActionErrorMessage(error: unknown, fallback: string | null): string {
  if (error && typeof error === 'object' && 'message' in error && typeof error.message === 'string') {
    return error.message;
  }

  if (error instanceof Error) {
    return error.message;
  }

  return fallback ?? 'Failed to update deployment state';
}

function getActionConfig(desiredJobState?: string): ActionConfig | null {
  const normalizedState = desiredJobState?.toUpperCase();

  if (normalizedState === 'RUNNING') {
    return {
      action: 'suspend',
      title: 'Suspend deployment?',
      description: 'This will patch spec.job.state to suspended.',
      confirmLabel: 'Suspend',
      buttonLabel: 'Suspend',
      danger: true,
    };
  }

  if (normalizedState === 'SUSPENDED') {
    return {
      action: 'resume',
      title: 'Resume deployment?',
      description: 'This will patch spec.job.state to running.',
      confirmLabel: 'Resume',
      buttonLabel: 'Resume',
      danger: false,
    };
  }

  return null;
}

export function DeploymentActionButton({ namespace, name, desiredJobState, size = 'middle' }: DeploymentActionButtonProps) {
  const { isAdminMode } = useAdminMode();
  const actionConfig = getActionConfig(desiredJobState);
  const { suspendDeployment, resumeDeployment, isLoading, pendingAction, error } = useDeploymentActions(namespace, name);
  const messageApi = useMessageApi();

  const handleConfirm = async () => {
    if (!actionConfig) {
      return;
    }

    try {
      if (actionConfig.action === 'suspend') {
        await suspendDeployment();
      } else {
        await resumeDeployment();
      }

      void messageApi.success(`${actionConfig.confirmLabel} requested for ${namespace}/${name}`);
    } catch (requestError) {
      void messageApi.error(getActionErrorMessage(requestError, error));
    }
  };

  if (!isAdminMode) {
    return null;
  }

  if (!actionConfig) {
    return <Tag>Action unavailable</Tag>;
  }

  return (
    <Popconfirm
      title={actionConfig.title}
      description={actionConfig.description}
      okText={actionConfig.confirmLabel}
      cancelText="Cancel"
      onConfirm={handleConfirm}
      disabled={isLoading}
    >
      <Button
        danger={actionConfig.danger}
        disabled={isLoading}
        loading={pendingAction === actionConfig.action}
        size={size}
        type={actionConfig.danger ? 'default' : 'primary'}
      >
        {actionConfig.buttonLabel}
      </Button>
    </Popconfirm>
  );
}
