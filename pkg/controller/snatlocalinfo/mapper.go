package snatlocalinfo

import (
	"context"

	aciv1 "github.com/noironetworks/snat-operator/pkg/apis/aci/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var MapperLog = logf.Log.WithName("Mapper:")

type handlePodsForPodsMapper struct {
	client     client.Client
	predicates []predicate.Predicate
}
type handleSnatPoliciesMapper struct {
	client     client.Client
	predicates []predicate.Predicate
}

func (h *handlePodsForPodsMapper) Map(obj handler.MapObject) []reconcile.Request {
	if obj.Object == nil {
		return nil
	}

	pod, ok := obj.Object.(*corev1.Pod)
	if !ok {
		return nil
	}
	// MapperLog.Info("Inside pod map function", "Pod is:", pod.ObjectMeta.Name+"--"+pod.Spec.NodeName+"---"+pod.ObjectMeta.Namespace)

	// Get all the snatpolicies
	snatPolicyList := &aciv1.SnatPolicyList{}
	if err := h.client.List(context.TODO(), &client.ListOptions{Namespace: ""}, snatPolicyList); err != nil {
		return nil
	}

	requests := FilterPodsPerSnatPolicy(h.client, snatPolicyList, pod)
	return requests
}

func HandlePodsForPodsMapper(client client.Client, predicates []predicate.Predicate) handler.Mapper {
	return &handlePodsForPodsMapper{client, predicates}
}
func (h *handleSnatPoliciesMapper) Map(obj handler.MapObject) []reconcile.Request {
	MapperLog.Info("Snat Polcies Info Obj", "mapper handling first ###", obj.Object)
	if obj.Object == nil {
		return nil
	}

	snatpolicy, ok := obj.Object.(*aciv1.SnatPolicy)
	MapperLog.Info("Local Info Obj", "mapper handling ###", snatpolicy)
	if !ok {
		return nil
	}

	var requests []reconcile.Request
	requests = append(requests, reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name: "snat-policy-" + snatpolicy.ObjectMeta.Name + "-" + "created" + "-" + "new",
		},
	})
	return requests
}

func HandleSnatPolicies(client client.Client, predicates []predicate.Predicate) handler.Mapper {
	return &handleSnatPoliciesMapper{client, predicates}
}

/*

Given a list of SnatPolicy CR, this function filters out whether the pod should be included in reconcile request of
Map or not, based on CR spec of SnatPolicy from the list
*/
func FilterPodsPerSnatPolicy(c client.Client, snatPolicyList *aciv1.SnatPolicyList, pod *corev1.Pod) []reconcile.Request {
	var requests []reconcile.Request

Loop:
	for _, item := range snatPolicyList.Items {

		if false {
			// Need to check here how to handle this.
			//item.Spec.Selector == aciv1.SnatPolicy.Spec.Selector{} {
			MapperLog.Info("Cluster Scoped", "Needs special handling", item.Spec.SnatIp)
		} else {
			// Now need to match pod with correct snatPolicy item
			// According to priority:
			// 1: Labels of pod should match exactly with labels of podSelector
			// 2: Deployment of pod should match exactly with deployment of podSelector
			// 3: Namespace of pod should match exactly with namespace of podSelector
			// right now namespace approach is implemented.
			MapperLog.Info("Snat Polcies Info Obj", "mapper handling POD OBJ ###", pod)
			for _, label := range item.Spec.Selector.Labels {
				if _, ok := pod.ObjectMeta.Labels[label.Key]; ok {
					if label.Value == pod.ObjectMeta.Labels[label.Key] {
						requests = append(requests, reconcile.Request{
							NamespacedName: types.NamespacedName{
								Namespace: pod.ObjectMeta.Namespace,
								Name:      "snat-policyforpod-" + item.ObjectMeta.Name + "-" + pod.ObjectMeta.Name + "-" + "pod",
							},
						})
						break Loop
					}
				}
			}
			MapperLog.Info("Snat Polcies Info Obj", "mapper handling Deployment OBJ ###", pod)
			deployment := &appsv1.Deployment{}
			err := c.List(context.TODO(),
				&client.ListOptions{
					LabelSelector: labels.SelectorFromSet(pod.ObjectMeta.Labels),
				},
				deployment)
			if err == nil {
				MapperLog.Info("Snat Polcies Info Obj", "Labels for  Deployment OBJ ###", deployment.ObjectMeta.Labels)
				for _, label := range item.Spec.Selector.Labels {
					if _, ok := deployment.ObjectMeta.Labels[label.Key]; ok {
						if label.Value == deployment.ObjectMeta.Labels[label.Key] {
							requests = append(requests, reconcile.Request{
								NamespacedName: types.NamespacedName{
									Namespace: pod.ObjectMeta.Namespace,
									Name:      "snat-policyforpod-" + item.ObjectMeta.Name + "-" + pod.ObjectMeta.Name + "_" + "deployment",
								},
							})
							break Loop
						}
					}
				}
			} else {
				MapperLog.Error(err, "Deploymet error")
			}
			MapperLog.Info("Snat Polcies Info Obj", "mapper handling NameSpace OBJ ###", pod)
			namespace := &corev1.Namespace{}
			err = c.Get(context.TODO(), types.NamespacedName{Name: pod.ObjectMeta.Namespace}, namespace)
			if err == nil {
				for _, label := range item.Spec.Selector.Labels {
					if _, ok := namespace.ObjectMeta.Labels[label.Key]; ok {
						if label.Value == namespace.ObjectMeta.Labels[label.Key] {
							MapperLog.Info("Snat Polcies Matching", "Labels for  NameSpace OBJ ###", namespace.ObjectMeta.Labels)
							requests = append(requests, reconcile.Request{
								NamespacedName: types.NamespacedName{
									Namespace: pod.ObjectMeta.Namespace,
									Name:      "snat-policyforpod-" + item.ObjectMeta.Name + "-" + pod.ObjectMeta.Name + "-" + "namepsace",
								},
							})
							break Loop
						}
					}
				}
			} else {
				MapperLog.Error(err, "Name Space error")
			}
		}
	}
	return requests
}

func HandlePodsForDeployment(client client.Client, predicates []predicate.Predicate) handler.Mapper {
	return &handleSnatPoliciesMapper{client, predicates}
}
func HandlePodsForNameSpace(client client.Client, predicates []predicate.Predicate) handler.Mapper {
	return &handleSnatPoliciesMapper{client, predicates}
}
