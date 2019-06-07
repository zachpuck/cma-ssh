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

	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/pkg/errors"
	clusterv1alpha1 "github.com/samsung-cnct/cma-ssh/pkg/apis/cluster/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func validateMachineSet(machineSet *clusterv1alpha1.CnctMachineSet) (bool, error) {
	selector, err := metav1.LabelSelectorAsSelector(&machineSet.Spec.Selector)
	if err != nil {
		return false, errors.Wrapf(err, "Failed to parse MachineSet %q label selector", machineSet.Name)
	}

	if !selector.Matches(labels.Set(machineSet.Spec.MachineTemplate.Labels)) {
		return false, errors.Errorf("Failed validation on MachineSet %q label selector does not match machine template label", machineSet.Name)
	}
	return true, nil
}

func PollImmediate(interval, timeout time.Duration, condition wait.ConditionFunc) error {
	return wait.PollImmediate(interval, timeout, condition)
}

func (c *ReconcileMachineSet) getMachineSetsForMachine(m *clusterv1alpha1.CnctMachine) []*clusterv1alpha1.CnctMachineSet {
	if len(m.Labels) == 0 {
		log.Info("No machine sets found for Machine because it has no labels", "machine", m)
		return nil
	}
	msList := &clusterv1alpha1.CnctMachineSetList{}
	listOptions := &client.ListOptions{
		Namespace: m.Namespace,
	}
	err := c.Client.List(context.Background(), listOptions, msList)
	if err != nil {
		log.Error(err, "Failed to list machineSets")
		return nil
	}
	var mss []*clusterv1alpha1.CnctMachineSet
	for idx := range msList.Items {
		ms := &msList.Items[idx]
		if hasMatchingLabels(ms, m) {
			mss = append(mss, ms)
		}
	}
	return mss
}

// shouldExcludeMachine returns true if the machine should be filtered out, false otherwise.
func shouldExcludeMachine(machineSet *clusterv1alpha1.CnctMachineSet, machine *clusterv1alpha1.CnctMachine) bool {
	// Has an owneref to the wrong machineSet controller
	if metav1.GetControllerOf(machine) != nil && !metav1.IsControlledBy(machine, machineSet) {
		log.Info("Machine is not controlled by MachineSet", "machine",
			machine.Name, "machineset", machineSet.Name)
		return true
	}
	if machine.ObjectMeta.DeletionTimestamp != nil {
		return true
	}
	if !hasMatchingLabels(machineSet, machine) {
		return true
	}
	return false
}

func hasMatchingLabels(machineSet *clusterv1alpha1.CnctMachineSet, machine *clusterv1alpha1.CnctMachine) bool {
	selector, err := metav1.LabelSelectorAsSelector(&machineSet.Spec.Selector)
	if err != nil {
		log.Error(err, "Unable to convert selector")
		return false
	}
	// If a deployment with a nil or empty selector creeps in, it should match nothing, not everything.
	if selector.Empty() {
		log.Info("Machineset has empty selector", "machineSet", machineSet.Name)
		return false
	}
	if !selector.Matches(labels.Set(machine.Labels)) {
		log.Info("Machine does not match machineset selector labels", "machine", machine.Name)
		return false
	}
	return true
}
