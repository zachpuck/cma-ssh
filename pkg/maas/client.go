/*
Copyright 2019 The Kubernetes Authors.

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

package maas

import (
	"context"
	"fmt"

	clusterv1 "github.com/samsung-cnct/cma-ssh/pkg/apis/cluster/v1alpha1"
	"k8s.io/klog"

	"github.com/juju/gomaasapi"
)

const (
	ClusterAPIMachineIDAnnotationKey = "cluster.k8s.io/providerID" // Indicates a machine has been allocated
)

type Client struct {
	Controller gomaasapi.Controller
}

type ClientParams struct {
	ApiURL     string
	ApiVersion string
	ApiKey     string
}

func New(params *ClientParams) (Client, error) {
	controller, err := gomaasapi.NewController(gomaasapi.ControllerArgs{
		BaseURL: params.ApiURL,
		APIKey:  params.ApiKey})
	if err != nil {
		return Client{}, fmt.Errorf("error creating controller with version: %v", err)
	}

	return Client{Controller: controller}, nil
}

// Create creates a machine
func (c Client) Create(ctx context.Context, cluster *clusterv1.CnctCluster, machine *clusterv1.CnctMachine) error {
	klog.Infof("Creating machine %s", machine.Name)

	// TODO: Tag MAAS machine

	// Allocate MAAS machine
	allocateArgs := gomaasapi.AllocateMachineArgs{Tags: []string{}}
	m, _, err := c.Controller.AllocateMachine(allocateArgs)
	if err != nil {
		klog.Info("Create failed to allocate machine")
		return fmt.Errorf("error allocating machine %s: %v", machine.Name, err)
	}
	providerID := m.SystemID()

	// Deploy MAAS machine
	startArgs := gomaasapi.StartArgs{
		UserData:     "",
		DistroSeries: "",
		Kernel:       "",
		Comment:      "",
	}
	err = m.Start(startArgs)
	if err != nil {
		klog.Infof("Create failed to deploy machine %s", machine.Name)
		return nil
	}

	klog.Infof("Created machine %s (%s)", machine.Name, providerID)
	return nil
}

// Delete deletes a machine
func (c Client) Delete(ctx context.Context, cluster *clusterv1.CnctCluster, machine *clusterv1.CnctMachine) error {
	if machine.Spec.ProviderID == nil {
		klog.Warningf("can not delete  machine %s, providerID not set", machine.Name)
		return fmt.Errorf("machine %s has not been created", machine.Name)
	}

	// Release MAAS machine
	releaseArgs := gomaasapi.ReleaseMachinesArgs{SystemIDs: []string{*machine.Spec.ProviderID}}
	if err := c.Controller.ReleaseMachines(releaseArgs); err != nil {
		klog.Warningf("error releasing machine %s (%s): %v", machine.Name, *machine.Spec.ProviderID, err)
		return nil
	}

	return nil
}

// Update updates a machine
func (c Client) Update(ctx context.Context, cluster *clusterv1.CnctCluster, machine *clusterv1.CnctMachine) error {
	return nil
}

// Exists test for the existence of a machine
func (c Client) Exist(ctx context.Context, cluster *clusterv1.CnctCluster, machine *clusterv1.CnctMachine) (bool, error) {
	// ProviderID will be nil until Create completes successfully
	if machine.Spec.ProviderID == nil {
		return false, nil
	}

	// Get list of machines with tag
	machineArgs := gomaasapi.MachinesArgs{SystemIDs: []string{*machine.Spec.ProviderID}}
	machines, err := c.Controller.Machines(machineArgs)
	if err != nil {
		return false, fmt.Errorf("error listing machine %s (%s): %v", machine.Name, *machine.Spec.ProviderID, err)
	}
	if len(machines) != 1 {
		return false, fmt.Errorf("expected 1 machine %s (%s), found %d", machine.Name, *machine.Spec.ProviderID, len(machines))
	}

	return true, nil
}
