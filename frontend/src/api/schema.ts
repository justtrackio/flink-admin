// FlinkDeployment CRD types for SSE streaming

export interface FlinkDeploymentMetadata {
  name: string;
  namespace: string;
  uid: string;
  creationTimestamp: string;
  labels?: Record<string, string>;
}

export interface FlinkDeploymentResource {
  cpu: number;
  memory: string;
}

export interface FlinkDeploymentJob {
  jarURI: string;
  entryClass: string;
  args?: string[];
  parallelism: number;
  state: string;
  upgradeMode: string;
}

export interface FlinkDeploymentJobManager {
  replicas: number;
  resource: FlinkDeploymentResource;
}

export interface FlinkDeploymentTaskManager {
  resource: FlinkDeploymentResource;
}

export interface FlinkDeploymentSpec {
  image: string;
  flinkVersion: string;
  flinkConfiguration?: Record<string, string>;
  job: FlinkDeploymentJob;
  jobManager: FlinkDeploymentJobManager;
  taskManager: FlinkDeploymentTaskManager;
}

export interface FlinkClusterInfo {
  'flink-version': string;
  'flink-revision': string;
  'total-cpu': string;
  'total-memory': string;
}

export interface FlinkCheckpointInfo {
  lastPeriodicCheckpointTimestamp?: number;
}

export interface FlinkSavepointInfo {
  lastPeriodicSavepointTimestamp?: number;
  savepointHistory?: string[];
}

export interface FlinkJobStatus {
  state: string;
  startTime?: string;
  updateTime?: string;
  checkpointInfo?: FlinkCheckpointInfo;
  savepointInfo?: FlinkSavepointInfo;
}

export interface FlinkDeploymentStatus {
  lifecycleState: string;
  jobManagerDeploymentStatus?: string;
  jobStatus?: FlinkJobStatus;
  clusterInfo?: FlinkClusterInfo;
}

export interface FlinkDeployment {
  kind: string;
  apiVersion: string;
  metadata: FlinkDeploymentMetadata;
  spec: FlinkDeploymentSpec;
  status?: FlinkDeploymentStatus;
}

export type DeploymentEventType = 'ADDED' | 'MODIFIED' | 'DELETED';

export interface DeploymentEvent {
  type: DeploymentEventType;
  deployment: FlinkDeployment;
}
