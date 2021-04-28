package csigc

import (
	"context"
	"fmt"
	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/controllers/dynakube"
	"github.com/Dynatrace/dynatrace-operator/controllers/utils"
	"github.com/Dynatrace/dynatrace-operator/dtclient"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"time"
)

// CSIGarbageCollector removes unused and outdated agent versions
type CSIGarbageCollector struct {
	client       client.Client
	logger       logr.Logger
	dtcBuildFunc dynakube.DynatraceClientFunc
}

// NewReconciler returns a new CSIGarbageCollector
func NewReconciler(client client.Client) *CSIGarbageCollector {
	return &CSIGarbageCollector{
		client:       client,
		logger:       log.Log.WithName("csi.gc.controller"),
		dtcBuildFunc: dynakube.BuildDynatraceClient,
	}
}

func (r *CSIGarbageCollector) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&dynatracev1alpha1.DynaKube{}).
		Complete(r)
}

var _ reconcile.Reconciler = &CSIGarbageCollector{}

func (r *CSIGarbageCollector) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	r.logger.Info("reconciling csi driver", "namespace", request.Namespace, "name", request.Name)

	var dk dynatracev1alpha1.DynaKube
	if err := r.client.Get(ctx, request.NamespacedName, &dk); err != nil {
		if k8serrors.IsNotFound(err) {
			return newReconcileResult(), nil
		}
		return newReconcileResult(), err
	}

	var tkns corev1.Secret
	if err := r.client.Get(ctx, client.ObjectKey{Name: utils.GetTokensName(&dk), Namespace: dk.Namespace}, &tkns); err != nil {
		return newReconcileResult(), fmt.Errorf("failed to query tokens: %w", err)
	}

	dtc, err := r.dtcBuildFunc(r.client, &dk, &tkns)
	if err != nil {
		return newReconcileResult(), fmt.Errorf("failed to create Dynatrace client: %w", err)
	}

	ci, err := dtc.GetConnectionInfo()
	if err != nil {
		return newReconcileResult(), fmt.Errorf("failed to fetch connection info: %w", err)
	}

	ver, err := dtc.GetLatestAgentVersion(dtclient.OsUnix, dtclient.InstallerTypePaaS)
	if err != nil {
		return newReconcileResult(), fmt.Errorf("failed to query OneAgent version: %w", err)
	}

	if err := runBinaryGarbageCollection(r.logger, ci.TenantUUID, ver); err != nil {
		return newReconcileResult(), fmt.Errorf("garbage collection failed: %w", err)
	}

	return newReconcileResult(), nil
}

func newReconcileResult() reconcile.Result {
	return reconcile.Result{
		Requeue:      false,
		RequeueAfter: time.Hour,
	}
}
