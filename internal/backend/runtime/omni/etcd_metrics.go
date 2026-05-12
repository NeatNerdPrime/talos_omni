// Copyright (c) 2026 Sidero Labs, Inc.
//
// Use of this software is governed by the Business Source License
// included in the LICENSE file.

package omni

import (
	"context"

	"github.com/cosi-project/runtime/pkg/resource"
	"github.com/cosi-project/runtime/pkg/state"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/siderolabs/omni/internal/pkg/auth/actor"
)

// etcdMetrics records Prometheus metrics for write operations on the etcd-backed state.
type etcdMetrics struct {
	operations *prometheus.CounterVec
	bytes      *prometheus.CounterVec
}

var _ prometheus.Collector = &etcdMetrics{}

func newEtcdMetrics() *etcdMetrics {
	labels := []string{"operation", "actor", "actor_id", "resource_type"}

	return &etcdMetrics{
		operations: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "omni_etcd_operations_total",
				Help: "Number of successful etcd write operations, split by actor and resource type.",
			},
			labels,
		),
		bytes: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "omni_etcd_resource_bytes_total",
				Help: "Total marshaled bytes of resources written to (or removed from) etcd, split by actor and resource type.",
			},
			labels,
		),
	}
}

// Describe implements prometheus.Collector.
func (m *etcdMetrics) Describe(ch chan<- *prometheus.Desc) {
	prometheus.DescribeByCollect(m, ch)
}

// Collect implements prometheus.Collector.
func (m *etcdMetrics) Collect(ch chan<- prometheus.Metric) {
	m.operations.Collect(ch)
	m.bytes.Collect(ch)
}

// Observe implements etcd.ObserverFunc.
//
// Always returns nil — metric writes do not fail, and we never want to fail a successful etcd mutation because of a metrics emission issue.
func (m *etcdMetrics) Observe(ctx context.Context, eventType state.EventType, resourceType resource.Type, phase, previousPhase resource.Phase, marshaledBytes int) error {
	op := operationLabel(eventType, phase, previousPhase)
	if op == "" {
		return nil
	}

	caller := actor.Classify(ctx)

	metricID := ""
	if caller.Type == actor.TypeInfraProvider {
		metricID = caller.BaseName()
	}

	labels := []string{op, string(caller.Type), metricID, resourceType}

	m.operations.WithLabelValues(labels...).Inc()
	m.bytes.WithLabelValues(labels...).Add(float64(marshaledBytes))

	return nil
}

// operationLabel maps an etcd event to a metric label value.
//
// An Update that flips a resource's phase from running to tearing-down is reported as
// "teardown" — the first half of a delete (the second half is the eventual "destroy" once
// finalizers clear). Updates that occur *while* a resource is already tearing down (e.g. a
// controller removing a finalizer to allow the destroy) stay under "update", so we don't drown
// the original user-attributed teardown event in controller cleanup writes.
func operationLabel(t state.EventType, phase, previousPhase resource.Phase) string {
	switch t {
	case state.Created:
		return "create"
	case state.Updated:
		if previousPhase == resource.PhaseRunning && phase == resource.PhaseTearingDown {
			return "teardown"
		}

		return "update"
	case state.Destroyed:
		return "destroy"
	case state.Bootstrapped, state.Errored, state.Noop:
	}

	return ""
}
