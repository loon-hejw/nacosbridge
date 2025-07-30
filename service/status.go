package service

import (
	"context"
	"reflect"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type NeedLeaderElectionNotification interface {
	OnElectedLeader()
}

// StatusUpdate contains an all the information needed to change an object's status to perform a specific update.
// Send down a channel to the goroutine that actually writes the changes back.
type StatusUpdate struct {
	NamespacedName types.NamespacedName
	Resource       client.Object
	Mutator        StatusMutator
}

func NewStatusUpdate(name, namespace string, resource client.Object, mutator StatusMutator) StatusUpdate {
	return StatusUpdate{
		NamespacedName: types.NamespacedName{
			Name:      name,
			Namespace: namespace,
		},
		Resource: resource,
		Mutator:  mutator,
	}
}

// StatusMutator is an interface to hold mutator functions for status updates.
type StatusMutator interface {
	Mutate(obj client.Object) client.Object
}

// StatusMutatorFunc is a function adaptor for StatusMutators.
type StatusMutatorFunc func(client.Object) client.Object

// Mutate adapts the StatusMutatorFunc to fit through the StatusMutator interface.
func (m StatusMutatorFunc) Mutate(old client.Object) client.Object {
	if m == nil {
		return nil
	}
	return m(old)
}

// StatusUpdateHandler holds the details required to actually write an Update back to the referenced object.
type StatusUpdateHandler struct {
	client        client.Client
	sendUpdates   chan struct{}
	updateChannel chan StatusUpdate
	ToNotify      []NeedLeaderElectionNotification
}

func NewStatusUpdateHandler() *StatusUpdateHandler {
	return &StatusUpdateHandler{
		sendUpdates:   make(chan struct{}),
		updateChannel: make(chan StatusUpdate, 100),
	}
}

func (suh *StatusUpdateHandler) apply(upd StatusUpdate) {
	fn := func() error {
		obj := upd.Resource
		kind := upd.Resource.GetObjectKind().GroupVersionKind().Kind

		// Get the resource.
		if err := suh.client.Get(context.Background(), upd.NamespacedName, obj); err != nil {
			log.Log.Info("get obj failed", "name", upd.NamespacedName.Name, "namespace", upd.NamespacedName.Namespace, "kind", kind, "error", err)
			return err
		}

		newObj := upd.Mutator.Mutate(obj)

		if isSpecEqual(obj, newObj) {
			log.Log.Info("skip update no-op", "name", upd.NamespacedName.Name, "namespace", upd.NamespacedName.Namespace, "kind", kind)
			return nil
		} else {
			if err := suh.client.Update(context.Background(), newObj); err != nil {
				log.Log.Error(err, "updated status failed", "name", upd.NamespacedName.Name, "namespace", upd.NamespacedName.Namespace, "kind", kind)
				return err
			}
			log.Log.Info("updated status ok", "name", upd.NamespacedName.Name, "namespace", upd.NamespacedName.Namespace, "kind", kind)
		}

		if isStatusEqual(obj, newObj) {
			log.Log.Info("skip update no-op", "name", upd.NamespacedName.Name, "namespace", upd.NamespacedName.Namespace, "kind", kind)
			return nil
		} else {
			if err := suh.client.Status().Update(context.Background(), newObj); err != nil {
				log.Log.Error(err, "updated status failed", "name", upd.NamespacedName.Name, "namespace", upd.NamespacedName.Namespace, "kind", kind)
				return err
			}
			log.Log.Info("updated status ok", "name", upd.NamespacedName.Name, "namespace", upd.NamespacedName.Namespace, "kind", kind)
		}
		return nil
	}

	if err := retry.RetryOnConflict(retry.DefaultBackoff, fn); err != nil {
		log.Log.Error(err, "unable to update status", "name", upd.NamespacedName.Name, "namespace", upd.NamespacedName.Namespace,
			"kind", upd.Resource.GetObjectKind().GroupVersionKind().Kind, "error", err.Error())
	}
}

func (suh *StatusUpdateHandler) NeedLeaderElection() bool {
	return true
}

func (suh *StatusUpdateHandler) InjectClient(c client.Client) error {
	suh.client = c
	return nil
}

// Start runs the goroutine to perform status writes.
func (suh *StatusUpdateHandler) Start(ctx context.Context) error {
	log.Log.Info("started status update handler")
	defer log.Log.Info("stopped status update handler")

	// Enable StatusUpdaters to start sending updates to this handler.
	close(suh.sendUpdates)

	for _, t := range suh.ToNotify {
		go t.OnElectedLeader()
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case upd := <-suh.updateChannel:
			suh.apply(upd)
		}
	}
}

// Writer retrieves the interface that should be used to write to the StatusUpdateHandler.
func (suh *StatusUpdateHandler) Writer() StatusUpdater {
	return &StatusUpdateWriter{
		enabled:       suh.sendUpdates,
		updateChannel: suh.updateChannel,
	}
}

// StatusUpdater describes an interface to send status updates somewhere.
type StatusUpdater interface {
	Send(su StatusUpdate)
}

// StatusUpdateWriter takes status updates and sends these to the StatusUpdateHandler via a channel.
type StatusUpdateWriter struct {
	enabled       <-chan struct{}
	updateChannel chan<- StatusUpdate
}

// Send sends the given StatusUpdate off to the update channel for writing by the StatusUpdateHandler.
func (suw *StatusUpdateWriter) Send(update StatusUpdate) {
	// Non-blocking receive to see if we should pass along update.
	select {
	case <-suw.enabled:
		suw.updateChannel <- update
	default:

	}
}

// isStatusEqual checks that two objects of supported Kubernetes types
// have equivalent Status structs.
func isStatusEqual(objA, objB interface{}) bool {
	switch a := objA.(type) {
	case *corev1.Service:

		if a.Labels == nil || a.Labels["nacosbridge.io/status-changed"] != "true" {
			return false
		}
		if b, ok := objB.(*corev1.Service); ok {
			if reflect.DeepEqual(a.Status, b.Status) {
				return true
			}
		}
	default:
		return reflect.DeepEqual(objA, objB)
	}
	return false
}

func isSpecEqual(objA, objB interface{}) bool {
	switch a := objA.(type) {
	case *corev1.Service:

		if a.Labels == nil || a.Labels["nacosbridge.io/spec-changed"] != "true" {
			return false
		}

		if b, ok := objB.(*corev1.Service); ok {
			if reflect.DeepEqual(a.Spec, b.Spec) {
				return true
			}
		}
	default:
		return reflect.DeepEqual(objA, objB)
	}
	return false
}
