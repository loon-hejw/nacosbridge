package controller

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/cache"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Service struct {
	Client  client.Client
	Handler cache.ResourceEventHandler
}

// +kubebuilder:rbac:groups=core,resources=services,verbs=*
// +kubebuilder:rbac:groups=core,resources=services/status,verbs=*
// +kubebuilder:rbac:groups=core,resources=services/finalizers,verbs=*

func (s *Service) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	service := &corev1.Service{}
	if err := s.Client.Get(ctx, req.NamespacedName, service); err != nil {
		if apierrors.IsNotFound(err) {
			service.Name = req.NamespacedName.Name
			service.Namespace = req.NamespacedName.Namespace

			s.Handler.OnDelete(service)
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if service.Labels == nil {
		service.Labels = make(map[string]string)
	}

	s.Handler.OnAdd(service, false)
	return ctrl.Result{}, nil
}

func (s *Service) SetupWithManager(mgr ctrl.Manager) error {
	s.Client = mgr.GetClient()
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Service{}).
		Complete(s)
}
