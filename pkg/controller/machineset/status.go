/*
Copyright 2019 Samsung SDS.

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

package machineset

import (
	"context"
	"time"

	"github.com/samsung-cnct/cma-ssh/pkg/apis/cluster/common"
	clusterv1alpha1 "github.com/samsung-cnct/cma-ssh/pkg/apis/cluster/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// calculateStatus returns a copy of Status with the current replica counts
func calculateStatus(ms *clusterv1alpha1.CnctMachineSet, filteredMachines []*clusterv1alpha1.CnctMachine) clusterv1alpha1.MachineSetStatus {
	msCopy := ms.DeepCopy()
	newStatus := msCopy.Status
	// Count the number of machines that have labels matching the labels of the machine
	// template of the replica set.
	fullyLabeledReplicasCount := 0
	templateLabel := labels.Set(ms.Spec.MachineTemplate.Labels).AsSelectorPreValidated()
	for _, machine := range filteredMachines {
		if templateLabel.Matches(labels.Set(machine.Labels)) {
			log.Info("Found matching Label")
			fullyLabeledReplicasCount++
		}
	}
	newStatus.Replicas = int32(len(filteredMachines))
	newStatus.FullyLabeledReplicas = int32(fullyLabeledReplicasCount)
	return newStatus
}

// updateStatus updates the MachineSet Status and adds an event
func (r *ReconcileMachineSet) updateStatus(
	machineSetInstance *clusterv1alpha1.CnctMachineSet,
	eventType string,
	event common.ControllerEvents,
	eventMessage common.ControllerEvents,
	args ...interface{},
) error {
	machineSetFreshInstance := &clusterv1alpha1.CnctMachineSet{}
	err := r.Get(
		context.Background(),
		client.ObjectKey{
			Namespace: machineSetInstance.GetNamespace(),
			Name:      machineSetInstance.GetName(),
		}, machineSetFreshInstance)
	if err != nil {
		return err
	}
	machineSetFreshInstance.Status.Phase = machineSetInstance.Status.Phase
	machineSetFreshInstance.Status.LastUpdated = &metav1.Time{Time: time.Now()}
	machineSetFreshInstance.Status.Replicas = machineSetInstance.Status.Replicas
	machineSetFreshInstance.Status.FullyLabeledReplicas = machineSetInstance.Status.FullyLabeledReplicas

	err = r.Update(context.Background(), machineSetFreshInstance)
	if err != nil {
		return err
	}
	if eventType != "" {
		r.Eventf(machineSetFreshInstance, eventType,
			string(event), string(eventMessage), args...)
	}
	return nil
}
