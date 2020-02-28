// Copyright 2019 Cisco Systems, Inc.
//
// Licensed under the Apache License, Version 2.0 (the License);
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an AS IS BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
// implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package snatpolicy

import (
	"context"
	"math/rand"
	"strconv"
	"testing"

	aciv1 "github.com/noironetworks/snat-operator/pkg/apis/aci/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

// TestSnatPolicyController runs ReconcileSnatPolicy.Reconcile() against a
// fake client that tracks a snatpolicy object.
func TestSnatPolicyController(t *testing.T) {
	// Set the logger to development mode for verbose logs.
	logf.SetLogger(logf.ZapLogger(true))
	var (
		name      = "example-snatpolicy1"
		namespace = "snat"
	)
	expsnatplicy := &aciv1.SnatPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: aciv1.SnatPolicySpec{
			SnatIp: []string{"10.3.4.5"},
		},
	}
	snatpolicylist := &aciv1.SnatPolicyList{}
	snatlocalinfo := &aciv1.SnatLocalInfo{}
	snatlocalinfolist := &aciv1.SnatLocalInfoList{}
	// Objects to track in the fake client.
	objs := []runtime.Object{
		expsnatplicy,
	}
	// Register operator types with the runtime scheme.
	s := scheme.Scheme
	s.AddKnownTypes(aciv1.SchemeGroupVersion, expsnatplicy)
	s.AddKnownTypes(aciv1.SchemeGroupVersion, snatpolicylist)
	s.AddKnownTypes(aciv1.SchemeGroupVersion, snatlocalinfo)
	s.AddKnownTypes(aciv1.SchemeGroupVersion, snatlocalinfolist)

	// Create a fake client to mock API calls.
	cl := fake.NewFakeClient(objs...)
	pod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
		},
	}
	podNames := make([]string, 3)
	for i := 0; i < 3; i++ {
		pod.ObjectMeta.Name = name + ".pod." + strconv.Itoa(rand.Int())
		podNames[i] = pod.ObjectMeta.Name
		if err := cl.Create(context.TODO(), pod.DeepCopy()); err != nil {
			t.Fatalf("create pod %d: (%v)", i, err)
		}
	}
	// Create a ReconcileSnatPolicy object with the scheme and fake client.
	r := &ReconcileSnatPolicy{client: cl, scheme: s}
	// Mock request to simulate Reconcile() being called on an event for a
	// watched resource .
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      name,
			Namespace: namespace,
		},
	}
	_, err := r.Reconcile(req)
	if err != nil {
		t.Fatalf("reconcile: (%v)", err)
	}
	snatpolicy := &aciv1.SnatPolicy{}
	err = r.client.Get(context.TODO(), req.NamespacedName, snatpolicy)
	if err != nil {
		t.Fatalf("reconcile: (%v)", err)
	}

	t.Log("info", snatpolicy)

	if snatpolicy.Status.State != aciv1.Ready {
		t.Fatalf("Failed to Set the state: ")
	} else {
		t.Log("Policy Status is Ready")
	}

}
