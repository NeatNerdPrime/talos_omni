// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package client_test

import (
	"context"
	"log"

	"github.com/cosi-project/runtime/pkg/safe"
	"github.com/cosi-project/runtime/pkg/state"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/siderolabs/omni/client/pkg/client"
	"github.com/siderolabs/omni/client/pkg/omni/resources/omni"
	"github.com/siderolabs/omni/client/pkg/template"
)

//nolint:wsl,testableexamples,gocognit,gocyclo,cyclop
func Example() {
	// This example shows how to use Omni client to access resources.

	// For this example we will use an Omni service account.
	// You can create one in advance:
	//
	// omnictl serviceaccount create example.account
	// Created service account "example.account" with public key ID "<REDACTED>"
	//
	// Set the following environment variables to use the service account:
	// OMNI_ENDPOINT=https://<account>.omni.siderolabs.io:443
	// OMNI_SERVICE_ACCOUNT_KEY=base64encodedkey
	//
	// Note: Store the service account key securely, it will not be displayed again.
	ctx := context.Background()

	// Create a new client.
	c, err := client.New("https://<account>.omni.siderolabs.io:443", client.WithServiceAccount(
		"base64encodedkey", // From the generated service account.
	))
	if err != nil {
		log.Panicf("failed to create omni client %s", err)
	}

	defer func() {
		if e := c.Close(); e != nil {
			log.Printf("failed to close client %s", e)
		}
	}()

	// Omni uses COSI (https://github.com/cosi-project/runtime) for resource management,
	// the same runtime used in Talos.
	st := c.Omni().State()

	// List all machine statuses.
	machines, err := safe.ReaderListAll[*omni.MachineStatus](ctx, st)
	if err != nil {
		log.Panicf("failed to get machines %s", err)
	}

	var (
		cluster string
		machine *omni.MachineStatus
	)

	for item := range machines.All() {
		log.Printf("machine %s, connected: %t", item.Metadata(), item.TypedSpec().Value.GetConnected())

		// Find a machine that is allocated to a cluster, for use in the Talos API example below.
		if clusterName, ok := item.Metadata().Labels().Get(omni.LabelCluster); ok && machine == nil {
			cluster = clusterName
			machine = item
		}
	}

	// Create an empty cluster via a template.
	// Use template.Load to load a cluster template from YAML instead.
	tmpl := template.WithCluster("example.cluster")

	if _, err = tmpl.Sync(ctx, st); err != nil {
		log.Panicf("failed to sync cluster %s", err)
	}

	log.Printf("synced cluster")

	// Delete the cluster.
	if _, err = tmpl.Delete(ctx, st); err != nil {
		log.Panicf("failed to delete the cluster %s", err)
	}

	log.Printf("destroyed cluster")

	// No machines found, exit.
	if machine == nil {
		log.Printf("no allocated machines found, exit")

		return
	}

	// Use the Talos API through Omni.
	// You can use machine UUID as Omni will properly resolve it into machine IP.
	cpuInfo, err := c.Talos().WithCluster(
		cluster,
	).WithNodes(
		machine.Metadata().ID(),
	).CPUInfo(ctx, &emptypb.Empty{})
	if err != nil {
		log.Panicf("failed to read machine CPU info %s", err)
	}

	for _, message := range cpuInfo.Messages {
		for i, info := range message.CpuInfo {
			log.Printf("machine %s, CPU %d family %s", machine.Metadata(), i, info.CpuFamily)
		}

		if len(message.CpuInfo) == 0 {
			log.Printf("no CPU info for machine %s", machine.Metadata())
		}
	}

	// Get the talosconfig for the whole instance.
	_, err = c.Management().Talosconfig(ctx)
	if err != nil {
		log.Panicf("failed to get talosconfig %s", err)
	}

	// Resource management via the COSI state API.
	// The following example uses MachineClass as a representative user-managed resource to demonstrate
	// watching for changes, as well as create, update, and destroy operations.
	// The same patterns apply to any other user-managed resource type.

	// Watch the MachineClass kind for changes.
	// safe.StateWatchKind subscribes to all events for a resource type; safe.StateWatch watches a single
	// resource by ID. The watch runs until its context is canceled, so the consumer is in full control
	// of its lifetime.
	watchCtx, watchCancel := context.WithCancel(ctx)
	defer watchCancel()

	eventCh := make(chan safe.WrappedStateEvent[*omni.MachineClass])

	if err = safe.StateWatchKind(watchCtx, st, omni.NewMachineClass("").Metadata(), eventCh); err != nil {
		log.Panicf("failed to watch machine classes %s", err)
	}

	go func() {
		for {
			select {
			case <-watchCtx.Done():
				return
			case event := <-eventCh:
				if event.Type() == state.Errored {
					log.Printf("machine class watch error: %s", event.Error())

					return
				}

				res, resErr := event.Resource()
				if resErr != nil {
					log.Printf("machine class watch: failed to decode resource %s", resErr)

					continue
				}

				log.Printf("machine class event: %s %s", event.Type(), res.Metadata().ID())
			}
		}
	}()

	// MachineClass selects machines using label selector expressions.
	// Each entry in MatchLabels is a comma-separated set of conditions (AND within one entry, OR across entries).
	machineClass := omni.NewMachineClass("test")
	machineClass.Metadata().Labels().Set("my-label", "my-value")

	machineClass.TypedSpec().Value.MatchLabels = []string{
		"omni.sidero.dev/arch = amd64, omni.sidero.dev/cores > 2", // amd64 machines with more than 2 cores
	}

	if err = st.Create(ctx, machineClass); err != nil {
		log.Panicf("failed to create machine class %s", err)
	}

	// Update the machine class: add an OR condition to also match arm64 machines.
	updated, err := safe.StateUpdateWithConflicts(ctx, st, machineClass.Metadata(), func(res *omni.MachineClass) error {
		res.TypedSpec().Value.MatchLabels = append(res.TypedSpec().Value.MatchLabels, "omni.sidero.dev/arch = arm64")

		return nil
	})
	if err != nil {
		log.Panicf("failed to update machine class %s", err)
	}

	log.Printf("updated machine class labels: %v", updated.TypedSpec().Value.MatchLabels)

	// For upsert (create-or-update) semantics, safe.StateModify or safe.StateModifyWithResult can be used instead.

	// Destroy the machine class.
	//
	// The correct deletion sequence for COSI resources is: call Teardown, wait until all finalizers are
	// cleared (controllers holding them will release upon seeing the teardown phase), then call Destroy.
	// Calling only Teardown is not sufficient, and calling Destroy directly without Teardown first will fail
	// if any finalizers are present. TeardownAndDestroy handles this full sequence.
	if err = st.TeardownAndDestroy(ctx, machineClass.Metadata()); err != nil {
		log.Panicf("failed to destroy machine class %s", err)
	}
}
