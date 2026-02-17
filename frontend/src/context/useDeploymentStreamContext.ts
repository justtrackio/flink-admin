import { useContext } from 'react';
import type { DeploymentStreamContextValue } from './deploymentStreamContext';
import { DeploymentStreamContext } from './deploymentStreamContext';

export function useDeploymentStreamContext(): DeploymentStreamContextValue {
  const context = useContext(DeploymentStreamContext);

  if (!context) {
    throw new Error('useDeploymentStreamContext must be used within DeploymentStreamProvider');
  }

  return context;
}
