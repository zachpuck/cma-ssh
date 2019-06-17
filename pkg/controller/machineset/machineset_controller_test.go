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
	"reflect"
	"testing"
	"time"

	"github.com/onsi/gomega"
	clusterv1alpha1 "github.com/samsung-cnct/cma-ssh/pkg/apis/cluster/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const timeout = time.Second * 5

func TestReconcile(t *testing.T) {
	var expectedRequest = reconcile.Request{NamespacedName: types.NamespacedName{Name: "foo", Namespace: "default"}}
	var c client.Client
	g := gomega.NewGomegaWithT(t)
	instance := &clusterv1alpha1.CnctMachineSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foo",
			Namespace: "default",
		},
		Spec: clusterv1alpha1.MachineSetSpec{
			Replicas: 2,
			Selector: metav1.LabelSelector{
				MatchLabels: map[string]string{"foo": "bar"},
			},
			MachineTemplate: clusterv1alpha1.MachineTemplate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "default",
					Labels: map[string]string{
						"foo": "bar",
					},
				},
				Spec: clusterv1alpha1.MachineSpec{
					InstanceType: "standard",
				},
			},
		},
	}

	// Test ValidateMachineSet
	isvalid, err := ValidateMachineSet(instance)
	g.Expect(isvalid).To(gomega.BeTrue(), "MachineSet should be valid")

	// Setup the Manager and Controller.  Wrap the Controller Reconcile function so it writes each request to a
	// channel when it is finished.
	mgr, err := manager.New(cfg, manager.Options{})
	g.Expect(err).NotTo(gomega.HaveOccurred())
	c = mgr.GetClient()

	r := newReconciler(mgr)
	recFn, requests := SetupTestReconcile(r)

	g.Expect(add(mgr, recFn, r.MachineToMachineSets)).NotTo(gomega.HaveOccurred())

	stopMgr, mgrStopped := StartTestManagerGomega(mgr, g)

	defer func() {
		close(stopMgr)
		mgrStopped.Wait()
	}()

	// Create the CnctMachineSet object and expect the Reconcile and Deployment to be created
	err = c.Create(context.TODO(), instance)
	// The instance object may not be a valid object because it might be missing some required fields.
	// Please modify the instance object by adding required fields and then remove the following if statement.
	if apierrors.IsInvalid(err) {
		t.Logf("failed to create object, got an invalid object error: %v", err)
		return
	}
	g.Expect(err).NotTo(gomega.HaveOccurred())

	defer c.Delete(context.TODO(), instance)

	select {
	case recv := <-requests:
		if recv != expectedRequest {
			t.Error("received request does not match expected request")
		}
	case <-time.After(timeout):
		t.Error("timed out waiting for request")
	}
}

func TestMachineSetToMachines(t *testing.T) {
	machineSetList := &clusterv1alpha1.CnctMachineSetList{
		TypeMeta: metav1.TypeMeta{
			Kind: "CnctMachineSetList",
		},
		Items: []clusterv1alpha1.CnctMachineSet{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "withMatchingLabels",
					Namespace: "test",
				},
				Spec: clusterv1alpha1.MachineSetSpec{
					Selector: metav1.LabelSelector{
						MatchLabels: map[string]string{
							"foo": "bar",
						},
					},
				},
			},
		},
	}
	controller := true
	m := clusterv1alpha1.CnctMachine{
		TypeMeta: metav1.TypeMeta{
			Kind: "CnctMachine",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "withOwnerRef",
			Namespace: "test",
			OwnerReferences: []metav1.OwnerReference{
				{
					Name:       "Owner",
					Kind:       "CnctMachineSet",
					Controller: &controller,
				},
			},
		},
	}
	m2 := clusterv1alpha1.CnctMachine{
		TypeMeta: metav1.TypeMeta{
			Kind: "CnctMachine",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "noOwnerRefNoLabels",
			Namespace: "test",
		},
	}
	m3 := clusterv1alpha1.CnctMachine{
		TypeMeta: metav1.TypeMeta{
			Kind: "CnctMachine",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "withMatchingLabels",
			Namespace: "test",
			Labels: map[string]string{
				"foo": "bar",
			},
		},
	}
	testsCases := []struct {
		machine   clusterv1alpha1.CnctMachine
		mapObject handler.MapObject
		expected  []reconcile.Request
	}{
		{
			machine: m,
			mapObject: handler.MapObject{
				Meta:   m.GetObjectMeta(),
				Object: &m,
			},
			expected: []reconcile.Request{},
		},
		{
			machine: m2,
			mapObject: handler.MapObject{
				Meta:   m2.GetObjectMeta(),
				Object: &m2,
			},
			expected: nil,
		},
		{
			machine: m3,
			mapObject: handler.MapObject{
				Meta:   m3.GetObjectMeta(),
				Object: &m3,
			},
			expected: []reconcile.Request{
				{NamespacedName: client.ObjectKey{Namespace: "test", Name: "withMatchingLabels"}},
			},
		},
	}

	clusterv1alpha1.AddToScheme(scheme.Scheme)
	r := &ReconcileMachineSet{
		Client: fake.NewFakeClient(&m, &m2, &m3, machineSetList),
		scheme: scheme.Scheme,
	}

	for _, tc := range testsCases {
		got := r.MachineToMachineSets(tc.mapObject)
		if !reflect.DeepEqual(got, tc.expected) {
			t.Errorf("Case %s. Got: %v, expected: %v", tc.machine.Name, got, tc.expected)
		}
	}
}

func TestShouldExcludeMachine(t *testing.T) {
	controller := true
	testCases := []struct {
		machineSet clusterv1alpha1.CnctMachineSet
		machine    clusterv1alpha1.CnctMachine
		expected   bool
	}{
		{
			machineSet: clusterv1alpha1.CnctMachineSet{},
			machine: clusterv1alpha1.CnctMachine{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "withNoMatchingOwnerRef",
					Namespace: "test",
					OwnerReferences: []metav1.OwnerReference{
						{
							Name:       "Owner",
							Kind:       "CnctMachineSet",
							Controller: &controller,
						},
					},
				},
			},
			expected: true,
		},
		{
			machineSet: clusterv1alpha1.CnctMachineSet{
				Spec: clusterv1alpha1.MachineSetSpec{
					Selector: metav1.LabelSelector{
						MatchLabels: map[string]string{
							"foo": "bar",
						},
					},
				},
			},
			machine: clusterv1alpha1.CnctMachine{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "withMatchingLabels",
					Namespace: "test",
					Labels: map[string]string{
						"foo": "bar",
					},
				},
			},
			expected: false,
		},
		{
			machineSet: clusterv1alpha1.CnctMachineSet{},
			machine: clusterv1alpha1.CnctMachine{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "withDeletionTimestamp",
					Namespace:         "test",
					DeletionTimestamp: &metav1.Time{Time: time.Now()},
					Labels: map[string]string{
						"foo": "bar",
					},
				},
			},
			expected: true,
		},
	}

	for _, tc := range testCases {
		got := shouldExcludeMachine(&tc.machineSet, &tc.machine)
		if got != tc.expected {
			t.Errorf("Case %s. Got: %v, expected: %v", tc.machine.Name, got, tc.expected)
		}
	}
}

func TestAdoptOrphan(t *testing.T) {
	m := clusterv1alpha1.CnctMachine{
		ObjectMeta: metav1.ObjectMeta{
			Name: "orphanMachine",
		},
	}
	ms := clusterv1alpha1.CnctMachineSet{
		ObjectMeta: metav1.ObjectMeta{
			Name: "adoptOrphanMachine",
		},
	}
	controller := true
	blockOwnerDeletion := true
	testCases := []struct {
		machineSet clusterv1alpha1.CnctMachineSet
		machine    clusterv1alpha1.CnctMachine
		expected   []metav1.OwnerReference
	}{
		{
			machine:    m,
			machineSet: ms,
			expected: []metav1.OwnerReference{
				{
					APIVersion:         clusterv1alpha1.SchemeGroupVersion.String(),
					Kind:               "CnctMachineSet",
					Name:               "adoptOrphanMachine",
					UID:                "",
					Controller:         &controller,
					BlockOwnerDeletion: &blockOwnerDeletion,
				},
			},
		},
	}

	clusterv1alpha1.AddToScheme(scheme.Scheme)
	r := &ReconcileMachineSet{
		Client: fake.NewFakeClient(&m),
		scheme: scheme.Scheme,
	}
	for _, tc := range testCases {
		r.adoptOrphan(&tc.machineSet, &tc.machine)
		got := tc.machine.GetOwnerReferences()
		if !reflect.DeepEqual(got, tc.expected) {
			t.Errorf("Case %s. Got: %+v, expected: %+v", tc.machine.Name, got, tc.expected)
		}
	}
}
