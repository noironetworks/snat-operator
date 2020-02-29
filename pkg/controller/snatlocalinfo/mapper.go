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
	"encoding/json"
	"reflect"
	"strings"

	"github.com/noironetworks/snat-operator/cmd/manager/utils"

	aciv1 "github.com/noironetworks/snat-operator/pkg/apis/aci/v1"
	openv1 "github.com/openshift/api/apps/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
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
type handleDeployment struct {
	client     client.Client
	predicates []predicate.Predicate
}
type handleService struct {
	client     client.Client
	predicates []predicate.Predicate
}
type handleNamespace struct {
	client     client.Client
	predicates []predicate.Predicate
}

const (
	POD        = "pod"
	SERVICE    = "service"
	DEPLOYMENT = "deployment"
	NAMESPACE  = "namespace"
	CLUSTER    = "cluster"
)

type handleDeploymentConfig struct {
	client     client.Client
	predicates []predicate.Predicate
}

const pattern = "@$@"

// Given a reconcile request name, it extracts out pod name by omiiting snat-policy- from it
// eg: snat-policy-foo-podname -> podname, foo
func GetPodNameFromReoncileRequest(requestName string) (string, string, string, string) {

	var name, resName, resType, snatIp string
	temp := strings.Split(requestName, pattern)
	if len(temp) == 5 {
		name, resName, resType, snatIp = temp[1], temp[2], temp[3], temp[4]
	} else if len(temp) == 4 {
		name, resName, resType = temp[1], temp[2], temp[3]
	}
	return resName, name, resType, snatIp
}

func (h *handlePodsForPodsMapper) Map(obj handler.MapObject) []reconcile.Request {
	if obj.Object == nil {
		return nil
	}
	pod, ok := obj.Object.(*corev1.Pod)
	if !ok {
		return nil
	}
	MapperLog.V(1).Info("Inside pod map function", "pod is:", "node: ", "namespace",
		pod.ObjectMeta.Name+"--"+pod.Spec.NodeName+"---"+pod.ObjectMeta.Namespace)

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
	var requests []reconcile.Request
	MapperLog.V(1).Info("Mapper handling the  object", "SnatPolicy: ", obj.Object)
	if obj.Object == nil {
		return nil
	}

	snatpolicy, ok := obj.Object.(*aciv1.SnatPolicy)
	if !ok {
		return nil
	}
	snatPolicynew, err := utils.GetSnatPolicyCR(h.client, snatpolicy.ObjectMeta.Name)
	if err != nil && errors.IsNotFound(err) {
		return nil
	}
	// if polciy Spec is not equal means policy is updated
	if reflect.DeepEqual(snatpolicy.Spec, snatPolicynew.Spec) {
		// check to block all the port status updates
		if snatPolicynew.GetDeletionTimestamp() == nil && len(snatpolicy.Status.SnatPortsAllocated) != 0 {
			return nil
		}
		requests = append(requests, reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name: "snat-policy" + pattern + snatpolicy.ObjectMeta.Name +
					pattern + "created" + pattern + "new",
			},
		})
	} else {
		PolicyString, err := json.Marshal(snatpolicy)
		if err != nil {
			MapperLog.Error(err, "Failed to marshal snatip")
		}
		MapperLog.V(1).Info("SnatPolicy object is updated", "SnatPolicy: ", snatpolicy)
		requests = append(requests, reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name: "snat-policy" + pattern + snatpolicy.ObjectMeta.Name +
					pattern + "deleted" + pattern + string(PolicyString),
			},
		})
	}
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
	var resType string
Loop:
	for _, item := range snatPolicyList.Items {
		if item.GetDeletionTimestamp() != nil || item.Status.State == aciv1.Failed {
			continue
		}
		if reflect.DeepEqual(item.Spec.Selector, aciv1.PodSelector{}) {
			// Need to check here how to handle this.
			//item.Spec.Selector == aciv1.SnatPolicy.Spec.Selector{} {
			MapperLog.Info("Cluster Scoped snat polcy: ", "SnatIp: ", item.Spec.SnatIp)
			if !pod.Spec.HostNetwork {
				requests = append(requests, reconcile.Request{
					NamespacedName: types.NamespacedName{
						Namespace: pod.ObjectMeta.Namespace,
						Name: "snat-policyforpod" + pattern + item.ObjectMeta.Name + pattern +
							pod.ObjectMeta.Name + pattern + CLUSTER,
					},
				})
				if resType == "" {
					resType = CLUSTER
				}
			}
		} else {
			// This case will come when namespace is specified in the policy
			if item.Spec.Selector.Namespace != "" && item.Spec.Selector.Namespace != pod.ObjectMeta.Namespace {
				continue
			}

			// Now need to match pod with correct snatPolicy item
			// According to priority:
			// 1: Labels of pod should match exactly with labels of podSelector
			// 2: Deployment labels should match exactly with Policy labels
			// 3: Namespace labels should match exactly with Policy labels
			// right now namespace approach is implemented.

			// This case is for Services

			// This case is for Services where SnatIP is Service IP
			if len(item.Spec.SnatIp) == 0 {
				// Handle it for Services.
				SerivesList := &corev1.ServiceList{}
				err := c.List(context.TODO(),
					&client.ListOptions{
						Namespace: item.Spec.Selector.Namespace,
					},
					SerivesList)
				if err == nil {
					matches := false
					for _, service := range SerivesList.Items {
						if len(service.Status.LoadBalancer.Ingress) == 0 {
							continue
						}
						if utils.MatchLabels(service.Spec.Selector, pod.ObjectMeta.Labels) {
							if utils.MatchLabels(item.Spec.Selector.Labels, service.ObjectMeta.Labels) {
								resType = SERVICE
								matches = true
							} else if len(item.Spec.Selector.Labels) == 0 && item.Spec.Selector.Namespace == service.ObjectMeta.Namespace {
								resType = NAMESPACE
								matches = true
							}
						}
						if matches {
							requests = append(requests, reconcile.Request{
								NamespacedName: types.NamespacedName{
									Namespace: pod.ObjectMeta.Namespace,
									Name: "snat-policyforpod" + pattern + item.ObjectMeta.Name + pattern +
										pod.ObjectMeta.Name + pattern + resType + pattern + service.Status.LoadBalancer.Ingress[0].IP,
								},
							})
							if resType == POD || resType == SERVICE {
								break Loop
							}
						}
					}
				}
				continue
			}
			// This case if for no labels and policy applied on only namespace
			if len(item.Spec.Selector.Labels) == 0 {
				if item.Spec.Selector.Namespace == pod.ObjectMeta.Namespace {
					MapperLog.Info("Found pod:", "Matching namespace: ", item.Spec.Selector.Namespace)
					requests = append(requests, reconcile.Request{
						NamespacedName: types.NamespacedName{
							Namespace: pod.ObjectMeta.Namespace,
							Name: "snat-policyforpod" + pattern + item.ObjectMeta.Name + pattern +
								pod.ObjectMeta.Name + pattern + NAMESPACE,
						},
					})
					if resType == CLUSTER || resType == "" {
						resType = NAMESPACE
					}
					continue
				}
			}
			//These Cases will be matched with labels
			if utils.MatchLabels(item.Spec.Selector.Labels, pod.ObjectMeta.Labels) {
				MapperLog.Info("Found pod:", "Matching pod lables: ", pod.ObjectMeta.Labels)
				requests = append(requests, reconcile.Request{
					NamespacedName: types.NamespacedName{
						Namespace: pod.ObjectMeta.Namespace,
						Name: "snat-policyforpod" + pattern + item.ObjectMeta.Name +
							pattern + pod.ObjectMeta.Name + pattern + POD,
					},
				})
				resType = POD
				break Loop
			}

			replicaset := &appsv1.ReplicaSet{}
			for _, rs := range pod.OwnerReferences {
				if rs.Kind == "ReplicaSet" && rs.Name != "" {
					err := c.Get(context.TODO(), types.NamespacedName{Namespace: pod.ObjectMeta.Namespace, Name: rs.Name}, replicaset)
					if err != nil {
						//MapperLog.Error(err, "Replication get  Failed")
					}
					break
				}
			}
			var depname string
			for _, dep := range replicaset.OwnerReferences {
				if dep.Kind == "Deployment" && dep.Name != "" {
					depname = dep.Name
					break
				}
			}
			// Check for Matching Label for Deployment
			if depname != "" {
				deployment := &appsv1.Deployment{}
				// Need to revisit this code how to set proper NameSpace here
				err := c.Get(context.TODO(), types.NamespacedName{Namespace: pod.ObjectMeta.Namespace, Name: depname}, deployment)
				if err == nil {
					MapperLog.Info("Found pod:", "Matching deployment labels: ", deployment.ObjectMeta.Labels)
					if utils.MatchLabels(item.Spec.Selector.Labels, deployment.ObjectMeta.Labels) {
						requests = append(requests, reconcile.Request{
							NamespacedName: types.NamespacedName{
								Namespace: pod.ObjectMeta.Namespace,
								Name: "snat-policyforpod" + pattern + item.ObjectMeta.Name +
									pattern + pod.ObjectMeta.Name + pattern + DEPLOYMENT,
							},
						})
						if resType == CLUSTER || resType == NAMESPACE || resType == "" {
							resType = DEPLOYMENT
						}
						continue
					}
				} else if err != nil && errors.IsNotFound(err) {
					MapperLog.Info("No deployment found with", "DeploymentName: ", depname)
				} else if err != nil {
					//MapperLog.Error(err, " deployment get error")
				}
			}
			rc := &corev1.ReplicationController{}
			for _, rs := range pod.OwnerReferences {
				if rs.Kind == "ReplicationController" && rs.Name != "" {
					err := c.Get(context.TODO(), types.NamespacedName{Namespace: pod.ObjectMeta.Namespace, Name: rs.Name}, rc)
					if err != nil {
						//MapperLog.Error(err, "Replication get  Failed")
					}
					break
				}
			}
			// Check for Matching Label for Deployment Config
			var depconfname string
			for _, dep := range rc.OwnerReferences {
				if dep.Kind == "DeploymentConfig" && dep.Name != "" {
					depconfname = dep.Name
					break
				}
			}
			if depconfname != "" {
				deploymentconfig := &openv1.DeploymentConfig{}
				err := c.Get(context.TODO(), types.NamespacedName{Namespace: "default", Name: depconfname}, deploymentconfig)
				if err == nil {
					MapperLog.Info("Found pod:", "Matching deploymentConf Labels: ", deploymentconfig.ObjectMeta.Labels)
					if utils.MatchLabels(item.Spec.Selector.Labels, deploymentconfig.ObjectMeta.Labels) {
						requests = append(requests, reconcile.Request{
							NamespacedName: types.NamespacedName{
								Namespace: pod.ObjectMeta.Namespace,
								Name: "snat-policyforpod" + pattern + item.ObjectMeta.Name + pattern +
									pod.ObjectMeta.Name + pattern + DEPLOYMENT,
							},
						})
						if resType == CLUSTER || resType == NAMESPACE || resType == "" {
							resType = DEPLOYMENT
						}
						continue
					}
				} else if err != nil && errors.IsNotFound(err) {
					MapperLog.Info("No deploymentConfig found with", "DeploymentConfName: ", depconfname)
				} else if err != nil {
					//MapperLog.Error(err, " deployment get error")
				}
			}
			namespace := &corev1.Namespace{}
			err := c.Get(context.TODO(), types.NamespacedName{Name: pod.ObjectMeta.Namespace}, namespace)
			if err == nil {
				if utils.MatchLabels(item.Spec.Selector.Labels, namespace.ObjectMeta.Labels) {
					MapperLog.Info("Found Pod", " Matching nameSpace labels: ", namespace.ObjectMeta.Labels)
					requests = append(requests, reconcile.Request{
						NamespacedName: types.NamespacedName{
							Namespace: pod.ObjectMeta.Namespace,
							Name: "snat-policyforpod" + pattern + item.ObjectMeta.Name + pattern +
								pod.ObjectMeta.Name + pattern + NAMESPACE,
						},
					})
					if resType == CLUSTER || resType == "" {
						resType = NAMESPACE
					}
					continue
				}
			} else if err != nil && errors.IsNotFound(err) {
				MapperLog.Info("Obj", "No namespace found with: ", namespace)
			} else if err != nil {
				//MapperLog.Error(err, "namespace get error")
			}
		}
	}
	var newrequests []reconcile.Request
	for _, request := range requests {
		_, _, res, _ := GetPodNameFromReoncileRequest(request.Name)
		if resType == res {
			newrequests = append(newrequests, request)
			break
		}
	}
	return newrequests
}

func HandleDeploymentForDeploymentMapper(client client.Client, predicates []predicate.Predicate) handler.Mapper {
	return &handleDeployment{client, predicates}
}
func HandleNameSpaceForNameSpaceMapper(client client.Client, predicates []predicate.Predicate) handler.Mapper {
	return &handleNamespace{client, predicates}
}

func HandleDeploymentConfigForDeploymentMapper(client client.Client, predicates []predicate.Predicate) handler.Mapper {
	return &handleDeploymentConfig{client, predicates}
}
func HandleServicesForServiceMapper(client client.Client, predicates []predicate.Predicate) handler.Mapper {
	return &handleService{client, predicates}
}
func (h *handleDeployment) Map(obj handler.MapObject) []reconcile.Request {
	MapperLog.V(1).Info("Mapper handling obj", "Deployment: ", obj.Object)
	var requests []reconcile.Request
	curdep := &appsv1.Deployment{}
	var slectorString []byte
	if obj.Object == nil {
		return nil
	}
	deployment, ok := obj.Object.(*appsv1.Deployment)
	if !ok {
		return nil
	}
	err := h.client.Get(context.TODO(), types.NamespacedName{Namespace: deployment.ObjectMeta.Namespace,
		Name: deployment.ObjectMeta.Name}, curdep)
	if err == nil && curdep.GetDeletionTimestamp() == nil {
		slectorString, err = json.Marshal(deployment.ObjectMeta.Labels)
	} else {
		slectorString, err = json.Marshal(deployment.Spec.Selector.MatchLabels)
	}
	if err != nil {
		MapperLog.Error(err, "Failed to marshal slectorstring")
		return nil
	}
	requests = append(requests, reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name: "snat-deployment" + pattern + deployment.ObjectMeta.Namespace + pattern +
				deployment.ObjectMeta.Name + pattern + string(slectorString),
		},
	})
	return requests
}

func (h *handleDeploymentConfig) Map(obj handler.MapObject) []reconcile.Request {
	MapperLog.V(1).Info("Mapper handling obj", "deploymentConfig:", obj.Object)
	var requests []reconcile.Request
	curdep := &openv1.DeploymentConfig{}
	var slectorString []byte
	if obj.Object == nil {
		return nil
	}
	deployment, ok := obj.Object.(*openv1.DeploymentConfig)
	if !ok {
		return nil
	}
	err := h.client.Get(context.TODO(), types.NamespacedName{Namespace: deployment.ObjectMeta.Namespace,
		Name: deployment.ObjectMeta.Name}, curdep)
	if err == nil && curdep.GetDeletionTimestamp() == nil {
		// otherwise set the updated label
		slectorString, err = json.Marshal(deployment.ObjectMeta.Labels)
	} else {
		//set the selector in case of delete of deployment
		slectorString, err = json.Marshal(deployment.Spec.Selector)
	}
	if err != nil {
		MapperLog.Error(err, "Failed to marshal slector")
		return nil
	}
	requests = append(requests, reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name: "snat-deploymentconfig" + pattern + deployment.ObjectMeta.Namespace + pattern +
				deployment.ObjectMeta.Name + pattern + string(slectorString),
		},
	})
	return requests
}

func (h *handleNamespace) Map(obj handler.MapObject) []reconcile.Request {
	var requests []reconcile.Request
	curnamespace := &corev1.Namespace{}
	var slectorString []byte
	MapperLog.V(1).Info("Mapper handling obj", "Namespace: ", obj.Object)
	if obj.Object == nil {
		return nil
	}
	namespace, ok := obj.Object.(*corev1.Namespace)
	if !ok {
		return nil
	}
	err := h.client.Get(context.TODO(), types.NamespacedName{Name: namespace.ObjectMeta.Name}, curnamespace)
	if err == nil && curnamespace.GetDeletionTimestamp() == nil {
		slectorString, err = json.Marshal(namespace.ObjectMeta.Labels)
		if err != nil {
			MapperLog.Error(err, "Failed to marshal slectorString")
			return nil
		}
	}

	requests = append(requests, reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name: "snat-namespace" + pattern + "labelchange" + pattern +
				namespace.ObjectMeta.Name + pattern + string(slectorString),
		},
	})
	return requests
}

func (h *handleService) Map(obj handler.MapObject) []reconcile.Request {
	var requests []reconcile.Request
	MapperLog.V(1).Info("Mapper handling obj: ", "Service: ", obj.Object)
	if obj.Object == nil {
		return nil
	}
	service, ok := obj.Object.(*corev1.Service)
	if !ok {
		return nil
	}
	slectorString, err := json.Marshal(service.Spec.Selector)
	if err != nil {
		MapperLog.Error(err, "Failed to marshal snatip")
		return nil
	}
	requests = append(requests, reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name: "snat-service" + pattern + service.ObjectMeta.Namespace + pattern +
				service.ObjectMeta.Name + pattern + string(slectorString),
		},
	})
	return requests
}
