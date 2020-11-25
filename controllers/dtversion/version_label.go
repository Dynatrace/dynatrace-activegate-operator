package dtversion

import (
	"context"
	"errors"
	"time"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/controllers/dtpullsecret"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type VersionLabelReconciler struct {
	client.Client
	log         logr.Logger
	instance    *dynatracev1alpha1.DynaKube
	matchLabels map[string]string //kubemon.BuildLabelsFromInstance(instance)
}

func NewReconciler(clt client.Client, log logr.Logger, instance *dynatracev1alpha1.DynaKube, matchLabels map[string]string) *VersionLabelReconciler {
	return &VersionLabelReconciler{
		Client:      clt,
		log:         log,
		instance:    instance,
		matchLabels: matchLabels,
	}
}

func (r *VersionLabelReconciler) Reconcile() (reconcile.Result, error) {
	pods, err := NewPodFinder(r, r.instance, r.matchLabels).FindPods()
	if err != nil {
		r.log.Error(err, "could not list pods")
		return reconcile.Result{}, err
	}

	err = r.setVersionLabelForPods(pods)
	if err != nil {
		return r.retryOnStatusError(err)
	}

	return reconcile.Result{}, nil
}

func (r *VersionLabelReconciler) setVersionLabelForPods(pods []corev1.Pod) error {
	for i := range pods {
		err := r.setVersionLabel(&pods[i])
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *VersionLabelReconciler) setVersionLabel(pod *corev1.Pod) error {
	versionLabel, err := r.getVersionLabelForPod(pod)
	if err != nil {
		return err
	}

	pod.Labels[VersionKey] = versionLabel
	err = r.Update(context.TODO(), pod)
	if err != nil {
		return err
	}
	return nil
}

func (r *VersionLabelReconciler) getVersionLabelForPod(pod *corev1.Pod) (string, error) {
	result := ""
	for _, status := range pod.Status.ContainerStatuses {
		if status.Image == "" {
			// If Image is not present, skip
			continue
		}

		imagePullSecret, err := dtpullsecret.GetImagePullSecret(r, r.instance)
		if err != nil {
			// Something wrong with pull secret, exit function entirely
			return "", err
		}

		dockerConfig, err := NewDockerConfig(imagePullSecret)
		// If an error is returned, try getting the image anyway

		versionLabel, err2 := NewPodImageInformation(status.Image, dockerConfig).GetVersionLabel()
		if err2 != nil && err != nil {
			// If an error is returned when getting matchLabels and an error occurred during parsing of the docker config
			// assume the error from parsing the docker config is the reason
			return "", err
		} else if err2 != nil {
			return "", err2
		}

		if result == "" {
			result = versionLabel
		}
	}

	return result, nil
}

func (r *VersionLabelReconciler) retryOnStatusError(err error) (reconcile.Result, error) {
	var statusError *k8serrors.StatusError
	if errors.As(err, &statusError) {
		// Since this happens early during deployment, pods might have been modified
		r.log.Info("retrying setting label due to status error. this error is normal and can be ignored at early stages of deployment", "error", err.Error())
		return reconcile.Result{RequeueAfter: 5 * time.Second}, nil
	}
	// Otherwise, fail loudly
	return reconcile.Result{RequeueAfter: 5 * time.Minute}, err
}
