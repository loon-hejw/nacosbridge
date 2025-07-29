/*
Copyright 2025.

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

package main

import (
	"flag"
	"nacosbridge/controller"
	"nacosbridge/service"
	"os"

	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	// +kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
}

func main() {
	opts := zap.Options{}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		HealthProbeBindAddress: "0",
		LeaderElection:         false,
		LeaderElectionID:       "d33b1eea.nacosbridge.io",
	})

	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	metric := service.NewMetrics(":9090")
	if err := mgr.Add(metric); err != nil {
		setupLog.Error(err, "unable to add metric handler")
		os.Exit(1)
	}

	statusUpdater := service.NewStatusUpdateHandler()
	if err := mgr.Add(statusUpdater); err != nil {
		setupLog.Error(err, "unable to add status update handler")
		os.Exit(1)
	}
	if err := statusUpdater.InjectClient(mgr.GetClient()); err != nil {
		setupLog.Error(err, "unable to inject client")
		os.Exit(1)
	}

	handler := service.NewService(statusUpdater.Writer())
	if err := mgr.Add(handler); err != nil {
		setupLog.Error(err, "unable to add service handler")
		os.Exit(1)
	}
	if err := (&controller.ConfigMap{Handler: handler}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to setup configmap controller")
		os.Exit(1)
	}
	if err := (&controller.Service{Handler: handler}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to setup service controller")
		os.Exit(1)
	}
	if err := (&controller.Node{Handler: handler}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to setup node controller")
		os.Exit(1)
	}

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
