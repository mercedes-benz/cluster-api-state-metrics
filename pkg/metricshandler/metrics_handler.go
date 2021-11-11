/*
Copyright 2019 The Kubernetes Authors All rights reserved.

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
[1]: https://github.com/kubernetes/kube-state-metrics/blob/e859b280fcc2/pkg/metricshandler/metrics_handler.go

The original source uses the kube-state-metrics's internal store package ([2])
which makes it impossible to re-use it. Compared to the upstream package this
metrics handler does not support sharding for now.

The original source was adjusted to:
- support the store.Builder struct instead of the upstream ksmtypes.BuilderInterface.
- remove sharding functionality to reduce complexity for the initial implementation.
*/

package metricshandler

import (
	"compress/gzip"
	"context"
	"io"
	"net/http"
	"strings"
	"sync"

	metricsstore "k8s.io/kube-state-metrics/v2/pkg/metrics_store"

	"github.com/daimler/cluster-api-state-metrics/pkg/options"
	"github.com/daimler/cluster-api-state-metrics/pkg/store"
)

type MetricsHandler struct {
	opts               *options.Options
	storeBuilder       *store.Builder
	enableGZIPEncoding bool

	// mtx protects metricsWriters, curShard, and curTotalShards
	mtx            *sync.RWMutex
	metricsWriters []metricsstore.MetricsWriter
}

// New creates and returns a new MetricsHandler with the given options.
func New(ctx context.Context, opts *options.Options, storeBuilder *store.Builder, enableGZIPEncoding bool) *MetricsHandler {
	m := &MetricsHandler{
		opts:               opts,
		storeBuilder:       storeBuilder,
		enableGZIPEncoding: enableGZIPEncoding,
		mtx:                &sync.RWMutex{},
	}
	// upstream adds the context during the sharding configuration.
	m.storeBuilder.WithContext(ctx)
	// upstream builds the metricsWriters in its sharding configuration
	m.metricsWriters = m.storeBuilder.Build()
	return m
}

func (m *MetricsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	m.mtx.RLock()
	defer m.mtx.RUnlock()
	resHeader := w.Header()
	var writer io.Writer = w

	resHeader.Set("Content-Type", `text/plain; version=`+"0.0.4")

	if m.enableGZIPEncoding {
		// Gzip response if requested. Taken from
		// github.com/prometheus/client_golang/prometheus/promhttp.decorateWriter.
		reqHeader := r.Header.Get("Accept-Encoding")
		parts := strings.Split(reqHeader, ",")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part == "gzip" || strings.HasPrefix(part, "gzip;") {
				writer = gzip.NewWriter(writer)
				resHeader.Set("Content-Encoding", "gzip")
			}
		}
	}

	for _, w := range m.metricsWriters {
		w.WriteAll(writer)
	}

	// In case we gzipped the response, we have to close the writer.
	if closer, ok := writer.(io.Closer); ok {
		closer.Close()
	}
}
