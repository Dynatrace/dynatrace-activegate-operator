package activegate

import (
	"context"
	"fmt"
	"time"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-activegate-operator/pkg/apis/dynatrace/v1alpha1"
	"github.com/Dynatrace/dynatrace-activegate-operator/pkg/controller/builder"
	"github.com/Dynatrace/dynatrace-activegate-operator/pkg/controller/parser"
	"github.com/Dynatrace/dynatrace-activegate-operator/pkg/controller/version"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type updateService interface {
	FindOutdatedPods(
		r *ReconcileActiveGate,
		logger logr.Logger,
		instance *dynatracev1alpha1.ActiveGate) ([]corev1.Pod, error)
	IsLatest(logger logr.Logger, image string, imageID string, imagePullSecret *corev1.Secret) (bool, error)
	UpdatePods(
		r *ReconcileActiveGate,
		pod *corev1.Pod,
		instance *dynatracev1alpha1.ActiveGate,
		secret *corev1.Secret) (*reconcile.Result, error)
}

type activeGateUpdateService struct{}

func (us *activeGateUpdateService) UpdatePods(
	r *ReconcileActiveGate,
	pod *corev1.Pod,
	instance *dynatracev1alpha1.ActiveGate,
	secret *corev1.Secret) (*reconcile.Result, error) {
	if instance == nil {
		return nil, fmt.Errorf("instance is nil")
	} else if !instance.Spec.DisableActivegateUpdate &&
		instance.Status.UpdatedTimestamp.Add(UpdateInterval).Before(time.Now()) {
		log.Info("checking for outdated pods")
		// Check if pods have latest activegate version
		outdatedPods, err := r.updateService.FindOutdatedPods(r, log, instance)
		if err != nil {
			result := builder.ReconcileAfterFiveMinutes()
			// Too many requests, requeue after five minutes
			return &result, err
		}

		err = r.deletePods(log, outdatedPods)
		if err != nil {
			log.Error(err, err.Error())
			return &reconcile.Result{}, err
		}
		r.updateInstanceStatus(pod, instance, secret)
	} else if instance.Spec.DisableActivegateUpdate {
		log.Info("Skipping updating pods because of configuration", "disableActivegateUpdate", true)
	}
	return nil, nil
}

func (us *activeGateUpdateService) FindOutdatedPods(
	r *ReconcileActiveGate,
	logger logr.Logger,
	instance *dynatracev1alpha1.ActiveGate) ([]corev1.Pod, error) {
	pods, err := r.findPods(instance)
	if err != nil {
		logger.Error(err, "failed to list pods")
		return nil, err
	}

	var outdatedPods []corev1.Pod
	for _, pod := range pods {
		for _, status := range pod.Status.ContainerStatuses {
			if status.ImageID == "" || instance.Spec.Image == "" {
				// If image is not yet pulled or not given skip check
				continue
			}
			logger.Info("pods container status", "pod", pod.Name, "container", status.Name, "image id", status.ImageID)

			imagePullSecret := &corev1.Secret{}
			err := r.client.Get(context.TODO(), client.ObjectKey{Namespace: pod.Namespace, Name: ImagePullSecret}, imagePullSecret)
			if err != nil {
				logger.Error(err, err.Error())
			}

			isLatest, err := r.updateService.IsLatest(logger, instance.Spec.Image, status.ImageID, imagePullSecret)
			if err != nil {
				logger.Error(err, err.Error())
				//Error during image check, do nothing an continue with next status
				continue
			}

			if !isLatest {
				logger.Info("pod is outdated", "name", pod.Name)
				outdatedPods = append(outdatedPods, pod)
				// Pod is outdated, break loop
				break
			}
		}
	}

	return outdatedPods, nil
}

func (us *activeGateUpdateService) IsLatest(logger logr.Logger, image string, imageID string, imagePullSecret *corev1.Secret) (bool, error) {
	dockerConfig, err := parser.NewDockerConfig(imagePullSecret)
	if err != nil {
		logger.Info(err.Error())
	}

	dockerVersionChecker := version.NewDockerVersionChecker(image, imageID, dockerConfig)
	return dockerVersionChecker.IsLatest()
}

const (
	ImagePullSecret = "dynatrace-activegate-registry"
)
