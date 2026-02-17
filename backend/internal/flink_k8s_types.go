package internal

import (
	"encoding/json"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// FlinkDeployment represents the FlinkDeployment custom resource (group flink.apache.org, version v1beta1).
// Only fields used by deployment.yml are modeled here.
type FlinkDeployment struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              FlinkDeploymentSpec   `json:"spec,omitempty"`
	Status            FlinkDeploymentStatus `json:"status,omitempty"`
}

func (d *FlinkDeployment) GetStatusGroup() string {
	switch d.Status.JobStatus.State {
	case "CREATED", "RUNNING", "RESTARTING", "RECONCILING":
		return "running"
	case "FINISHED":
		return "finished"
	case "FAILED", "CANCELED":
		return "failed"
	default:
		return "unknown"
	}
}

type FlinkDeploymentSpec struct {
	FlinkConfiguration map[string]any              `json:"flinkConfiguration,omitempty"`
	FlinkVersion       string                      `json:"flinkVersion,omitempty"`
	Image              string                      `json:"image,omitempty"`
	ImagePullPolicy    string                      `json:"imagePullPolicy,omitempty"`
	Ingress            *FlinkDeploymentIngress     `json:"ingress,omitempty"`
	JobManager         *FlinkDeploymentJobManager  `json:"jobManager,omitempty"`
	ServiceAccount     string                      `json:"serviceAccount,omitempty"`
	TaskManager        *FlinkDeploymentTaskManager `json:"taskManager,omitempty"`
	Job                *FlinkDeploymentJob         `json:"job,omitempty"`
}

type FlinkDeploymentIngress struct {
	Annotations map[string]string `json:"annotations,omitempty"`
	ClassName   string            `json:"className,omitempty"`
	Template    string            `json:"template,omitempty"`
}

type FlinkDeploymentJobManager struct {
	PodTemplate *FlinkDeploymentPodTemplate `json:"podTemplate,omitempty"`
	Replicas    int                         `json:"replicas,omitempty"`
	Resource    *FlinkDeploymentResource    `json:"resource,omitempty"`
}

type FlinkDeploymentTaskManager struct {
	PodTemplate *FlinkDeploymentPodTemplate `json:"podTemplate,omitempty"`
	Resource    *FlinkDeploymentResource    `json:"resource,omitempty"`
}

type FlinkDeploymentPodTemplate struct {
	Metadata *FlinkDeploymentPodTemplateMetadata `json:"metadata,omitempty"`
	Spec     *FlinkDeploymentPodTemplateSpec     `json:"spec,omitempty"`
}

type FlinkDeploymentPodTemplateMetadata struct {
	Annotations map[string]string `json:"annotations,omitempty"`
	Name        string            `json:"name,omitempty"`
}

type FlinkDeploymentPodTemplateSpec struct {
	NodeSelector map[string]string           `json:"nodeSelector,omitempty"`
	Tolerations  []FlinkDeploymentToleration `json:"tolerations,omitempty"`
	Containers   []FlinkDeploymentContainer  `json:"containers,omitempty"`
}

type FlinkDeploymentContainer struct {
	Name      string                             `json:"name,omitempty"`
	Resources *FlinkDeploymentContainerResources `json:"resources,omitempty"`
}

type FlinkDeploymentContainerResources struct {
	Requests map[string]string `json:"requests,omitempty"`
}

type FlinkDeploymentToleration struct {
	Effect string `json:"effect,omitempty"`
	Key    string `json:"key,omitempty"`
	Value  string `json:"value,omitempty"`
}

type FlinkDeploymentResource struct {
	CPU    float32 `json:"cpu,omitempty"`
	Memory string  `json:"memory,omitempty"`
}

type FlinkDeploymentJob struct {
	JarURI      string   `json:"jarURI,omitempty"`
	EntryClass  string   `json:"entryClass,omitempty"`
	Args        []string `json:"args,omitempty"`
	Parallelism int      `json:"parallelism,omitempty"`
	UpgradeMode string   `json:"upgradeMode,omitempty"`
	State       string   `json:"state,omitempty"`
}

type FlinkDeploymentStatus struct {
	ClusterInfo                *FlinkClusterInfo  `json:"clusterInfo,omitempty"`
	JobManagerDeploymentStatus string             `json:"jobManagerDeploymentStatus,omitempty"`
	JobStatus                  FlinkJobStatus     `json:"jobStatus,omitempty"`
	LifecycleState             string             `json:"lifecycleState,omitempty"`
	Conditions                 []metav1.Condition `json:"conditions,omitempty"`
}

type FlinkClusterInfo struct {
	FlinkRevision string `json:"flink-revision,omitempty"`
	FlinkVersion  string `json:"flink-version,omitempty"`
	TotalCPU      string `json:"total-cpu,omitempty"`
	TotalMemory   string `json:"total-memory,omitempty"`
}

type FlinkJobStatus struct {
	CheckpointInfo *FlinkCheckpointInfo `json:"checkpointInfo,omitempty"`
	SavepointInfo  *FlinkSavepointInfo  `json:"savepointInfo,omitempty"`
	JobId          string               `json:"jobId,omitempty"`
	State          string               `json:"state,omitempty"`
	StartTime      string               `json:"startTime,omitempty"`
	UpdateTime     string               `json:"updateTime,omitempty"`
}

type FlinkCheckpointInfo struct {
	LastPeriodicCheckpointTimestamp int64 `json:"lastPeriodicCheckpointTimestamp,omitempty"`
}

type FlinkSavepointInfo struct {
	LastPeriodicSavepointTimestamp int64    `json:"lastPeriodicSavepointTimestamp,omitempty"`
	SavepointHistory               []string `json:"savepointHistory,omitempty"`
}

// FlinkDeploymentList for list operations (optional completeness).
type FlinkDeploymentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []FlinkDeployment `json:"items"`
}

func FromUnstructured(val any) (*FlinkDeployment, error) {
	var ok bool
	var err error
	var object *unstructured.Unstructured
	var data []byte

	if object, ok = val.(*unstructured.Unstructured); !ok {
		return nil, fmt.Errorf("unexpected object type: %T", val)
	}

	if data, err = object.MarshalJSON(); err != nil {
		return nil, fmt.Errorf("could not marshal object: %w", err)
	}

	deployment := &FlinkDeployment{}
	if err = json.Unmarshal(data, deployment); err != nil {
		return nil, fmt.Errorf("could not unmarshal object: %w", err)
	}

	return deployment, nil
}
