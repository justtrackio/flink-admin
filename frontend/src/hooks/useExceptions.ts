import { useEffect, useState } from 'react';
import { apiClient } from '../api/client';
import type { FlinkJobExceptions } from '../api/schema';

interface ExceptionsState {
  data: FlinkJobExceptions | null;
  isLoading: boolean;
  error: string | null;
  refetch: () => void;
}

/**
 * Hook to fetch Flink job exception history from the Flink REST API.
 * Returns the exception data, loading state, error state, and a refetch function.
 */
export function useExceptions(namespace: string, name: string): ExceptionsState {
  const [data, setData] = useState<FlinkJobExceptions | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [refetchTrigger, setRefetchTrigger] = useState(0);

  const refetch = () => {
    setRefetchTrigger((prev) => prev + 1);
  };

  useEffect(() => {
    let cancelled = false;

    const fetchExceptions = async () => {
      setIsLoading(true);
      setError(null);

      try {
        const result = await apiClient.get<FlinkJobExceptions>(
          `/api/deployments/${namespace}/${name}/exceptions`
        );

        if (!cancelled) {
          setData(result);
          setIsLoading(false);
        }
      } catch (err) {
        if (!cancelled) {
          setError(err instanceof Error ? err.message : 'Failed to fetch exceptions');
          setIsLoading(false);
        }
      }
    };

    fetchExceptions();

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
