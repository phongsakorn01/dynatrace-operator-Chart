// +kubebuilder:object:generate=true
// +groupName=dynatrace.com
// +versionName=v1alpha2
// +kubebuilder:validation:Optional
package edgeconnect

import (
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EdgeConnectSpec defines the desired state of EdgeConnect.
type EdgeConnectSpec struct { //nolint:revive
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html

	// Adds additional annotations to the EdgeConnect pods
	Annotations map[string]string `json:"annotations,omitempty"`

	// Adds additional labels to the EdgeConnect pods
	Labels map[string]string `json:"labels,omitempty"`

	// Amount of replicas for your EdgeConnect (the default value is: 1)
	// +kubebuilder:default:=1
	Replicas *int32 `json:"replicas"`

	// Node selector to control the selection of nodes for the EdgeConnect pods
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// KubernetesAutomation enables Kubernetes Automation for Workflows
	KubernetesAutomation *KubernetesAutomationSpec `json:"kubernetesAutomation,omitempty"`

	// General configurations for proxy settings.
	// +kubebuilder:validation:Optional
	Proxy *ProxySpec `json:"proxy,omitempty"`

	// Overrides the default image
	ImageRef ImageRefSpec `json:"imageRef,omitempty"`

	// Location of the Dynatrace API to connect to, including your specific environment UUID
	// +kubebuilder:validation:Required
	ApiServer string `json:"apiServer"`

	// Restrict outgoing HTTP requests to your internal resources to specified hosts
	// +kubebuilder:example:="internal.example.org,*.dev.example.org"
	HostRestrictions []string `json:"hostRestrictions,omitempty"`

	// Pull secret for your private registry
	CustomPullSecret string `json:"customPullSecret,omitempty"`

	// Adds custom root certificate from a configmap. Put the certificate under certs within your configmap.
	// +kubebuilder:validation:Optional
	CaCertsRef string `json:"caCertsRef,omitempty"`

	// ServiceAccountName that allows EdgeConnect to access the Kubernetes API
	// +kubebuilder:default:=dynatrace-edgeconnect
	ServiceAccountName string `json:"serviceAccountName,omitempty"`

	// EdgeConnect uses the OAuth client to authenticate itself with the Dynatrace platform.
	// +kubebuilder:validation:Required
	OAuth OAuthSpec `json:"oauth"`

	// Defines resources requests and limits for single pods
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`

	// Adds additional environment variables to the EdgeConnect pods
	Env []corev1.EnvVar `json:"env,omitempty"`

	// Sets tolerations for the EdgeConnect pods
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`

	// Sets topology spread constraints for the EdgeConnect pods
	TopologySpreadConstraints []corev1.TopologySpreadConstraint `json:"topologySpreadConstraints,omitempty"`

	// Host patterns to be set in the tenant, only considered when provisioning is enabled.
	// +kubebuilder:validation:Optional
	HostPatterns []string `json:"hostPatterns,omitempty"`

	// Enables automatic restarts of EdgeConnect pods in case a new version is available (the default value is: true)
	// +kubebuilder:default:=true
	AutoUpdate bool `json:"autoUpdate"`
}

type OAuthSpec struct {
	// Name of the secret that holds oauth clientId/secret
	// +kubebuilder:validation:Required
	ClientSecret string `json:"clientSecret"`
	// Token endpoint URL of Dynatrace SSO
	// +kubebuilder:validation:Required
	Endpoint string `json:"endpoint"`
	// URN identifying your account. You get the URN when creating the OAuth client
	// +kubebuilder:validation:Required
	Resource string `json:"resource"`
	// Determines if the operator will create the EdgeConnect and light OAuth client on the cluster using the credentials provided. Requires more scopes than default behavior.
	// +kubebuilder:validation:Optional
	Provisioner bool `json:"provisioner"`
}

type ImageRefSpec struct {
	// Custom EdgeConnect image repository
	// +kubebuilder:example:="docker.io/dynatrace/edgeconnect"
	Repository string `json:"repository,omitempty"`

	// Indicates version of the EdgeConnect image to use
	Tag string `json:"tag,omitempty"`
}

type ProxySpec struct {
	// Server address (hostname or IP address) of the proxy.
	Host string `json:"host,omitempty"`

	// NoProxy represents the NO_PROXY or no_proxy environment
	// variable. It specifies a string that contains comma-separated values
	// specifying hosts that should be excluded from proxying.
	NoProxy string `json:"noProxy,omitempty"`

	// Secret name which contains the username and password used for authentication with the proxy, using the
	// "Basic" HTTP authentication scheme.
	AuthRef string `json:"authRef,omitempty"`

	// Port of the proxy.
	Port uint32 `json:"port,omitempty"`
}

type KubernetesAutomationSpec struct {
	// Enables Kubernetes Automation for Workflows
	Enabled bool `json:"enabled,omitempty"`
}

// EdgeConnectStatus defines the observed state of EdgeConnect.
type EdgeConnectStatus struct { //nolint:revive
	// Defines the current state (Running, Updating, Error, ...)
	DeploymentPhase status.DeploymentPhase `json:"phase,omitempty"`

	// Version used for the Edgeconnect image
	Version status.VersionStatus `json:"version,omitempty"`

	// Indicates when the resource was last updated
	UpdatedTimestamp metav1.Time `json:"updatedTimestamp,omitempty"`

	// kube-system namespace uid
	KubeSystemUID string `json:"kubeSystemUID,omitempty"`

	// Conditions includes status about the current state of the instance
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// SetPhase sets the status phase on the EdgeConnect object.
func (dk *EdgeConnectStatus) SetPhase(phase status.DeploymentPhase) bool {
	upd := phase != dk.DeploymentPhase
	dk.DeploymentPhase = phase

	return upd
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// EdgeConnect is the Schema for the EdgeConnect API
// +k8s:openapi-gen=true
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=edgeconnects,scope=Namespaced,categories=dynatrace,shortName={ec,ecs}
// +kubebuilder:printcolumn:name="ApiServer",type=string,JSONPath=`.spec.apiServer`
// +kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
// +kubebuilder:storageversion
// +kubebuilder:validation:XValidation:rule="size(self.metadata.name) <= 40",reason="FieldValueInvalid",fieldPath=".metadata",message="The length limit for the name of a EdgeConnect is 40, because it is the base for the name of resources related to the EdgeConnect. The limit is necessary because kubernetes uses the name of some resources for the label value, which has a limit of 63 characters."
type EdgeConnect struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Status EdgeConnectStatus `json:"status,omitempty"`
	Spec   EdgeConnectSpec   `json:"spec,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// EdgeConnectList contains a list of EdgeConnect
// +k8s:openapi-gen=true
// +kubebuilder:object:root=true
type EdgeConnectList struct { //nolint:revive
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []EdgeConnect `json:"items"`
}

const (
	KubernetesDefaultDNS     = "kubernetes.default.svc.cluster.local"
	kubernetesHostnameSuffix = "kubernetes-automation"
)

func init() {
	v1alpha2.SchemeBuilder.Register(&EdgeConnect{}, &EdgeConnectList{})
}

func (e *EdgeConnect) HostPatterns() []string {
	if !e.IsK8SAutomationEnabled() {
		return e.Spec.HostPatterns
	}

	var hostPatterns []string

	for _, hostPattern := range e.Spec.HostPatterns {
		if !strings.EqualFold(hostPattern, e.K8sAutomationHostPattern()) {
			hostPatterns = append(hostPatterns, hostPattern)
		}
	}

	hostPatterns = append(hostPatterns, e.K8sAutomationHostPattern())

	return hostPatterns
}

type HostMapping struct {
	From string `json:"from"`
	To   string `json:"to"`
}

func (e *EdgeConnect) HostMappings() []HostMapping {
	hostMappings := make([]HostMapping, 0)
	hostMappings = append(hostMappings, HostMapping{From: e.K8sAutomationHostPattern(), To: KubernetesDefaultDNS})

	return hostMappings
}

func (e *EdgeConnect) K8sAutomationHostPattern() string {
	return e.Name + "." + e.Namespace + "." + e.Status.KubeSystemUID + "." + kubernetesHostnameSuffix
}