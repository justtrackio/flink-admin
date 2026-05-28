import { useCallback, useMemo } from 'react';
import type { FlinkDeployment, DeploymentEvent } from '../api/schema';
import { useSseStream } from './useSseStream';

const MIN_BACKOFF_MS = 1_000;
const MAX_BACKOFF_MS = 1_000;
const HEARTBEAT_TIMEOUT_MS = 15_000;
const HEARTBEAT_CHECK_INTERVAL_MS = 5_000;

interface DeploymentStreamState {
  deployments: FlinkDeployment[];
  isConnected: boolean;
  error: string | null;
  retry: () => void;
}

export function useDeploymentStream(): DeploymentStreamState {
  const getInitialState = useCallback(() => new Map<string, FlinkDeployment>(), []);

  const parseMessage = useCallback((data: string): DeploymentEvent | null => {
    if (data === '') {
      return null;
    }

    return JSON.parse(data) as DeploymentEvent;
  }, []);

  const reduceState = useCallback(
    (state: Map<string, FlinkDeployment>, event: DeploymentEvent): Map<string, FlinkDeployment> => {
      const next = new Map(state);

      if (event.type === 'DELETED') {
        next.delete(event.deployment.metadata.uid);
      } else {
        next.set(event.deployment.metadata.uid, event.deployment);
      }

      return next;
    },
    []
  );

  const {
    data: deploymentsMap,
    isConnected,
    error,
    retry,
  } = useSseStream<Map<string, FlinkDeployment>, DeploymentEvent>({
    url: '/api/deployments/watch',
    getInitialState,
    parseMessage,
    reduceState,
    resetStateOnConnect: true,
    minBackoffMs: MIN_BACKOFF_MS,
    maxBackoffMs: MAX_BACKOFF_MS,
    heartbeatTimeoutMs: HEARTBEAT_TIMEOUT_MS,
    heartbeatCheckIntervalMs: HEARTBEAT_CHECK_INTERVAL_MS,
  });

  const deployments = useMemo(() => Array.from(deploymentsMap.values()), [deploymentsMap]);

  return {
    deployments,
    isConnected,
    error,
    retry,
  };
}
