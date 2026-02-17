import type { ReactNode } from 'react';
import { DeploymentStreamContext } from './deploymentStreamContext';
import { useDeploymentStream } from '../hooks/useDeploymentStream';

export function DeploymentStreamProvider({ children }: { children: ReactNode }) {
  const streamState = useDeploymentStream();
  
  return (
    <DeploymentStreamContext.Provider value={streamState}>
      {children}
    </DeploymentStreamContext.Provider>
  );
}
