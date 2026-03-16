package internal

type DeploymentSelectorInput struct {
	Namespace string `uri:"namespace"`
	Name      string `uri:"name"`
}
