import { useCallback, useState } from 'react';
import { apiClient } from '../api/client';
import type { RecoverDeploymentResponse, UpdateDeploymentStateResponse } from '../api/schema';

type DeploymentAction = 'suspend' | 'resume' | 'recover';

interface DeploymentActionsState {
  suspendDeployment: () => Promise<UpdateDeploymentStateResponse>;
  resumeDeployment: () => Promise<UpdateDeploymentStateResponse>;
  recoverDeployment: (initialSavepointPath: string) => Promise<RecoverDeploymentResponse>;
  isLoading: boolean;
  pendingAction: DeploymentAction | null;
  error: string | null;
}

function getErrorMessage(error: unknown): string {
  if (error && typeof error === 'object' && 'message' in error && typeof error.message === 'string') {
    return error.message;
  }

  if (error instanceof Error) {
    return error.message;
  }

  return 'Failed to update deployment state';
}

export function useDeploymentActions(namespace: string, name: string): DeploymentActionsState {
  const [pendingAction, setPendingAction] = useState<DeploymentAction | null>(null);
  const [error, setError] = useState<string | null>(null);

  const requestAction = useCallback(async (action: DeploymentAction): Promise<UpdateDeploymentStateResponse> => {
    setPendingAction(action);
    setError(null);

    try {
      return await apiClient.post<UpdateDeploymentStateResponse>(`/api/deployments/${namespace}/${name}/${action}`);
    } catch (requestError) {
      const message = getErrorMessage(requestError);
      setError(message);
      throw requestError;
    } finally {
      setPendingAction(null);
    }
  }, [name, namespace]);

  const suspendDeployment = useCallback(() => {
    return requestAction('suspend');
  }, [requestAction]);

  const resumeDeployment = useCallback(() => {
    return requestAction('resume');
  }, [requestAction]);

  const recoverDeployment = useCallback(async (initialSavepointPath: string): Promise<RecoverDeploymentResponse> => {
    setPendingAction('recover');
    setError(null);

    try {
      return await apiClient.post<RecoverDeploymentResponse>(`/api/deployments/${namespace}/${name}/recover`, {
        initialSavepointPath,
      });
    } catch (requestError) {
      const message = getErrorMessage(requestError);
      setError(message);
      throw requestError;
    } finally {
      setPendingAction(null);
    }
  }, [name, namespace]);

  return {
    suspendDeployment,
    resumeDeployment,
    recoverDeployment,
    isLoading: pendingAction !== null,
    pendingAction,
    error,
  };
}
