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
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

// TestSnatPolicyController runs ReconcileSnatPolicy.Reconcile() against a
// fake client that tracks a snatpolicy object.
func TestSnatLocalInfoController(t *testing.T) {
	// Set the logger to development mode for verbose logs.
	logf.SetLogger(logf.ZapLogger(true))
	var (
		name      = "example-snatpolicy1"
		namespace = "snat"
		nodename  = "testnode"
		snatip    = "10.3.4.5"
	)
	expsnatplicy := &aciv1.SnatPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: aciv1.SnatPolicySpec{
			SnatIp: []string{snatip},
		},
		Status: aciv1.SnatPolicyStatus{
			State: "Ready",
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
			t.Fatalf("create pod %d: (%v)", i, err)
		}
	}

	// Create a ReconcileSnatPolicy object with the scheme and fake client.
	r := &ReconcileSnatLocalInfo{client: cl, scheme: s}
	// Mock request to simulate Reconcile() being called on an event for a
	// watched resource .
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

	*snatlocalinfo, err = utils.GetLocalInfoCR(cl, nodename, "default")
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
