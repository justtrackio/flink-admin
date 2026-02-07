import { useEffect, useState } from 'react';
import { apiClient } from '../api/client';
import type { FlinkCheckpointStatistics } from '../api/schema';

interface CheckpointsState {
  data: FlinkCheckpointStatistics | null;
  isLoading: boolean;
  error: string | null;
  refetch: () => void;
}

/**
 * Hook to fetch checkpoint statistics for a deployment from the Flink REST API.
 * Returns checkpoint data, loading state, error state, and a refetch function.
 */
export function useCheckpoints(namespace: string, name: string): CheckpointsState {
  const [data, setData] = useState<FlinkCheckpointStatistics | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [refetchTrigger, setRefetchTrigger] = useState(0);

  const refetch = () => {
    setRefetchTrigger((prev) => prev + 1);
  };

  useEffect(() => {
    let cancelled = false;

    const fetchCheckpoints = async () => {
      setIsLoading(true);
      setError(null);

      try {
        const result = await apiClient.get<FlinkCheckpointStatistics>(
          `/api/deployments/${namespace}/${name}/checkpoints`
        );

        if (!cancelled) {
          setData(result);
          setIsLoading(false);
        }
      } catch (err) {
        if (!cancelled) {
          setError(err instanceof Error ? err.message : 'Failed to fetch checkpoints');
          setIsLoading(false);
        }
      }
    };

    fetchCheckpoints();

    return () => {
      cancelled = true;
    };
  }, [namespace, name, refetchTrigger]);

  return {
    data,
    isLoading,
    error,
    refetch,
  };
}
