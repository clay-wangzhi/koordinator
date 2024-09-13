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
	"math/rand"
	"os"
	"time"

	"k8s.io/component-base/logs"
	frameworkruntime "k8s.io/kubernetes/pkg/scheduler/framework/runtime"

	"github.com/clay-wangzhi/koordinator/cmd/koord-scheduler/app"
	"github.com/clay-wangzhi/koordinator/pkg/scheduler/plugins/defaultprebind"
	"github.com/clay-wangzhi/koordinator/pkg/scheduler/plugins/loadaware"

	// Ensure metric package is initialized
	_ "k8s.io/component-base/metrics/prometheus/clientgo"
	// Ensure scheme package is initialized.
	_ "github.com/clay-wangzhi/koordinator/pkg/scheduler/apis/config/scheme"
)

var koordinatorPlugins = map[string]frameworkruntime.PluginFactory{
	loadaware.Name:      loadaware.New,
	defaultprebind.Name: defaultprebind.New,
}

func flatten(plugins map[string]frameworkruntime.PluginFactory) []app.Option {
	options := make([]app.Option, 0, len(plugins))
	for name, factoryFn := range plugins {
		options = append(options, app.WithPlugin(name, factoryFn))
	}
	return options
}

func main() {
	rand.Seed(time.Now().UnixNano())

	// Register custom plugins to the scheduler framework.
	// Later they can consist of scheduler profile(s) and hence
	// used by various kinds of workloads.
	command := app.NewSchedulerCommand(
		flatten(koordinatorPlugins)...,
	)

	logs.InitLogs()
	defer logs.FlushLogs()

	if err := command.Execute(); err != nil {
		os.Exit(1)
	}
}
