package controller

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/cache"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ConfigMap struct {
	Client  client.Client
	Handler cache.ResourceEventHandler
}

// +kubebuilder:rbac:groups=core,resources=configmaps,verbs=*
// +kubebuilder:rbac:groups=core,resources=configmaps/status,verbs=*
// +kubebuilder:rbac:groups=core,resources=configmaps/finalizers,verbs=*

func (c *ConfigMap) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	configmap := &corev1.ConfigMap{}
	if err := c.Client.Get(ctx, req.NamespacedName, configmap); err != nil {
		if apierrors.IsNotFound(err) {
			configmap.Name = req.NamespacedName.Name
			configmap.Namespace = req.NamespacedName.Namespace
			configmap.Labels = make(map[string]string)
			c.Handler.OnDelete(configmap)
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if configmap.Labels == nil {
		configmap.Labels = make(map[string]string)
	}
	if val, ok := configmap.Labels["nacosbridge.io/config"]; ok && val == "true" {
		c.Handler.OnAdd(configmap, false)
	}
	return ctrl.Result{}, nil
}

func (c *ConfigMap) SetupWithManager(mgr ctrl.Manager) error {
	c.Client = mgr.GetClient()
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.ConfigMap{}).
		Complete(c)
}
