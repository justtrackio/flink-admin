import { useEffect, useState } from 'react';
import { apiClient } from '../api/client';
import type { StorageCheckpointMetadataResponse } from '../api/schema';

interface StorageCheckpointMetadataState {
  data: StorageCheckpointMetadataResponse | null;
  isLoading: boolean;
  error: string | null;
  refetch: () => void;
}

export function useStorageCheckpointMetadata(
  namespace: string,
  name: string,
  entryType: string,
  jobId: string,
  entryName: string
): StorageCheckpointMetadataState {
  const [data, setData] = useState<StorageCheckpointMetadataResponse | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [refetchTrigger, setRefetchTrigger] = useState(0);

  const refetch = () => {
    setRefetchTrigger((prev) => prev + 1);
  };

  useEffect(() => {
    let cancelled = false;

    const fetchStorageCheckpointMetadata = async () => {
      setIsLoading(true);
      setError(null);

      try {
        const result = await apiClient.get<StorageCheckpointMetadataResponse>(
          `/api/deployments/${namespace}/${name}/storage-checkpoints/${entryType}/${jobId}/${entryName}/metadata`
        );

        if (!cancelled) {
          setData(result);
          setIsLoading(false);
        }
      } catch (err) {
        if (!cancelled) {
          setError(err instanceof Error ? err.message : 'Failed to fetch storage checkpoint metadata');
          setIsLoading(false);
        }
      }
    };

    fetchStorageCheckpointMetadata();

    return () => {
      cancelled = true;
    };
  }, [namespace, name, entryType, jobId, entryName, refetchTrigger]);

  return { data, isLoading, error, refetch };
}
