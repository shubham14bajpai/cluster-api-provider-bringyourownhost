// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package controllers

import (
	"context"
	"reflect"
	"sync"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/pkg/errors"
	infrav1 "github.com/vmware-tanzu/cluster-api-provider-bringyourownhost/apis/infrastructure/v1beta1"
	"github.com/vmware-tanzu/cluster-api-provider-bringyourownhost/common"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"

	clusterutilv1 "sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/annotations"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/cluster-api/util/patch"
)

var (
	DefaultAPIEndpointPort    = 6443
	clusterControlledType     = &infrav1.ByoCluster{}
	clusterControlledTypeName = reflect.TypeOf(clusterControlledType).Elem().Name()
	clusterControlledTypeGVK  = infrav1.GroupVersion.WithKind(clusterControlledTypeName)
)

// ByoClusterReconciler reconciles a ByoCluster object
type ByoClusterReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

type ClusterContext struct {
	context.Context
	cluster           *clusterv1.Cluster
	byohCluster       *infrav1.ByoCluster
	genericEventCache sync.Map
}

//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=byoclusters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=byoclusters/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=byoclusters/finalizers,verbs=update
// +kubebuilder:rbac:groups=cluster.x-k8s.io,resources=clusters;clusters/status,verbs=get;list;watch

func (r *ByoClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (_ ctrl.Result, reterr error) {
	logger := log.FromContext(ctx)

	// Get the ByoCluster resource for this request.
	byoCluster := &infrav1.ByoCluster{}
	if err := r.Client.Get(ctx, req.NamespacedName, byoCluster); err != nil {
		if apierrors.IsNotFound(err) {
			logger.V(4).Info("ByoCluster not found, won't reconcile", "key", req.NamespacedName)
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	// Fetch the CAPI Cluster.
	cluster, err := clusterutilv1.GetOwnerCluster(ctx, r.Client, byoCluster.ObjectMeta)
	if err != nil {
		return reconcile.Result{}, err
	}
	if cluster == nil {
		logger.Info("Waiting for Cluster Controller to set OwnerRef on ByoCluster")
		return reconcile.Result{}, nil
	}
	if annotations.IsPaused(cluster, byoCluster) {
		logger.V(4).Info("ByoCluster %s/%s linked to a cluster that is paused",
			byoCluster.Namespace, byoCluster.Name)
		return reconcile.Result{}, nil
	}

	// Create the patch helper.
	patchHelper, err := patch.NewHelper(byoCluster, r.Client)
	if err != nil {
		return reconcile.Result{}, errors.Wrapf(
			err,
			"failed to init patch helper for %s %s/%s",
			byoCluster.GroupVersionKind(),
			byoCluster.Namespace,
			byoCluster.Name)
	}

	// Create the cluster context for this request.
	clusterContext := &ClusterContext{
		cluster:     cluster,
		byohCluster: byoCluster,
	}

	// Always issue a patch when exiting this function so changes to the
	// resource are patched back to the API server.
	defer func() {
		if err := patchByoCluster(ctx, patchHelper, byoCluster); err != nil {
			logger.Error(err, "failed to patch ByoCluster")
			if reterr == nil {
				reterr = err
			}
		}
	}()

	// Handle deleted clusters
	if !byoCluster.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, byoCluster)
	}

	// Handle non-deleted clusters
	return r.reconcileNormal(clusterContext)
}

func patchByoCluster(ctx context.Context, patchHelper *patch.Helper, byoCluster *infrav1.ByoCluster) error {
	// Always update the readyCondition by summarizing the state of other conditions.
	// A step counter is added to represent progress during the provisioning process (instead we are hiding it during the deletion process).
	conditions.SetSummary(byoCluster,
		conditions.WithStepCounterIf(byoCluster.ObjectMeta.DeletionTimestamp.IsZero()),
	)

	// Patch the object, ignoring conflicts on the conditions owned by this controller.
	return patchHelper.Patch(
		ctx,
		byoCluster,
		patch.WithOwnedConditions{Conditions: []clusterv1.ConditionType{
			clusterv1.ReadyCondition,
		}},
	)
}

// GetByoMachinesInCluster gets a cluster's ByoMachine resources.
func GetByoMachinesInCluster(
	ctx context.Context,
	controllerClient client.Client,
	namespace, clusterName string) ([]*infrav1.ByoMachine, error) {

	labels := map[string]string{clusterv1.ClusterLabelName: clusterName}
	machineList := &infrav1.ByoMachineList{}

	if err := controllerClient.List(
		ctx, machineList,
		client.InNamespace(namespace),
		client.MatchingLabels(labels)); err != nil {
		return nil, err
	}

	machines := make([]*infrav1.ByoMachine, len(machineList.Items))
	for i := range machineList.Items {
		machines[i] = &machineList.Items[i]
	}

	return machines, nil
}

func (r ByoClusterReconciler) reconcileDelete(ctx context.Context, byoCluster *infrav1.ByoCluster) (reconcile.Result, error) {
	logger := log.FromContext(ctx)

	byoMachines, err := GetByoMachinesInCluster(ctx, r.Client, byoCluster.Namespace, byoCluster.Name)
	if err != nil {
		return reconcile.Result{}, errors.Wrapf(err,
			"unable to list ByoMachines part of ByoCluster %s/%s", byoCluster.Namespace, byoCluster.Name)
	}

	if len(byoMachines) > 0 {
		logger.Info("Waiting for ByoMachines to be deleted", "count", len(byoMachines))
		return reconcile.Result{RequeueAfter: 10 * time.Second}, nil
	}
	// Cluster is deleted so remove the finalizer.
	controllerutil.RemoveFinalizer(byoCluster, infrav1.ClusterFinalizer)

	return ctrl.Result{}, nil
}

func (r ByoClusterReconciler) reconcileNormal(ctx *ClusterContext) (reconcile.Result, error) {
	// If the ByoCluster doesn't have our finalizer, add it.
	logger := log.FromContext(ctx)
	controllerutil.AddFinalizer(ctx.byohCluster, infrav1.ClusterFinalizer)

	if ctx.byohCluster.Spec.ControlPlaneEndpoint.Port == 0 {
		ctx.byohCluster.Spec.ControlPlaneEndpoint.Port = int32(DefaultAPIEndpointPort)
	}

	ctx.byohCluster.Status.Ready = true

	// Ensure the ByohCluster is reconciled when the API server first comes online.
	// A reconcile event will only be triggered if the Cluster is not marked as
	// ControlPlaneInitialized.
	r.reconcileByohClusterWhenAPIServerIsOnline(ctx)
	if ctx.byohCluster.Spec.ControlPlaneEndpoint.IsZero() {
		logger.Info("control plane endpoint is not reconciled")
		return reconcile.Result{}, nil
	}

	// Wait until the API server is online and accessible.
	if !r.isAPIServerOnline(ctx) {
		return reconcile.Result{}, nil
	}

	return reconcile.Result{}, nil
}

var (
	// apiServerTriggers is used to prevent multiple goroutines for a single
	// Cluster that poll to see if the target API server is online.
	apiServerTriggers   = map[types.UID]struct{}{}
	apiServerTriggersMu sync.Mutex
)

func (r ByoClusterReconciler) reconcileByohClusterWhenAPIServerIsOnline(ctx *ClusterContext) {
	logger := log.FromContext(ctx)
	if conditions.IsTrue(ctx.cluster, clusterv1.ControlPlaneInitializedCondition) {
		logger.Info("skipping reconcile when API server is online",
			"reason", "controlPlaneInitialized")
		return
	}
	apiServerTriggersMu.Lock()
	defer apiServerTriggersMu.Unlock()
	if _, ok := apiServerTriggers[ctx.cluster.UID]; ok {
		logger.Info("skipping reconcile when API server is online",
			"reason", "alreadyPolling")
		return
	}
	apiServerTriggers[ctx.cluster.UID] = struct{}{}
	go func() {
		// Block until the target API server is online.
		logger.Info("start polling API server for online check")
		wait.PollImmediateInfinite(time.Second*1, func() (bool, error) { return r.isAPIServerOnline(ctx), nil }) // nolint:errcheck
		logger.Info("stop polling API server for online check")
		logger.Info("triggering GenericEvent", "reason", "api-server-online")
		eventChannel := ctx.GetGenericEventChannelFor(ctx.byohCluster.GetObjectKind().GroupVersionKind())
		eventChannel <- event.GenericEvent{
			Object: ctx.byohCluster,
		}

		// Once the control plane has been marked as initialized it is safe to
		// remove the key from the map that prevents multiple goroutines from
		// polling the API server to see if it is online.
		logger.Info("start polling for control plane initialized")
		wait.PollImmediateInfinite(time.Second*1, func() (bool, error) { return r.isControlPlaneInitialized(ctx, ctx.cluster), nil }) // nolint:errcheck
		logger.Info("stop polling for control plane initialized")
		apiServerTriggersMu.Lock()
		delete(apiServerTriggers, ctx.cluster.UID)
		apiServerTriggersMu.Unlock()
	}()
}

func (r ByoClusterReconciler) isAPIServerOnline(ctx *ClusterContext) bool {
	logger := log.FromContext(ctx)
	if kubeClient, err := common.NewKubeClient(ctx, r.Client, ctx.cluster); err == nil {
		if _, err := kubeClient.CoreV1().Nodes().List(ctx, metav1.ListOptions{}); err == nil {
			// The target cluster is online. To make sure the correct control
			// plane endpoint information is logged, it is necessary to fetch
			// an up-to-date Cluster resource. If this fails, then set the
			// control plane endpoint information to the values from the
			// ByohCluster resource, as it must have the correct information
			// if the API server is online.
			cluster := &clusterv1.Cluster{}
			clusterKey := client.ObjectKey{Namespace: ctx.cluster.Namespace, Name: ctx.cluster.Name}
			if err := r.Client.Get(ctx, clusterKey, cluster); err != nil {
				cluster = ctx.cluster.DeepCopy()
				cluster.Spec.ControlPlaneEndpoint.Host = ctx.byohCluster.Spec.ControlPlaneEndpoint.Host
				cluster.Spec.ControlPlaneEndpoint.Port = ctx.byohCluster.Spec.ControlPlaneEndpoint.Port
				logger.Error(err, "failed to get updated cluster object while checking if API server is online")
			}
			logger.Info(
				"API server is online",
				"controlPlaneEndpoint", cluster.Spec.ControlPlaneEndpoint.String())
			return true
		}
	}
	return false
}

func (r ByoClusterReconciler) isControlPlaneInitialized(ctx context.Context, cluster *clusterv1.Cluster) bool {
	logger := log.FromContext(ctx)
	c := &clusterv1.Cluster{}
	clusterKey := client.ObjectKey{Namespace: cluster.Namespace, Name: cluster.Name}
	if err := r.Client.Get(ctx, clusterKey, c); err != nil {
		if !apierrors.IsNotFound(err) {
			logger.Error(err, "failed to get updated cluster object while checking if control plane is initialized")
			return false
		}
		logger.Info("exiting early because cluster no longer exists")
		return true
	}
	return conditions.IsTrue(cluster, clusterv1.ControlPlaneInitializedCondition)
}

// SetupWithManager sets up the controller with the Manager.
func (r *ByoClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		// Watch the controlled, infrastructure resource.
		For(clusterControlledType).
		// Watch the CAPI resource that owns this infrastructure resource.
		Watches(
			&source.Kind{Type: &clusterv1.Cluster{}},
			handler.EnqueueRequestsFromMapFunc(clusterutilv1.ClusterToInfrastructureMapFunc(infrav1.GroupVersion.WithKind(clusterControlledTypeGVK.Kind))),
		).
		Complete(r)
}

// GetGenericEventChannelFor returns a generic event channel for a resource
// specified by the provided GroupVersionKind.
func (c *ClusterContext) GetGenericEventChannelFor(gvk schema.GroupVersionKind) chan event.GenericEvent {
	if val, ok := c.genericEventCache.Load(gvk); ok {
		return val.(chan event.GenericEvent)
	}
	val, _ := c.genericEventCache.LoadOrStore(gvk, make(chan event.GenericEvent))
	return val.(chan event.GenericEvent)
}
