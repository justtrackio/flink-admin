import { createContext } from 'react';
import type { FlinkDeployment } from '../api/schema';

export interface DeploymentStreamContextValue {
  deployments: FlinkDeployment[];
  isConnected: boolean;
  error: string | null;
  retry: () => void;
}

export const DeploymentStreamContext = createContext<DeploymentStreamContextValue | undefined>(undefined);
