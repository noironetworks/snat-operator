package e2e

import (
	goctx "context"
	"testing"
	"time"

	"github.com/noironetworks/snat-operator/cmd/manager/utils"
	apis "github.com/noironetworks/snat-operator/pkg/apis"
	aciv1 "github.com/noironetworks/snat-operator/pkg/apis/aci/v1"
	framework "github.com/operator-framework/operator-sdk/pkg/test"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	retryInterval        = time.Second * 5
	timeout              = time.Second * 60
	cleanupRetryInterval = time.Second * 1
	cleanupTimeout       = time.Second * 5
)

var (
	name       = "busybox1"
	namespace  = "default"
	nodename   = "minikube"
	snatip     = "10.3.4.5"
	macaddress = "a0:36:9f:84:91:6e"
)

func TestSnatOperator(t *testing.T) {
	err := framework.AddToFrameworkScheme(apis.AddToScheme, &aciv1.SnatPolicy{})
	err = framework.AddToFrameworkScheme(apis.AddToScheme, &aciv1.SnatPolicyList{})
	err = framework.AddToFrameworkScheme(apis.AddToScheme, &aciv1.SnatLocalInfo{})
	err = framework.AddToFrameworkScheme(apis.AddToScheme, &aciv1.SnatGlobalInfo{})
	err = framework.AddToFrameworkScheme(apis.AddToScheme, &appsv1.Deployment{})
	err = framework.AddToFrameworkScheme(apis.AddToScheme, &corev1.Namespace{})
	err = framework.AddToFrameworkScheme(apis.AddToScheme, &corev1.Pod{})
	if err != nil {
		t.Fatalf("failed to add custom resource scheme to framework: %v", err)
	}
	// run subtests
	t.Run("snat-group", func(t *testing.T) {
		t.Run("Cluster", SnatOperatorTest)
	})

}

// deploymentForSnatPolicy returns a  Deployment object
func deploymentForSnatPolicy() *appsv1.Deployment {
	var replicas int32
	replicas = 3
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    map[string]string{"key": "value"},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "busybox"},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": "busybox"},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Image:   "busybox",
						Name:    "busybox",
						Command: []string{"busybox", "-m=64", "-o", "modern", "-v"},
						Ports: []corev1.ContainerPort{{
							ContainerPort: 11211,
							Name:          "busybox",
						}},
					}},
				},
			},
		},
	}
	return dep
}

func snatCreateNodeInfo(f *framework.Framework, ctx *framework.TestCtx) error {
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
	err := f.Client.Create(goctx.TODO(), nodeinfo, &framework.CleanupOptions{TestContext: ctx, Timeout: cleanupTimeout, RetryInterval: cleanupRetryInterval})
	if err != nil {
		return err
	}
	return nil
}

func snatClusterpolicy(t *testing.T, f *framework.Framework, ctx *framework.TestCtx) error {
	t.Logf("snatClusterpolicy started")
	expsnatpolicy := &aciv1.SnatPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "snatpolicy",
			Namespace: namespace,
		},
		Spec: aciv1.SnatPolicySpec{
			SnatIp: []string{"10.3.4.5"},
		},
	}
	// use TestCtx's create helper to screate the object and add a cleanup function for the new object
	err := f.Client.Create(goctx.TODO(), expsnatpolicy, &framework.CleanupOptions{TestContext: ctx, Timeout: cleanupTimeout, RetryInterval: cleanupRetryInterval})
	if err != nil {
		return err
	}
	dep := deploymentForSnatPolicy()
	if err = f.Client.Create(goctx.TODO(), dep, &framework.CleanupOptions{TestContext: ctx, Timeout: cleanupTimeout, RetryInterval: cleanupRetryInterval}); err != nil {
		return err
	}
	time.Sleep(time.Second * 5)
	snatlocalinfo := &aciv1.SnatLocalInfo{}
	*snatlocalinfo, err = utils.GetLocalInfoCR(f.Client.Client, nodename, "default")
	if err != nil {
		t.Fatalf("reconcile: (%v)", err)
	}
	if len(snatlocalinfo.Spec.LocalInfos) < 3 {
		t.Fatalf("reconcile: (%v)", snatlocalinfo.Spec.LocalInfos)
	}
	for _, localinfo := range snatlocalinfo.Spec.LocalInfos {
		if localinfo.SnatScope != "cluster" {
			t.Fatalf("reconcile: (%v)", snatlocalinfo.Spec.LocalInfos)
		}
		if localinfo.SnatIp != snatip {
			t.Fatalf("reconcile: (%v)", snatlocalinfo.Spec.LocalInfos)
		}
	}
	err = f.Client.Delete(goctx.TODO(), expsnatpolicy)
	if err != nil {
		return err
	}
	time.Sleep(time.Second * 2)
	*snatlocalinfo, _ = utils.GetLocalInfoCR(f.Client.Client, nodename, "default")
	if len(snatlocalinfo.Spec.LocalInfos) != 0 {
		t.Fatalf("reconcile: (%v)", snatlocalinfo.Spec.LocalInfos)
	}
	return nil
}

func snatNamespacepolicy(t *testing.T, f *framework.Framework, ctx *framework.TestCtx) error {
	t.Logf("snatNamespacepolicy started")
	expsnatpolicy := &aciv1.SnatPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "snatpolicy-namespace",
			Namespace: namespace,
		},
		Spec: aciv1.SnatPolicySpec{
			SnatIp: []string{snatip},
			Selector: aciv1.PodSelector{
				Namespace: namespace,
			},
		},
	}
	// use TestCtx's create helper to screate the object and add a cleanup function for the new object
	err := f.Client.Create(goctx.TODO(), expsnatpolicy, &framework.CleanupOptions{TestContext: ctx, Timeout: cleanupTimeout, RetryInterval: cleanupRetryInterval})
	if err != nil {
		return err
	}
	time.Sleep(time.Second * 5)
	snatlocalinfo := &aciv1.SnatLocalInfo{}
	*snatlocalinfo, err = utils.GetLocalInfoCR(f.Client.Client, nodename, "default")
	if err != nil {
		t.Fatalf("reconcile: (%v)", err)
	}
	if len(snatlocalinfo.Spec.LocalInfos) < 3 {
		t.Fatalf("reconcile: (%v)", snatlocalinfo.Spec.LocalInfos)
	}
	for _, localinfo := range snatlocalinfo.Spec.LocalInfos {
		if localinfo.SnatScope != "namespace" {
			t.Fatalf("reconcile: (%v)", snatlocalinfo.Spec.LocalInfos)
		}
		if localinfo.SnatIp != snatip {
			t.Fatalf("reconcile: (%v)", snatlocalinfo.Spec.LocalInfos)
		}
	}
	err = f.Client.Delete(goctx.TODO(), expsnatpolicy)
	if err != nil {
		return err
	}
	time.Sleep(time.Second * 2)
	*snatlocalinfo, _ = utils.GetLocalInfoCR(f.Client.Client, nodename, "default")
	if len(snatlocalinfo.Spec.LocalInfos) != 0 {
		t.Fatalf("reconcile: (%v)", snatlocalinfo.Spec.LocalInfos)
	}
	return nil
}

func snatNamespacepolicy_withlabel(t *testing.T, f *framework.Framework, ctx *framework.TestCtx) error {
	t.Logf("snatNamespacepolicy_withlabel started")
	expsnatpolicy := &aciv1.SnatPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "snatpolicy-namespace-label",
			Namespace: namespace,
		},
		Spec: aciv1.SnatPolicySpec{
			SnatIp: []string{snatip},
			Selector: aciv1.PodSelector{
				Namespace: namespace,
				Labels:    map[string]string{"key": "value"},
			},
		},
	}
	// use TestCtx's create helper to screate the object and add a cleanup function for the new object
	err := f.Client.Create(goctx.TODO(), expsnatpolicy, &framework.CleanupOptions{TestContext: ctx, Timeout: cleanupTimeout, RetryInterval: cleanupRetryInterval})
	if err != nil {
		return err
	}

	count := 0
	snatlocalinfo := &aciv1.SnatLocalInfo{}
	for {
		*snatlocalinfo, err = utils.GetLocalInfoCR(f.Client.Client, nodename, "default")
		if err != nil {
			t.Fatalf("reconcile: (%v)", err)
		}
		if len(snatlocalinfo.Spec.LocalInfos) != 3 {
			if count < 3 {
				count++
				time.Sleep(time.Second * 4)
				continue
			} else {
				t.Fatalf("reconcile: (%v)", snatlocalinfo.Spec.LocalInfos)
				break
			}
		} else {
			break
		}
	}
	for _, localinfo := range snatlocalinfo.Spec.LocalInfos {
		if localinfo.SnatScope != "deployment" {
			t.Fatalf("reconcile: (%v)", snatlocalinfo.Spec.LocalInfos)
		}
		if localinfo.SnatIp != snatip {
			t.Fatalf("reconcile: (%v)", snatlocalinfo.Spec.LocalInfos)
		}
	}
	err = f.Client.Delete(goctx.TODO(), expsnatpolicy)
	if err != nil {
		return err
	}
	time.Sleep(time.Second * 2)
	*snatlocalinfo, _ = utils.GetLocalInfoCR(f.Client.Client, nodename, "default")
	if len(snatlocalinfo.Spec.LocalInfos) != 0 {
		t.Fatalf("reconcile: (%v)", snatlocalinfo.Spec.LocalInfos)
	}
	return nil
}

func SnatOperatorTest(t *testing.T) {
	t.Parallel()
	ctx := framework.NewTestCtx(t)
	defer ctx.Cleanup()
	err := ctx.InitializeClusterResources(&framework.CleanupOptions{TestContext: ctx, Timeout: cleanupTimeout, RetryInterval: cleanupRetryInterval})
	if err != nil {
		t.Fatalf("failed to initialize cluster resources: %v", err)
	}
	t.Log("Initialized cluster resources")
	/*namespace, err := ctx.GetNamespace()
	if err != nil {
		t.Fatal(err)
	}*/
	// get global framework variables
	f := framework.Global
	// wait for snat-operator to be ready
	/*err = e2eutil.WaitForOperatorDeployment(t, f.KubeClient, namespace, "snat-operator", 1, retryInterval, timeout)
	if err != nil {
		t.Fatal(err)
	}*/
	err = snatCreateNodeInfo(f, ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err = snatClusterpolicy(t, f, ctx); err != nil {
		t.Fatal(err)
	}
	if err = snatNamespacepolicy(t, f, ctx); err != nil {
		t.Fatal(err)
	}
	if err = snatNamespacepolicy_withlabel(t, f, ctx); err != nil {
		t.Fatal(err)
	}
}
