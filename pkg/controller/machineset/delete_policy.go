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
	"sort"

	clusterv1alpha1 "github.com/samsung-cnct/cma-ssh/pkg/apis/cluster/v1alpha1"
)

type (
	deletePriority     float64
	deletePriorityFunc func(machine *clusterv1alpha1.CnctMachine) deletePriority
)

const (
	mustDelete   deletePriority = 100.0
	betterDelete deletePriority = 50.0
	couldDelete  deletePriority = 20.0

	// DeleteNodeAnnotation marks nodes that will be given priority for deletion
	// when a machineset scales down. This annotation is given top priority on all delete policies.
	DeleteNodeAnnotation = "cluster.k8s.io/delete-machine"
)

func randomDeletePolicy(machine *clusterv1alpha1.CnctMachine) deletePriority {
	if machine.DeletionTimestamp != nil && !machine.DeletionTimestamp.IsZero() {
		return mustDelete
	}
	if machine.ObjectMeta.Annotations != nil && machine.ObjectMeta.Annotations[DeleteNodeAnnotation] != "" {
		return betterDelete
	}
	return couldDelete
}

type sortableMachines struct {
	machines []*clusterv1alpha1.CnctMachine
	priority deletePriorityFunc
}

func (m sortableMachines) Len() int      { return len(m.machines) }
func (m sortableMachines) Swap(i, j int) { m.machines[i], m.machines[j] = m.machines[j], m.machines[i] }
func (m sortableMachines) Less(i, j int) bool {
	return m.priority(m.machines[j]) < m.priority(m.machines[i]) // high to low
}

func getMachinesToDeletePrioritized(filteredMachines []*clusterv1alpha1.CnctMachine, diff int, fun deletePriorityFunc) []*clusterv1alpha1.CnctMachine {
	if diff >= len(filteredMachines) {
		return filteredMachines
	} else if diff <= 0 {
		return []*clusterv1alpha1.CnctMachine{}
	}
	sortable := sortableMachines{
		machines: filteredMachines,
		priority: fun,
	}
	sort.Sort(sortable)
	return sortable.machines[:diff]
}
