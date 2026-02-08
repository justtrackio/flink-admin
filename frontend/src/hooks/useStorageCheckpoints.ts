import { useEffect, useState } from 'react';
import { apiClient } from '../api/client';
import type { StorageCheckpointsResponse } from '../api/schema';

interface StorageCheckpointsState {
  data: StorageCheckpointsResponse | null;
  isLoading: boolean;
  error: string | null;
  refetch: () => void;
}

export function useStorageCheckpoints(namespace: string, name: string): StorageCheckpointsState {
  const [data, setData] = useState<StorageCheckpointsResponse | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [refetchTrigger, setRefetchTrigger] = useState(0);

  const refetch = () => {
    setRefetchTrigger((prev) => prev + 1);
  };

  useEffect(() => {
    let cancelled = false;

    const fetchStorageCheckpoints = async () => {
      setIsLoading(true);
      setError(null);

      try {
        const result = await apiClient.get<StorageCheckpointsResponse>(
          `/api/deployments/${namespace}/${name}/storage-checkpoints`
        );

        if (!cancelled) {
          setData(result);
          setIsLoading(false);
        }
      } catch (err) {
        if (!cancelled) {
          setError(err instanceof Error ? err.message : 'Failed to fetch storage checkpoints');
          setIsLoading(false);
        }
      }
    };

    fetchStorageCheckpoints();

    return () => {
      cancelled = true;
    };
  }, [namespace, name, refetchTrigger]);

  return { data, isLoading, error, refetch };
}
