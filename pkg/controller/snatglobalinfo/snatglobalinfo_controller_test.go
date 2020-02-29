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
package snatglobalinfo

import (
	"context"
	"reflect"
	"testing"

	aciv1 "github.com/noironetworks/snat-operator/pkg/apis/aci/v1"
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
func TestSnatGlobalInfoController(t *testing.T) {
	// Set the logger to development mode for verbose logs.
	logf.SetLogger(logf.ZapLogger(true))
	var (
		namespace  = "default"
		nodename   = "testnode"
		policyname = "example-snatpolicy"
		macaddress = "a0:36:9f:84:91:6e"
		snatip     = "10.3.4.5"
	)
	localinfos := make(map[string]aciv1.LocalInfo)
	localinfo := aciv1.LocalInfo{}
	localinfo.PodName = "testpod"
	localinfo.PodNamespace = namespace
	localinfo.SnatPolicyName = policyname
	localinfo.SnatScope = "cluster"
	localinfo.SnatIp = "10.3.4.5"
	localinfos["6bddfe56-dda2-11e9-b79b-acde48001122"] = localinfo

	snalocalInfo := &aciv1.SnatLocalInfo{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nodename,
			Namespace: namespace,
		},
		Spec: aciv1.SnatLocalInfoSpec{
			LocalInfos: localinfos,
		},
	}
	nodeinfo := &aciv1.NodeInfo{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nodename,
			Namespace: namespace,
		},
		Spec: aciv1.NodeInfoSpec{
			Nodename:   nodename,
			Macaddress: macaddress,
		},
	}
	expsnatplicy := &aciv1.SnatPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      policyname,
			Namespace: namespace,
		},
		Spec: aciv1.SnatPolicySpec{
			SnatIp: []string{snatip},
		},
	}
	snatglobainfo := &aciv1.SnatGlobalInfo{}
	nodeinfoList := &aciv1.NodeInfoList{}
	// Objects to track in the fake client.
	objs := []runtime.Object{
		expsnatplicy,
		snalocalInfo,
		nodeinfo,
	}
	s := scheme.Scheme
	s.AddKnownTypes(aciv1.SchemeGroupVersion, snalocalInfo)
	s.AddKnownTypes(aciv1.SchemeGroupVersion, snatglobainfo)
	s.AddKnownTypes(aciv1.SchemeGroupVersion, nodeinfo)
	s.AddKnownTypes(aciv1.SchemeGroupVersion, nodeinfoList)
	s.AddKnownTypes(aciv1.SchemeGroupVersion, expsnatplicy)
	// Create a fake client to mock API calls.
	cl := fake.NewFakeClient(objs...)

	// Create a ReconcileSnatPolicy object with the scheme and fake client.
	r := &ReconcileSnatGlobalInfo{client: cl, scheme: s}
	// Mock request to simulate Reconcile() being called on an event for a
	// watched resource .
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Namespace: namespace,
			Name:      "snat-localinfo$" + nodename,
		},
	}
	_, err := r.Reconcile(req)
	if err != nil {
		t.Fatalf("reconcile: (%v)", err)
	}
	err = cl.Get(context.TODO(), types.NamespacedName{Name: "snatglobalinfo",
		Namespace: "default"}, snatglobainfo)
	if err != nil {
		t.Fatalf("reconcile: (%v)", err)
	}
	globalinfo := snatglobainfo.Spec.GlobalInfos[nodename]
	portrange := aciv1.PortRange{Start: 5000, End: 7999}
	if globalinfo[0].SnatIp != snatip {
		t.Fatalf("failed to set snatip: (%v)", globalinfo[0].SnatIp)
	}
	if !reflect.DeepEqual(globalinfo[0].PortRanges[0], portrange) {
		t.Fatalf(" failed to set portrange: (%v)", globalinfo[0].PortRanges[0])
	}
	if globalinfo[0].MacAddress != macaddress {
		t.Fatalf(" failed to set portrange: (%v)", globalinfo[0].MacAddress)
	}
}
