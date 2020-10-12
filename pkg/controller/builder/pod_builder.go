package builder

import (
	"strings"

	"github.com/Dynatrace/dynatrace-activegate-operator/pkg/apis/dynatrace/v1alpha1"
	"github.com/Dynatrace/dynatrace-activegate-operator/pkg/dtclient"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func BuildActiveGatePodSpecs(
	activeGateSpec *v1alpha1.ActiveGateSpec,
	tenantInfo *dtclient.TenantInfo) corev1.PodSpec {
	sa := MonitoringServiceAccount
	image := ActivegateImage

	if activeGateSpec.ServiceAccountName != "" {
		activeGateSpec.ServiceAccountName = sa
	}
	if activeGateSpec.Image != "" {
		image = activeGateSpec.Image
	}
	if tenantInfo == nil {
		tenantInfo = &dtclient.TenantInfo{
			ID:        "",
			Token:     "",
			Endpoints: []string{},
		}
	}

	if activeGateSpec.Resources.Requests == nil {
		activeGateSpec.Resources.Requests = corev1.ResourceList{}
	}
	if _, hasCPUResource := activeGateSpec.Resources.Requests[corev1.ResourceCPU]; !hasCPUResource {
		// Set CPU resource to 1 * 10**(-1) Cores, e.g. 100mC
		activeGateSpec.Resources.Requests[corev1.ResourceCPU] = *resource.NewScaledQuantity(1, -1)
	}

	return corev1.PodSpec{
		Containers: []corev1.Container{{
			Name:            ActivegateName,
			Image:           image,
			Resources:       activeGateSpec.Resources,
			ImagePullPolicy: corev1.PullAlways,
			Env:             buildEnvVars(activeGateSpec, tenantInfo),
			Args:            buildArgs(),
		}},
		DNSPolicy:          activeGateSpec.DNSPolicy,
		NodeSelector:       activeGateSpec.NodeSelector,
		ServiceAccountName: sa,
		Affinity:           buildAffinity(),
		Tolerations:        activeGateSpec.Tolerations,
		PriorityClassName:  activeGateSpec.PriorityClassName,
	}
}

func buildArgs() []string {
	return []string{
		DtTenantArg,
		DtTokenArg,
		DtServerArg,
		DtCapabilitiesArg,
	}
}

func buildEnvVars(activeGatePodSpec *v1alpha1.ActiveGateSpec, tenantInfo *dtclient.TenantInfo) []corev1.EnvVar {
	return []corev1.EnvVar{
		{
			Name:  DtTenant,
			Value: tenantInfo.ID,
		},
		{
			Name:  DtToken,
			Value: tenantInfo.Token,
		},
		{
			Name:  DtServer,
			Value: tenantInfo.CommunicationEndpoint,
		},
		{
			Name:  DtCapabilities,
			Value: strings.Join(activeGatePodSpec.Capabilities, Comma),
		},
	}
}

func buildAffinity() *corev1.Affinity {
	return &corev1.Affinity{
		NodeAffinity: &corev1.NodeAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
				NodeSelectorTerms: []corev1.NodeSelectorTerm{
					{
						MatchExpressions: []corev1.NodeSelectorRequirement{
							{
								Key:      KubernetesBetaArch,
								Operator: corev1.NodeSelectorOpIn,
								Values:   []string{AMD64, ARM64},
							},
							{
								Key:      KubernetesBetaOs,
								Operator: corev1.NodeSelectorOpIn,
								Values:   []string{LINUX},
							},
						},
					},
					{
						MatchExpressions: []corev1.NodeSelectorRequirement{
							{
								Key:      KubernetesArch,
								Operator: corev1.NodeSelectorOpIn,
								Values:   []string{AMD64, ARM64},
							},
							{
								Key:      KubernetesOs,
								Operator: corev1.NodeSelectorOpIn,
								Values:   []string{LINUX},
							},
						},
					},
				},
			},
		},
	}
}

func BuildLabels(name string, labels map[string]string) map[string]string {
	result := BuildLabelsForQuery(name)
	for key, value := range labels {
		result[key] = value
	}
	return result
}

// buildLabels returns generic labels based on the name given for a Dynatrace OneAgent
func BuildLabelsForQuery(name string) map[string]string {
	return map[string]string{
		"dynatrace":  "activegate",
		"activegate": name,
	}
}

const (
	ActivegateImage = "612044533526.dkr.ecr.us-east-1.amazonaws.com/activegate:latest"
	ActivegateName  = "dynatrace-activegate-operator"

	MonitoringServiceAccount = "dynatrace-activegate"

	KubernetesArch     = "kubernetes.io/arch"
	KubernetesOs       = "kubernetes.io/os"
	KubernetesBetaArch = "beta.kubernetes.io/arch"
	KubernetesBetaOs   = "beta.kubernetes.io/os"

	AMD64 = "amd64"
	ARM64 = "arm64"
	LINUX = "linux"

	DtTenant       = "DT_TENANT"
	DtServer       = "DT_SERVER"
	DtToken        = "DT_TOKEN"
	DtCapabilities = "DT_CAPABILITIES"

	DtTenantArg       = "--tenant=$(DT_TENANT)"
	DtTokenArg        = "--token=$(DT_TOKEN)"
	DtServerArg       = "--server=$(DT_SERVER)"
	DtCapabilitiesArg = "--enable=$(DT_CAPABILITIES)"

	Comma = ","
)
