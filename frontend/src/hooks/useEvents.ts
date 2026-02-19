import { useEffect, useState } from 'react';
import { apiClient } from '../api/client';
import type { K8sEventsResponse, K8sEvent } from '../api/schema';

interface EventsState {
  data: K8sEvent[] | null;
  isLoading: boolean;
  error: string | null;
  refetch: () => void;
}

/**
 * Hook to fetch Kubernetes events (events.k8s.io/v1) related to a FlinkDeployment.
 * Returns event data, loading state, error state, and a refetch function.
 */
export function useEvents(namespace: string, name: string): EventsState {
  const [data, setData] = useState<K8sEvent[] | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [refetchTrigger, setRefetchTrigger] = useState(0);

  const refetch = () => {
    setRefetchTrigger((prev) => prev + 1);
  };

  useEffect(() => {
    let cancelled = false;

    const fetchEvents = async () => {
      setIsLoading(true);
      setError(null);

      try {
        const result = await apiClient.get<K8sEventsResponse>(
          `/api/deployments/${namespace}/${name}/events`
        );

        if (!cancelled) {
          setData(result.events);
          setIsLoading(false);
        }
      } catch (err) {
        if (!cancelled) {
          setError(err instanceof Error ? err.message : 'Failed to fetch events');
          setIsLoading(false);
        }
      }
    };

    fetchEvents();

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
