package snatlocalinfo

import (
	"context"
	"encoding/json"
	"reflect"

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

func (h *handlePodsForPodsMapper) Map(obj handler.MapObject) []reconcile.Request {
	if obj.Object == nil {
		return nil
	}

	pod, ok := obj.Object.(*corev1.Pod)
	if !ok {
		return nil
	}
	MapperLog.Info("Inside pod map function", "Pod is:", pod.ObjectMeta.Name+"--"+pod.Spec.NodeName+"---"+pod.ObjectMeta.Namespace)

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
	MapperLog.Info("Snat Polcies Info Obj", "mapper handling first ###", obj.Object)
	if obj.Object == nil {
		return nil
	}

	snatpolicy, ok := obj.Object.(*aciv1.SnatPolicy)
	MapperLog.Info("Local Info Obj", "mapper handling ###", snatpolicy)
	if !ok {
		return nil
	}
	snatPolicynew, err := utils.GetSnatPolicyCR(h.client, snatpolicy.ObjectMeta.Name)
	if err != nil && errors.IsNotFound(err) {
		return nil
	}
	// if polciy Spec is not equal means policy is updated
	if reflect.DeepEqual(snatpolicy.Spec, snatPolicynew.Spec) {
		requests = append(requests, reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name: "snat-policy$" + snatpolicy.ObjectMeta.Name + "$" + "created" + "$" + "new",
			},
		})
	} else {
		PolicyString, err := json.Marshal(snatpolicy)
		if err != nil {
			MapperLog.Error(err, "Failed to Marshal snatIp")
		}
		MapperLog.Info("OldObj", "Object is Updated: ", snatpolicy)
		requests = append(requests, reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name: "snat-policy$" + snatpolicy.ObjectMeta.Name + "$" + "deleted" + "$" + string(PolicyString),
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
		if item.GetDeletionTimestamp() != nil || item.Status.State != aciv1.Ready {
			continue
		}
		if reflect.DeepEqual(item.Spec.Selector, aciv1.PodSelector{}) {
			// Need to check here how to handle this.
			//item.Spec.Selector == aciv1.SnatPolicy.Spec.Selector{} {
			MapperLog.Info("Cluster Scoped", "for snatIP", item.Spec.SnatIp)
			if !pod.Spec.HostNetwork {
				requests = append(requests, reconcile.Request{
					NamespacedName: types.NamespacedName{
						Namespace: pod.ObjectMeta.Namespace,
						Name:      "snat-policyforpod$" + item.ObjectMeta.Name + "$" + pod.ObjectMeta.Name + "$" + CLUSTER,
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
					if len(item.Spec.Selector.Labels) == 0 {
						for _, service := range SerivesList.Items {
							if reflect.DeepEqual(pod.ObjectMeta.Labels, service.Spec.Selector) {
								if len(service.Status.LoadBalancer.Ingress) == 0 {
									continue
								}
								requests = append(requests, reconcile.Request{
									NamespacedName: types.NamespacedName{
										Namespace: pod.ObjectMeta.Namespace,
										Name:      "snat-policyforservicepod$" + item.ObjectMeta.Name + "$" + pod.ObjectMeta.Name + "$" + service.Status.LoadBalancer.Ingress[0].IP,
									},
								})
								if resType != POD {
									resType = service.Status.LoadBalancer.Ingress[0].IP
								}
							}

						}
					} else {
						for _, service := range SerivesList.Items {
							if reflect.DeepEqual(pod.ObjectMeta.Labels, service.Spec.Selector) {
								if utils.MatchLabels(item.Spec.Selector.Labels, service.ObjectMeta.Labels) {
									if len(service.Status.LoadBalancer.Ingress) == 0 {
										continue
									}
									requests = append(requests, reconcile.Request{
										NamespacedName: types.NamespacedName{
											Namespace: pod.ObjectMeta.Namespace,
											Name:      "snat-policyforservicepod$" + item.ObjectMeta.Name + "$" + pod.ObjectMeta.Name + "$" + service.Status.LoadBalancer.Ingress[0].IP,
										},
									})
									if resType != POD {
										resType = service.Status.LoadBalancer.Ingress[0].IP
									}
									//break Loop
								}
							}
						}
					}
				}
				continue
			}
			// This case if for no labels and policy applied on only namespace
			if len(item.Spec.Selector.Labels) == 0 {
				if item.Spec.Selector.Namespace == pod.ObjectMeta.Namespace {
					MapperLog.Info("Pod Matching for", "NameSpace:", item.Spec.Selector.Namespace)
					requests = append(requests, reconcile.Request{
						NamespacedName: types.NamespacedName{
							Namespace: pod.ObjectMeta.Namespace,
							Name:      "snat-policyforpod$" + item.ObjectMeta.Name + "$" + pod.ObjectMeta.Name + "$" + NAMESPACE,
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
				MapperLog.Info("Snat Polcies Info Obj", "Matches Pod Lables###", pod.ObjectMeta.Labels)
				requests = append(requests, reconcile.Request{
					NamespacedName: types.NamespacedName{
						Namespace: pod.ObjectMeta.Namespace,
						Name:      "snat-policyforpod$" + item.ObjectMeta.Name + "$" + pod.ObjectMeta.Name + "$" + POD,
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
					MapperLog.Info("Snat Polcies Info Obj", "Labels for  Deployment OBJ ###", deployment.ObjectMeta.Labels)
					if utils.MatchLabels(item.Spec.Selector.Labels, deployment.ObjectMeta.Labels) {
						requests = append(requests, reconcile.Request{
							NamespacedName: types.NamespacedName{
								Namespace: pod.ObjectMeta.Namespace,
								Name:      "snat-policyforpod$" + item.ObjectMeta.Name + "$" + pod.ObjectMeta.Name + "$" + DEPLOYMENT,
							},
						})
						if resType == CLUSTER || resType == NAMESPACE || resType == "" {
							resType = DEPLOYMENT
						}
						continue
					}
				} else if err != nil && errors.IsNotFound(err) {
					MapperLog.Info("Obj", "No deployment found with: ", depname)
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
					MapperLog.Info("Snat Polcies Info Obj", "Labels for  DeploymentConf OBJ ###", deploymentconfig.ObjectMeta.Labels)
					if utils.MatchLabels(item.Spec.Selector.Labels, deploymentconfig.ObjectMeta.Labels) {
						requests = append(requests, reconcile.Request{
							NamespacedName: types.NamespacedName{
								Namespace: pod.ObjectMeta.Namespace,
								Name:      "snat-policyforpod$" + item.ObjectMeta.Name + "$" + pod.ObjectMeta.Name + "$" + DEPLOYMENT,
							},
						})
						if resType == CLUSTER || resType == NAMESPACE || resType == "" {
							resType = DEPLOYMENT
						}
						continue
					}
				} else if err != nil && errors.IsNotFound(err) {
					MapperLog.Info("Obj", "No deploymentConfig found with: ", depconfname)
				} else if err != nil {
					//MapperLog.Error(err, " deployment get error")
				}
			}
			MapperLog.Info("Snat Polcies Info Obj", "mapper handling NameSpace OBJ ###", pod)
			namespace := &corev1.Namespace{}
			err := c.Get(context.TODO(), types.NamespacedName{Name: pod.ObjectMeta.Namespace}, namespace)
			if err == nil {
				if utils.MatchLabels(item.Spec.Selector.Labels, namespace.ObjectMeta.Labels) {
					MapperLog.Info("Snat Polcies Matching", "Labels for  NameSpace OBJ ###", namespace.ObjectMeta.Labels)
					requests = append(requests, reconcile.Request{
						NamespacedName: types.NamespacedName{
							Namespace: pod.ObjectMeta.Namespace,
							Name:      "snat-policyforpod$" + item.ObjectMeta.Name + "$" + pod.ObjectMeta.Name + "$" + NAMESPACE,
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
		_, _, res := utils.GetPodNameFromReoncileRequest(request.Name)
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
	MapperLog.Info("Deployment  Info Obj", "mapper handling first ###", obj.Object)
	var requests []reconcile.Request
	if obj.Object == nil {
		return nil
	}
	deployment, ok := obj.Object.(*appsv1.Deployment)
	if !ok {
		return nil
	}
	slectorString, err := json.Marshal(deployment.Spec.Selector.MatchLabels)
	if err != nil {
		MapperLog.Error(err, "Failed to Marshal snatIp")
	}
	requests = append(requests, reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name: "snat-deployment$" + deployment.ObjectMeta.Namespace + "$" + deployment.ObjectMeta.Name + "$" + string(slectorString),
		},
	})
	return requests
}

func (h *handleDeploymentConfig) Map(obj handler.MapObject) []reconcile.Request {
	MapperLog.Info("DeploymentConfig  Info Obj", "mapper handling first ###", obj.Object)
	var requests []reconcile.Request
	if obj.Object == nil {
		return nil
	}
	deployment, ok := obj.Object.(*openv1.DeploymentConfig)
	if !ok {
		return nil
	}
	slectorString, err := json.Marshal(deployment.Spec.Selector)
	if err != nil {
		MapperLog.Error(err, "Failed to Marshal snatIp")
	}
	requests = append(requests, reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name: "snat-deploymentconfig$" + deployment.ObjectMeta.Namespace + "$" + deployment.ObjectMeta.Name + "$" + string(slectorString),
		},
	})
	return requests
}

func (h *handleNamespace) Map(obj handler.MapObject) []reconcile.Request {
	var requests []reconcile.Request
	MapperLog.Info("NameSpace Info Obj", "mapper handling first ###", obj.Object)
	if obj.Object == nil {
		return nil
	}

	namespace, ok := obj.Object.(*corev1.Namespace)
	if !ok {
		return nil
	}
	requests = append(requests, reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name: "snat-namespace$" + "labelchange" + "$" + namespace.ObjectMeta.Name + "$" + NAMESPACE,
		},
	})
	return requests
}

func (h *handleService) Map(obj handler.MapObject) []reconcile.Request {
	var requests []reconcile.Request
	MapperLog.Info("Service Info Obj", "mapper handling first ###", obj.Object)
	if obj.Object == nil {
		return nil
	}

	service, ok := obj.Object.(*corev1.Service)
	if !ok {
		return nil
	}
	slectorString, err := json.Marshal(service.Spec.Selector)
	if err != nil {
		MapperLog.Error(err, "Failed to Marshal snatIp")
	}
	requests = append(requests, reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name: "snat-service$" + service.ObjectMeta.Namespace + "$" + service.ObjectMeta.Name + "$" + string(slectorString),
		},
	})
	return requests
}
