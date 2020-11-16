package builder

import (
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/apis/dynatrace/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func BuildActiveGatePodSpecs(instance *v1alpha1.DynaKube, kubeSystemUID types.UID) (corev1.PodSpec, error) {
	sa := MonitoringServiceAccount
	image := ""
	activeGateSpec := &instance.Spec.KubernetesMonitoringSpec

	if activeGateSpec.ServiceAccountName != "" {
		sa = activeGateSpec.ServiceAccountName
	}
	if activeGateSpec.Image != "" {
		image = activeGateSpec.Image
	}

	if activeGateSpec.Resources.Requests == nil {
		activeGateSpec.Resources.Requests = corev1.ResourceList{}
	}
	if _, hasCPUResource := activeGateSpec.Resources.Requests[corev1.ResourceCPU]; !hasCPUResource {
		// Set CPU resource to 1 * 10**(-1) Cores, e.g. 100mC
		activeGateSpec.Resources.Requests[corev1.ResourceCPU] = *resource.NewScaledQuantity(1, -1)
	}

	p := corev1.PodSpec{
		Containers: []corev1.Container{{
			Name:            ActivegateName,
			Image:           image,
			Resources:       activeGateSpec.Resources,
			ImagePullPolicy: corev1.PullAlways,
			Env:             buildEnvVars(instance, kubeSystemUID),
			Args:            buildArgs(),
			ReadinessProbe:  buildReadinessProbe(),
			LivenessProbe:   buildLivenessProbe(),
		}},
		DNSPolicy:          activeGateSpec.DNSPolicy,
		NodeSelector:       activeGateSpec.NodeSelector,
		ServiceAccountName: sa,
		Affinity:           buildAffinity(),
		Tolerations:        activeGateSpec.Tolerations,
		PriorityClassName:  activeGateSpec.PriorityClassName,
	}

	err := preparePodSpecImmutableImage(&p, instance)
	if err != nil {
		return p, err
	}

	return p, nil
}

func buildLivenessProbe() *corev1.Probe {
	return &corev1.Probe{
		Handler: corev1.Handler{
			HTTPGet: &corev1.HTTPGetAction{
				Path:   "/rest/state",
				Port:   intstr.IntOrString{IntVal: 9999},
				Scheme: "HTTPS",
			},
		},
		InitialDelaySeconds: 30,
		PeriodSeconds:       30,
		FailureThreshold:    2,
	}
}

func buildReadinessProbe() *corev1.Probe {
	return &corev1.Probe{
		Handler: corev1.Handler{
			HTTPGet: &corev1.HTTPGetAction{
				Path:   "/rest/health",
				Port:   intstr.IntOrString{IntVal: 9999},
				Scheme: "HTTPS",
			},
		},
		InitialDelaySeconds: 30,
		PeriodSeconds:       15,
		FailureThreshold:    3,
	}
}

func buildArgs() []string {
	return []string{
		DtCapabilitiesArg,
	}
}

func buildEnvVars(instance *v1alpha1.DynaKube, kubeSystemUID types.UID) []corev1.EnvVar {
	var capabilities []string

	if instance.Spec.KubernetesMonitoringSpec.Enabled {
		capabilities = append(capabilities, "kubernetes_monitoring")
	}

	return []corev1.EnvVar{
		{
			Name:  DtCapabilities,
			Value: strings.Join(capabilities, Comma),
		},
		{
			Name:  DtIdSeedNamespace,
			Value: instance.Namespace,
		},
		{
			Name:  DtIdSeedClusterId,
			Value: string(kubeSystemUID),
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

func preparePodSpecImmutableImage(p *corev1.PodSpec, instance *v1alpha1.DynaKube) error {
	pullSecretName := instance.GetName() + "-pull-secret"
	if instance.Spec.CustomPullSecret != "" {
		pullSecretName = instance.Spec.CustomPullSecret
	}

	p.ImagePullSecrets = append(p.ImagePullSecrets, corev1.LocalObjectReference{
		Name: pullSecretName,
	})

	if instance.Spec.KubernetesMonitoringSpec.Image == "" {
		i, err := BuildActiveGateImage(instance.Spec.APIURL, instance.Spec.KubernetesMonitoringSpec.ActiveGateVersion)
		if err != nil {
			return err
		}
		p.Containers[0].Image = i
	}

	return nil
}

func BuildLabels(name string, labels map[string]string) map[string]string {
	result := BuildLabelsForQuery(name)
	for key, value := range labels {
		result[key] = value
	}
	return result
}

func MergeLabels(labels ...map[string]string) map[string]string {
	res := map[string]string{}
	for _, m := range labels {
		for k, v := range m {
			res[k] = v
		}
	}

	return res
}

// buildLabels returns generic labels based on the name given for a Dynatrace OneAgent
func BuildLabelsForQuery(name string) map[string]string {
	return map[string]string{
		"dynatrace":  "activegate",
		"activegate": name,
	}
}

const (
	ActivegateName = "dynatrace-operator"

	MonitoringServiceAccount = "dynatrace-kubernetes-monitoring"

	KubernetesArch     = "kubernetes.io/arch"
	KubernetesOs       = "kubernetes.io/os"
	KubernetesBetaArch = "beta.kubernetes.io/arch"
	KubernetesBetaOs   = "beta.kubernetes.io/os"

	AMD64 = "amd64"
	ARM64 = "arm64"
	LINUX = "linux"

	DtCapabilities    = "DT_CAPABILITIES"
	DtIdSeedNamespace = "DT_ID_SEED_NAMESPACE"
	DtIdSeedClusterId = "DT_ID_SEED_K8S_CLUSTER_ID"

	DtCapabilitiesArg = "--enable=kubernetes_monitoring"

	Comma = ","
)
