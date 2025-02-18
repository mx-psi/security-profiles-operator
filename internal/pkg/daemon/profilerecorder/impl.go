/*
Copyright 2021 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package profilerecorder

import (
	"context"

	"github.com/crossplane/crossplane-runtime/pkg/resource"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type defaultImpl struct{}

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -generate
//counterfeiter:generate . impl
type impl interface {
	NewClient(ctrl.Manager) (client.Client, error)
	ClientGet(client.Client, client.ObjectKey, client.Object) error
	NewControllerManagedBy(
		manager.Manager, string, func(obj runtime.Object) bool,
		func(obj runtime.Object) bool, reconcile.Reconciler) error
	ManagerGetClient(manager.Manager) client.Client
	ManagerGetEventRecorderFor(manager.Manager, string) record.EventRecorder
}

func (*defaultImpl) NewClient(mgr ctrl.Manager) (client.Client, error) {
	return client.New(mgr.GetConfig(), client.Options{})
}

func (*defaultImpl) ClientGet(
	c client.Client, key client.ObjectKey, obj client.Object,
) error {
	return c.Get(context.Background(), key, obj)
}

func (*defaultImpl) NewControllerManagedBy(
	m manager.Manager,
	name string,
	p1 func(obj runtime.Object) bool,
	p2 func(obj runtime.Object) bool,
	r reconcile.Reconciler,
) error {
	return ctrl.NewControllerManagedBy(m).
		Named(name).
		WithEventFilter(predicate.And(
			resource.NewPredicates(p1),
			resource.NewPredicates(p2),
		)).
		For(&corev1.Pod{}).
		Complete(r)
}

func (*defaultImpl) ManagerGetClient(m manager.Manager) client.Client {
	return m.GetClient()
}

func (*defaultImpl) ManagerGetEventRecorderFor(
	m manager.Manager, name string,
) record.EventRecorder {
	return m.GetEventRecorderFor(name)
}
