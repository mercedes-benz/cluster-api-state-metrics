/*
Copyright 2018 The Kubernetes Authors All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

The original file is located at [1].
[1]: https://github.com/kubernetes/kube-state-metrics/blob/e859b280fcc2/internal/store/builder.go

The original source was adjusted to:
- use a controller-runtime client instead of the client-go interface.
- remove sharding functionality to reduce complexity for the initial implementation.
- remove unused functions.
- use custom type for BuildStoresFunc to support controller-runtime client instead
  of client-go.

The upstream package is internal which is why we are not allowed to import and
use it directly.

Also it is not possible to re-use the wrapper at [2] because it is not flexible
enough.

[2]: https://github.com/kubernetes/kube-state-metrics/tree/master/pkg/builder
*/

package store

import (
	"context"
	"reflect"
	"sort"
	"strings"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
	ksmtypes "k8s.io/kube-state-metrics/v2/pkg/builder/types"
	generator "k8s.io/kube-state-metrics/v2/pkg/metric_generator"
	metricsstore "k8s.io/kube-state-metrics/v2/pkg/metrics_store"
	"k8s.io/kube-state-metrics/v2/pkg/options"
	"k8s.io/kube-state-metrics/v2/pkg/watch"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// BuildStoresFunc function signature that is used to return a list of cache.Store using a WithWatch client
type BuildStoresFunc func(metricFamilies []generator.FamilyGenerator,
	expectedType interface{},
	listWatchFunc func(client client.WithWatch, ns, fieldSelector string) cache.ListerWatcher,
	useAPIServerCache bool,
) []cache.Store

// Builder helps to build store. It follows the builder pattern
// (https://en.wikipedia.org/wiki/Builder_pattern).
type Builder struct {
	ctrlClient           client.WithWatch
	namespaces           options.NamespaceList
	namespaceFilter      string
	ctx                  context.Context
	enabledResources     []string
	allowDenyList        ksmtypes.AllowDenyLister
	listWatchMetrics     *watch.ListWatchMetrics
	buildStoresFunc      BuildStoresFunc
	allowAnnotationsList map[string][]string
	allowLabelsList      map[string][]string
	useAPIServerCache    bool
}

// NewBuilder returns a new builder.
func NewBuilder() *Builder {
	b := &Builder{}
	return b
}

// WithMetrics sets the metrics property of a Builder.
func (b *Builder) WithMetrics(r prometheus.Registerer) {
	b.listWatchMetrics = watch.NewListWatchMetrics(r)
}

// WithEnabledResources sets the enabledResources property of a Builder.
func (b *Builder) WithEnabledResources(r []string) error {
	for _, col := range r {
		if !resourceExists(col) {
			return errors.Errorf("resource %s does not exist. Available resources: %s", col, strings.Join(availableResources(), ","))
		}
	}

	var copy []string
	copy = append(copy, r...)

	sort.Strings(copy)

	b.enabledResources = copy
	return nil
}

// WithNamespaces sets the namespaces property of a Builder.
func (b *Builder) WithNamespaces(n options.NamespaceList, nsFilter string) {
	b.namespaces = n
	b.namespaceFilter = nsFilter
}

// WithContext sets the ctx property of a Builder.
func (b *Builder) WithContext(ctx context.Context) {
	b.ctx = ctx
}

// WithCtrlClient sets the controller-runtime client with watch as property of a Builder.
func (b *Builder) WithCtrlClient(c client.WithWatch) {
	b.ctrlClient = c
}

// WithAllowDenyList configures the allow or denylisted metric to be exposed
// by the store build by the Builder.
func (b *Builder) WithAllowDenyList(l ksmtypes.AllowDenyLister) {
	b.allowDenyList = l
}

// WithGenerateStoresFunc configures a custom generate store function
func (b *Builder) WithGenerateStoresFunc(f BuildStoresFunc, u bool) {
	b.buildStoresFunc = f
	b.useAPIServerCache = u
}

// DefaultGenerateStoresFunc returns default buildStores function
func (b *Builder) DefaultGenerateStoresFunc() BuildStoresFunc {
	return b.buildStores
}

// WithAllowAnnotations configures which annotations can be returned for metrics
func (b *Builder) WithAllowAnnotations(annotations map[string][]string) {
	if len(annotations) > 0 {
		b.allowAnnotationsList = annotations
	}
}

// WithAllowLabels configures which labels can be returned for metrics
func (b *Builder) WithAllowLabels(labels map[string][]string) {
	if len(labels) > 0 {
		b.allowLabelsList = labels
	}
}

// Build initializes and registers all enabled stores.
// It returns metrics writers which can be used to write out
// metrics from the stores.
func (b *Builder) Build() []metricsstore.MetricsWriter {
	if b.allowDenyList == nil {
		panic("allowDenyList should not be nil")
	}

	var metricsWriters []metricsstore.MetricsWriter
	var activeStoreNames []string

	for _, c := range b.enabledResources {
		constructor, ok := availableStores[c]
		if ok {
			stores := cacheStoresToMetricStores(constructor(b))
			activeStoreNames = append(activeStoreNames, c)
			if len(stores) == 1 {
				metricsWriters = append(metricsWriters, stores[0])
			} else {
				metricsWriters = append(metricsWriters, metricsstore.NewMultiStoreMetricsWriter(stores))
			}
		}
	}

	klog.Infof("Active resources: %s", strings.Join(activeStoreNames, ","))

	return metricsWriters
}

func resourceExists(name string) bool {
	_, ok := availableStores[name]
	return ok
}

func availableResources() []string {
	c := []string{}
	for name := range availableStores {
		c = append(c, name)
	}
	return c
}

func (b *Builder) buildStores(
	metricFamilies []generator.FamilyGenerator,
	expectedType interface{},
	listWatchFunc func(client client.WithWatch, ns string, fieldSelector string) cache.ListerWatcher,
	useAPIServerCache bool,
) []cache.Store {
	metricFamilies = generator.FilterMetricFamilies(b.allowDenyList, metricFamilies)
	composedMetricGenFuncs := generator.ComposeMetricGenFuncs(metricFamilies)
	familyHeaders := generator.ExtractMetricFamilyHeaders(metricFamilies)

	if b.namespaces.IsAllNamespaces() {
		store := metricsstore.NewMetricsStore(
			familyHeaders,
			composedMetricGenFuncs,
		)
		listWatcher := listWatchFunc(b.ctrlClient, v1.NamespaceAll, b.namespaceFilter)
		b.startReflector(expectedType, store, listWatcher, useAPIServerCache)
		return []cache.Store{store}
	}

	stores := make([]cache.Store, 0, len(b.namespaces))
	for _, ns := range b.namespaces {
		store := metricsstore.NewMetricsStore(
			familyHeaders,
			composedMetricGenFuncs,
		)
		listWatcher := listWatchFunc(b.ctrlClient, ns, b.namespaceFilter)
		b.startReflector(expectedType, store, listWatcher, useAPIServerCache)
		stores = append(stores, store)
	}

	return stores
}

// startReflector starts a Kubernetes client-go reflector with the given
// listWatcher and registers it with the given store.
func (b *Builder) startReflector(
	expectedType interface{},
	store cache.Store,
	listWatcher cache.ListerWatcher,
	useAPIServerCache bool,
) {
	instrumentedListWatch := watch.NewInstrumentedListerWatcher(listWatcher, b.listWatchMetrics, reflect.TypeOf(expectedType).String(), useAPIServerCache)
	reflector := cache.NewReflector(instrumentedListWatch, expectedType, store, 0)
	go reflector.Run(b.ctx.Done())
}

// cacheStoresToMetricStores converts []cache.Store into []*metricsstore.MetricsStore
func cacheStoresToMetricStores(cStores []cache.Store) []*metricsstore.MetricsStore {
	mStores := make([]*metricsstore.MetricsStore, 0, len(cStores))
	for _, store := range cStores {
		mStores = append(mStores, store.(*metricsstore.MetricsStore))
	}

	return mStores
}
