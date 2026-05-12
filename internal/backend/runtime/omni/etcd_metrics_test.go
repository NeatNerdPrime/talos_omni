// Copyright (c) 2026 Sidero Labs, Inc.
//
// Use of this software is governed by the Business Source License
// included in the LICENSE file.

package omni //nolint:testpackage // exercises unexported Observe.

import (
	"context"
	"strings"
	"testing"

	"github.com/cosi-project/runtime/pkg/resource"
	"github.com/cosi-project/runtime/pkg/state"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/siderolabs/omni/client/pkg/access"
	"github.com/siderolabs/omni/internal/pkg/auth"
	"github.com/siderolabs/omni/internal/pkg/auth/actor"
	"github.com/siderolabs/omni/internal/pkg/ctxstore"
)

func TestEtcdMetricsObserve(t *testing.T) {
	t.Parallel()

	m := newEtcdMetrics()

	// pure internal write — no identity in context, only the internal mark.
	internalCtx := actor.MarkContextAsInternalActor(context.Background())

	// internal create + update + destroy of "Foo" totaling 100+200+50 bytes
	require.NoError(t, m.Observe(internalCtx, state.Created, "Foo", resource.PhaseRunning, resource.PhaseRunning, 100))
	require.NoError(t, m.Observe(internalCtx, state.Updated, "Foo", resource.PhaseRunning, resource.PhaseRunning, 200))
	require.NoError(t, m.Observe(internalCtx, state.Destroyed, "Foo", resource.PhaseRunning, resource.PhaseRunning, 50))

	// user-initiated teardown that passes through MarkContextAsInternalActor — must attribute to user, and must label as "teardown" (running → tearingDown transition).
	userCtx := actor.MarkContextAsInternalActor(
		ctxstore.WithValue(context.Background(), auth.VerifiedEmailContextKey{Email: "alice@example.com"}),
	)
	require.NoError(t, m.Observe(userCtx, state.Updated, "Foo", resource.PhaseTearingDown, resource.PhaseRunning, 175))

	// Two follow-up internal updates while already tearing down. These must NOT be labeled "teardown" — they're regular "update"s.
	require.NoError(t, m.Observe(internalCtx, state.Updated, "Foo", resource.PhaseTearingDown, resource.PhaseTearingDown, 180))
	require.NoError(t, m.Observe(internalCtx, state.Updated, "Foo", resource.PhaseTearingDown, resource.PhaseTearingDown, 170))

	// infra provider create of "Bar" - actor_id should be the provider base name
	infraCtx := ctxstore.WithValue(
		context.Background(),
		auth.IdentityContextKey{Identity: "aws-1" + access.InfraProviderServiceAccountNameSuffix},
	)
	require.NoError(t, m.Observe(infraCtx, state.Created, "Bar", resource.PhaseRunning, resource.PhaseRunning, 1024))

	// non-write event types must be silently ignored
	require.NoError(t, m.Observe(internalCtx, state.Errored, "Foo", resource.PhaseRunning, resource.PhaseRunning, 1))
	require.NoError(t, m.Observe(internalCtx, state.Bootstrapped, "Foo", resource.PhaseRunning, resource.PhaseRunning, 1))
	require.NoError(t, m.Observe(internalCtx, state.Noop, "Foo", resource.PhaseRunning, resource.PhaseRunning, 1))

	expected := `
# HELP omni_etcd_operations_total Number of successful etcd write operations, split by actor and resource type.
# TYPE omni_etcd_operations_total counter
omni_etcd_operations_total{actor="infraprovider",actor_id="aws-1",operation="create",resource_type="Bar"} 1
omni_etcd_operations_total{actor="internal",actor_id="",operation="create",resource_type="Foo"} 1
omni_etcd_operations_total{actor="internal",actor_id="",operation="destroy",resource_type="Foo"} 1
omni_etcd_operations_total{actor="internal",actor_id="",operation="update",resource_type="Foo"} 3
omni_etcd_operations_total{actor="user",actor_id="",operation="teardown",resource_type="Foo"} 1
`

	assert.NoError(t, testutil.CollectAndCompare(m.operations, strings.NewReader(expected)))

	expectedBytes := `
# HELP omni_etcd_resource_bytes_total Total marshaled bytes of resources written to (or removed from) etcd, split by actor and resource type.
# TYPE omni_etcd_resource_bytes_total counter
omni_etcd_resource_bytes_total{actor="infraprovider",actor_id="aws-1",operation="create",resource_type="Bar"} 1024
omni_etcd_resource_bytes_total{actor="internal",actor_id="",operation="create",resource_type="Foo"} 100
omni_etcd_resource_bytes_total{actor="internal",actor_id="",operation="destroy",resource_type="Foo"} 50
omni_etcd_resource_bytes_total{actor="internal",actor_id="",operation="update",resource_type="Foo"} 550
omni_etcd_resource_bytes_total{actor="user",actor_id="",operation="teardown",resource_type="Foo"} 175
`

	assert.NoError(t, testutil.CollectAndCompare(m.bytes, strings.NewReader(expectedBytes)))
}
