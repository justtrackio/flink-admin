import { useMemo } from 'react';
import { useDeploymentStreamContext } from '../context/useDeploymentStreamContext';
import type { FlinkDeployment } from '../api/schema';

/**
 * Hook to retrieve a single deployment by namespace and name.
 * Returns undefined if the deployment is not found.
 */
export function useDeployment(namespace: string, name: string): FlinkDeployment | undefined {
  const { deployments } = useDeploymentStreamContext();
  
  return useMemo(() => {
    return deployments.find(
      (d) => d.metadata.namespace === namespace && d.metadata.name === name
    );
  }, [deployments, namespace, name]);
}
