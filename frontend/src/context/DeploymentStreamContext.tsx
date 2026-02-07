import { createContext, useContext, type ReactNode } from 'react';
import { useDeploymentStream } from '../hooks/useDeploymentStream';
import type { FlinkDeployment } from '../api/schema';

interface DeploymentStreamContextValue {
  deployments: FlinkDeployment[];
  isConnected: boolean;
  error: string | null;
  retry: () => void;
}

const DeploymentStreamContext = createContext<DeploymentStreamContextValue | undefined>(undefined);

export function DeploymentStreamProvider({ children }: { children: ReactNode }) {
  const streamState = useDeploymentStream();
  
  return (
    <DeploymentStreamContext.Provider value={streamState}>
      {children}
    </DeploymentStreamContext.Provider>
  );
}

export function useDeploymentStreamContext(): DeploymentStreamContextValue {
  const context = useContext(DeploymentStreamContext);
  
  if (!context) {
    throw new Error('useDeploymentStreamContext must be used within DeploymentStreamProvider');
  }
  
  return context;
}
