/*
Copyright 2018 Samsung SDS.

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

package appbundle

import (
	"testing"
	"time"

	"github.com/onsi/gomega"
	addonsv1alpha1 "github.com/samsung-cnct/cma-ssh/pkg/apis/addons/v1alpha1"
	"golang.org/x/net/context"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var c client.Client

var expectedRequest = reconcile.Request{NamespacedName: types.NamespacedName{Name: "foo", Namespace: "default"}}
var expectedInstalledRequest = reconcile.Request{
	NamespacedName: types.NamespacedName{Name: "installed-app", Namespace: "installed"}}
var expectedEmptyStatus = addonsv1alpha1.AppBundleStatusPhase("")

const timeout = time.Second * 5

func TestReconcile(t *testing.T) {
	g := gomega.NewGomegaWithT(t)
	instance := &addonsv1alpha1.AppBundle{ObjectMeta: metav1.ObjectMeta{Name: "foo", Namespace: "default"}}

	// Setup the Manager and Controller.  Wrap the Controller Reconcile function so it writes each request to a
	// channel when it is finished.
	mgr, err := manager.New(cfg, manager.Options{})
	g.Expect(err).NotTo(gomega.HaveOccurred())
	c = mgr.GetClient()

	recFn, requests := SetupTestReconcile(newReconciler(mgr))
	g.Expect(add(mgr, recFn)).NotTo(gomega.HaveOccurred())

	stopMgr, mgrStopped := StartTestManager(mgr, g)

	defer func() {
		close(stopMgr)
		mgrStopped.Wait()
	}()

	// Create the AppBundle object and expect the status to be ""
	err = c.Create(context.TODO(), instance)
	// The instance object may not be a valid object because it might be missing some required fields.
	// Please modify the instance object by adding required fields and then remove the following if statement.
	if apierrors.IsInvalid(err) {
		t.Logf("failed to create object, got an invalid object error: %v", err)
		return
	}
	g.Expect(err).NotTo(gomega.HaveOccurred())
	defer c.Delete(context.TODO(), instance)
	g.Eventually(requests, timeout).Should(gomega.Receive(gomega.Equal(expectedRequest)))

	appBundle := &addonsv1alpha1.AppBundle{}
	err = c.Get(context.Background(), types.NamespacedName{Name: "foo", Namespace: "default"}, appBundle)
	g.Expect(appBundle.Status.Phase).Should(gomega.Equal(expectedEmptyStatus))

	// Test already installed app
	installedAppBundle := &addonsv1alpha1.AppBundle{
		ObjectMeta: metav1.ObjectMeta{Name: "installed-app", Namespace: "installed"},
		Status:     addonsv1alpha1.AppBundleStatus{Phase: addonsv1alpha1.InstalledAppBundlePhase},
	}
	err = c.Create(context.Background(), installedAppBundle)
	if apierrors.IsInvalid(err) {
		t.Logf("failed to create installed object, got an invalid object error: %v", err)
		return
	}
	g.Expect(err).NotTo(gomega.HaveOccurred())
	defer c.Delete(context.Background(), installedAppBundle)
	g.Eventually(requests, timeout).Should(gomega.Receive(gomega.Equal(expectedInstalledRequest)))
}
