package controller

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/cache"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Node struct {
	Client  client.Client
	Handler cache.ResourceEventHandler
}

// +kubebuilder:rbac:groups=core,resources=nodes,verbs=*
// +kubebuilder:rbac:groups=core,resources=nodes/status,verbs=*
// +kubebuilder:rbac:groups=core,resources=nodes/finalizers,verbs=*

func (n *Node) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	node := &corev1.Node{}
	if err := n.Client.Get(ctx, req.NamespacedName, node); err != nil {
		if apierrors.IsNotFound(err) {
			node.Name = req.NamespacedName.Name
			node.Namespace = req.NamespacedName.Namespace
			n.Handler.OnDelete(node)
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	n.Handler.OnAdd(node, false)
	return ctrl.Result{}, nil
}

func (n *Node) SetupWithManager(mgr ctrl.Manager) error {
	n.Client = mgr.GetClient()
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Node{}).
		Complete(n)
}
