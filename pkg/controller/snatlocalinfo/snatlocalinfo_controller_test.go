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
package snatlocalinfo

import (
	"context"
	"testing"

	uuid "github.com/google/uuid"
	"github.com/noironetworks/snat-operator/cmd/manager/utils"
	aciv1 "github.com/noironetworks/snat-operator/pkg/apis/aci/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

const snatPolicyFinalizer = "finalizer.snatpolicy.aci.snat"

var (
	name      = "example-snatpolicy"
	namespace = "snat"
	nodename  = "testnode"
	snatip    = "10.3.4.5"
)

func CreatePods(namespace string, nodename string, cl client.Client) error {
	pod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Labels:    map[string]string{"key": "value"},
		},
		Spec: corev1.PodSpec{
			NodeName: nodename,
		},
		Status: corev1.PodStatus{
			Phase: "Running",
		},
	}
	podNames := make([]string, 3)
	for i := 0; i < 3; i++ {
		testid, _ := uuid.NewUUID()
		pod.ObjectMeta.UID = types.UID(testid.String())
		pod.ObjectMeta.Name = "test" + ".pod." + string(pod.ObjectMeta.UID)
		podNames[i] = pod.ObjectMeta.Name
		if err := cl.Create(context.TODO(), pod.DeepCopy()); err != nil {
			return err
		}
	}
	return nil
}

func CreatService(namespace string, cl client.Client) error {
	ingress := make([]corev1.LoadBalancerIngress, 5)
	ingress[0].IP = "10.20.30.40"
	service := corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "testservice",
			Namespace: namespace,
			Labels:    map[string]string{"key": "value"},
		},
		Spec: corev1.ServiceSpec{
			LoadBalancerIP: "10.20.30.40",
		},
		Status: corev1.ServiceStatus{
			LoadBalancer: corev1.LoadBalancerStatus{
				Ingress: ingress, /*[]corev1.LoadBalancerIngress{"10.20.30.40", nodename}*/
			},
		},
	}
	if err := cl.Create(context.TODO(), service.DeepCopy()); err != nil {
		return err
	}
	return nil
}

func CreateSchema(s *runtime.Scheme) {
	// Register operator types with the runtime scheme.
	s.AddKnownTypes(aciv1.SchemeGroupVersion, &aciv1.SnatPolicy{})
	s.AddKnownTypes(aciv1.SchemeGroupVersion, &aciv1.SnatPolicyList{})
	s.AddKnownTypes(aciv1.SchemeGroupVersion, &aciv1.SnatLocalInfo{})
	s.AddKnownTypes(aciv1.SchemeGroupVersion, &aciv1.SnatLocalInfoList{})
}

func CreateSnatpolicy(snatpolicy *aciv1.SnatPolicy, t *testing.T, cl *client.Client) (*aciv1.SnatLocalInfo, error) {

	// Register operator types with the runtime scheme.
	// Objects to track in the fake client.
	objs := []runtime.Object{
		snatpolicy,
	}
	s := scheme.Scheme
	CreateSchema(s)
	*cl = fake.NewFakeClient(objs...)
	err := CreatePods(namespace, nodename, *cl)

	if err != nil {
		t.Fatalf("create pod: (%v)", err)
	}
	err = CreatService(namespace, *cl)
	if err != nil {
		t.Fatalf("create Service: (%v)", err)
	}
	// Create a ReconcileSnatPolicy object with the scheme and fake client.
	r := &ReconcileSnatLocalInfo{client: *cl, scheme: s}
	// Mock request to simulate Reconcile() being called on an event for a
	// watched resource .
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name: "snat-policy" + pattern + name +
				pattern + "created" + pattern + "new",
		},
	}
	_, err = r.Reconcile(req)
	if err != nil {
		t.Fatalf("reconcile: (%v)", err)
	}
	snatlocalinfo := &aciv1.SnatLocalInfo{}
	*snatlocalinfo, err = utils.GetLocalInfoCR(*cl, nodename, "default")
	if err != nil {
		t.Fatalf("reconcile: (%v)", err)
	}
	return snatlocalinfo, err
}

func DeleteSnatPolicy(snatpolicy *aciv1.SnatPolicy, t *testing.T, cl client.Client) (*aciv1.SnatLocalInfo, error) {
	// Objects to track in the fake client.
	s := scheme.Scheme
	CreateSchema(s)
	// Create a ReconcileSnatPolicy object with the scheme and fake client.
	r := &ReconcileSnatLocalInfo{client: cl, scheme: s}
	// Mock request to simulate Reconcile() being called on an event for a
	// watched resource .
	var test metav1.Time
	test1 := test.Rfc3339Copy()
	snatPolicy := &aciv1.SnatPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:              name,
			DeletionTimestamp: &test1,
		},
		Spec: aciv1.SnatPolicySpec{
			SnatIp: []string{snatip},
			Selector: aciv1.PodSelector{
				Namespace: namespace,
				Labels:    map[string]string{"key": "value"},
			},
		},
		Status: aciv1.SnatPolicyStatus{
			State: "Ready",
		},
	}
	cl.Update(context.TODO(), snatPolicy)
	//snatPolicy, _ = utils.GetSnatPolicyCR(cl, name)
	if snatPolicy.GetDeletionTimestamp() != nil {
		//t.Fatalf("reconcile: (%v)", snatPolicy.GetDeletionTimestamp())
	}
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name: "snat-policy" + pattern + name +
				pattern + "created" + pattern + "new",
		},
	}
	_, err := r.Reconcile(req)
	if err != nil {
		t.Fatalf("reconcile: (%v)", err)
	}
	snatlocalinfo := &aciv1.SnatLocalInfo{}
	*snatlocalinfo, err = utils.GetLocalInfoCR(cl, nodename, "default")
	if err != nil {
		t.Fatalf("reconcile: (%v)", err)
	}
	cl.Delete(context.TODO(), snatpolicy)
	return snatlocalinfo, err

}

// TestSnatPolicyController runs ReconcileSnatPolicy.Reconcile() against a
// fake client that tracks a snatpolicy object.
func TestSnatLocalInfoController_Cluster(t *testing.T) {
	// Set the logger to development mode for verbose logs.
	logf.SetLogger(logf.ZapLogger(true))
	expsnatplicy := &aciv1.SnatPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: aciv1.SnatPolicySpec{
			SnatIp: []string{snatip},
		},
		Status: aciv1.SnatPolicyStatus{
			State: "Ready",
		},
	}

	s := scheme.Scheme
	CreateSchema(s)
	var cl client.Client
	snatlocalinfo, err := CreateSnatpolicy(expsnatplicy, t, &cl)
	if err != nil {
		t.Fatalf("reconcile: (%v)", err)
	}
	if len(snatlocalinfo.Spec.LocalInfos) != 3 {
		t.Fatalf("reconcile: (%v)", snatlocalinfo.Spec.LocalInfos)
	}
	for _, localinfo := range snatlocalinfo.Spec.LocalInfos {
		if localinfo.SnatScope != CLUSTER {
			t.Fatalf("reconcile: (%v)", snatlocalinfo.Spec.LocalInfos)
		}
		if localinfo.SnatIp != snatip {
			t.Fatalf("reconcile: (%v)", snatlocalinfo.Spec.LocalInfos)
		}
	}

}

func TestSnatLocalInfoController_NamSpace(t *testing.T) {
	// Set the logger to development mode for verbose logs.
	logf.SetLogger(logf.ZapLogger(true))
	expsnatplicy := &aciv1.SnatPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: aciv1.SnatPolicySpec{
			SnatIp: []string{snatip},
			Selector: aciv1.PodSelector{
				Namespace: namespace,
			},
		},
		Status: aciv1.SnatPolicyStatus{
			State: "Ready",
		},
	}
	var cl client.Client
	snatlocalinfo, err := CreateSnatpolicy(expsnatplicy, t, &cl)
	if err != nil {
		t.Fatalf("reconcile: (%v)", err)
	}
	if len(snatlocalinfo.Spec.LocalInfos) != 3 {
		t.Fatalf("reconcile: (%v)", snatlocalinfo.Spec.LocalInfos)
	}
	for _, localinfo := range snatlocalinfo.Spec.LocalInfos {
		if localinfo.SnatScope != NAMESPACE {
			t.Fatalf("reconcile: (%v)", snatlocalinfo.Spec.LocalInfos)
		}
		if localinfo.SnatIp != snatip {
			t.Fatalf("reconcile: (%v)", snatlocalinfo.Spec.LocalInfos)
		}
	}
}

func TestSnatLocalInfoController_NameSpace_withlabel(t *testing.T) {
	// Set the logger to development mode for verbose logs.
	logf.SetLogger(logf.ZapLogger(true))
	expsnatplicy := &aciv1.SnatPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: aciv1.SnatPolicySpec{
			SnatIp: []string{snatip},
			Selector: aciv1.PodSelector{
				Namespace: namespace,
				Labels:    map[string]string{"key": "value"},
			},
		},
		Status: aciv1.SnatPolicyStatus{
			State: "Ready",
		},
	}
	var cl client.Client
	snatlocalinfo, err := CreateSnatpolicy(expsnatplicy, t, &cl)
	if err != nil {
		t.Fatalf("reconcile: (%v)", err)
	}
	if len(snatlocalinfo.Spec.LocalInfos) != 3 {
		t.Fatalf("reconcile: (%v)", snatlocalinfo.Spec.LocalInfos)
	}
	for _, localinfo := range snatlocalinfo.Spec.LocalInfos {
		if localinfo.SnatScope != POD {
			t.Fatalf("reconcile: (%v)", snatlocalinfo.Spec.LocalInfos)
		}
		if localinfo.SnatIp != snatip {
			t.Fatalf("reconcile: (%v)", snatlocalinfo.Spec.LocalInfos)
		}
	}

	/*	snatlocalinfo, err = DeleteSnatPolicy(expsnatplicy, t, cl)
		time.Sleep(time.Second * 2)
		if err != nil {
			t.Fatalf("reconcile: (%v)", err)
		}
		if len(snatlocalinfo.Spec.LocalInfos) != 0 {
			t.Fatalf("reconcile: (%v)", snatlocalinfo.Spec.LocalInfos)
		} */
}

func TestSnatLocalInfoController_Service(t *testing.T) {
	// Set the logger to development mode for verbose logs.
	logf.SetLogger(logf.ZapLogger(true))
	expsnatplicy := &aciv1.SnatPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: aciv1.SnatPolicySpec{
			Selector: aciv1.PodSelector{
				Namespace: namespace,
				Labels:    map[string]string{"key": "value"},
			},
		},
		Status: aciv1.SnatPolicyStatus{
			State: "Ready",
		},
	}

	var cl client.Client
	snatlocalinfo, err := CreateSnatpolicy(expsnatplicy, t, &cl)
	if err != nil {
		t.Fatalf("reconcile: (%v)", err)
	}
	if len(snatlocalinfo.Spec.LocalInfos) != 3 {
		t.Fatalf("reconcile: (%v)", snatlocalinfo.Spec.LocalInfos)
	}
	for _, localinfo := range snatlocalinfo.Spec.LocalInfos {
		if localinfo.SnatScope != SERVICE {
			t.Fatalf("reconcile: (%v)", snatlocalinfo.Spec.LocalInfos)
		}
	}
}
func TestSnatLocalInfoController_Service_withoutlabel(t *testing.T) {
	// Set the logger to development mode for verbose logs.
	logf.SetLogger(logf.ZapLogger(true))
	expsnatplicy := &aciv1.SnatPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: aciv1.SnatPolicySpec{
			Selector: aciv1.PodSelector{
				Namespace: namespace,
			},
		},
		Status: aciv1.SnatPolicyStatus{
			State: "Ready",
		},
	}
	var cl client.Client
	snatlocalinfo, err := CreateSnatpolicy(expsnatplicy, t, &cl)
	if err != nil {
		t.Fatalf("reconcile: (%v)", err)
	}
	if len(snatlocalinfo.Spec.LocalInfos) != 3 {
		t.Fatalf("reconcile: (%v)", snatlocalinfo.Spec.LocalInfos)
	}
	for _, localinfo := range snatlocalinfo.Spec.LocalInfos {
		if localinfo.SnatScope != NAMESPACE {
			t.Fatalf("reconcile: (%v)", snatlocalinfo.Spec.LocalInfos)
		}
	}
}
