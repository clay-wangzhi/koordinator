/*
Copyright 2022 The Koordinator Authors.

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
	"time"

	"github.com/clay-wangzhi/cfs-quota-burst/cmd/koordlet/options"
	agent "github.com/clay-wangzhi/cfs-quota-burst/pkg/koordlet"
	"github.com/clay-wangzhi/cfs-quota-burst/pkg/koordlet/config"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
)

func main() {
	cfg := config.NewConfiguration()
	cfg.InitFlags(flag.CommandLine)
	klog.InitFlags(nil)
	flag.Parse()

	go wait.Forever(klog.Flush, 5*time.Second)
	defer klog.Flush()

	stopCtx := signals.SetupSignalHandler()

	// Get a config to talk to the apiserver
	klog.Info("Setting up kubeconfig for koordlet")
	err := cfg.InitKubeConfigForKoordlet(*options.KubeAPIQPS, *options.KubeAPIBurst)
	if err != nil {
		klog.Fatalf("Unable to setup kubeconfig: %v", err)
	}

	d, err := agent.NewDaemon(cfg)
	if err != nil {
		klog.Fatalf("Unable to setup koordlet daemon: %v", err)
	}

	// Start the Cmd
	klog.Info("Starting the koordlet daemon")
	d.Run(stopCtx.Done())
}
