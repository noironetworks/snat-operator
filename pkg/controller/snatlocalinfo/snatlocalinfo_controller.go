package snatlocalinfo

import (
	"context"
	"encoding/json"
	"os"
	"reflect"
	"strings"

	"github.com/noironetworks/snat-operator/cmd/manager/utils"
	aciv1 "github.com/noironetworks/snat-operator/pkg/apis/aci/v1"
	openv1 "github.com/openshift/api/apps/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_snatlocalinfo")

// Add creates a new SnatLocalInfo Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileSnatLocalInfo{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("snatlocalinfo-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource SnatLocalInfo
	err = c.Watch(&source.Kind{Type: &aciv1.SnatLocalInfo{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}
	// Watching for Pod changes
	err = c.Watch(&source.Kind{Type: &corev1.Pod{}},
		&handler.EnqueueRequestsFromMapFunc{ToRequests: HandlePodsForPodsMapper(mgr.GetClient(), []predicate.Predicate{})})
	if err != nil {
		return err
	}
	// Watching for Snat policy changes
	err = c.Watch(&source.Kind{Type: &aciv1.SnatPolicy{}},
		&handler.EnqueueRequestsFromMapFunc{ToRequests: HandleSnatPolicies(mgr.GetClient(), []predicate.Predicate{})})
	if err != nil {
		return err
	}
	// Watching for Deployment changes
	err = c.Watch(&source.Kind{Type: &appsv1.Deployment{}},
		&handler.EnqueueRequestsFromMapFunc{ToRequests: HandleDeploymentForDeploymentMapper(mgr.GetClient(), []predicate.Predicate{})})
	if err != nil {
		return err
	}
	// Watching for Deployment changes

	err = c.Watch(&source.Kind{Type: &openv1.DeploymentConfig{}},
		&handler.EnqueueRequestsFromMapFunc{ToRequests: HandleDeploymentConfigForDeploymentMapper(mgr.GetClient(), []predicate.Predicate{})})
	if err != nil {
		//return err
	}

	// Watching for Deployment changes
	err = c.Watch(&source.Kind{Type: &corev1.Service{}},
		&handler.EnqueueRequestsFromMapFunc{ToRequests: HandleServicesForServiceMapper(mgr.GetClient(), []predicate.Predicate{})})
	if err != nil {
		return err
	}

	// Watching for Namespace changes
	err = c.Watch(&source.Kind{Type: &corev1.Namespace{}},
		&handler.EnqueueRequestsFromMapFunc{ToRequests: HandleNameSpaceForNameSpaceMapper(mgr.GetClient(), []predicate.Predicate{})})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileSnatLocalInfo implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileSnatLocalInfo{}

// ReconcileSnatLocalInfo reconciles a SnatLocalInfo object
type ReconcileSnatLocalInfo struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileSnatLocalInfo) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling SnatLocalInfo")

	// If pod belongs to any resource in the snatPolicy
	switch {
	case strings.HasPrefix(request.Name, "snat-policyforpod"):
		result, err := r.handlePodEvent(request)
		return result, err
	case strings.HasPrefix(request.Name, "snat-policy"):
		result, err := r.handleSnatPolicyEvent(request)
		return result, err
	case strings.HasPrefix(request.Name, "snat-deploymentconfig"):
		result, err := r.handleDeploymentConfigEvent(request)
		return result, err
	case strings.HasPrefix(request.Name, "snat-deployment"):
		result, err := r.handleDeploymentEvent(request)
		return result, err
	case strings.HasPrefix(request.Name, "snat-namespace"):
		result, err := r.handleNameSpaceEvent(request)
		return result, err
	case strings.HasPrefix(request.Name, "snat-service"):
		result, err := r.handleServiceEvent(request)
		return result, err
	}
	// Fetch the SnatLocalInfo instance
	instance := &aciv1.SnatLocalInfo{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}
	return reconcile.Result{}, nil
}

// This function handles Pod events which are triggering snatLocalinfo's reconcile loop
func (r *ReconcileSnatLocalInfo) handlePodEvent(request reconcile.Request) (reconcile.Result, error) {
	// Podname: name of the pod for which loop was triggered
	// PolicyName: name of the snatPolicy for respective pod
	podName, snatPolicyName, resType, snatIp := GetPodNameFromReoncileRequest(request.Name)
	// This is to match the Service
	// Query this pod using k8s client
	foundPod := &corev1.Pod{}
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: podName, Namespace: request.Namespace}, foundPod)
	if err != nil && errors.IsNotFound(err) {
		log.V(1).Info("Deleted", "Pod: ", request.Name)
		return reconcile.Result{}, nil
	} else if err != nil {
		return reconcile.Result{}, err
	}
	snatPolicy, err := utils.GetSnatPolicyCR(r.client, snatPolicyName)
	if err != nil {
		log.Error(err, "not matching snatpolicy")
		return reconcile.Result{}, err
	}
	if snatPolicy.GetDeletionTimestamp() != nil {
		return reconcile.Result{}, nil
	}

	localInfo, err := utils.GetLocalInfoCR(r.client, foundPod.Spec.NodeName, os.Getenv("ACI_SNAT_NAMESPACE"))
	if err != nil {
		log.Error(err, "localInfo error")
	}
	if foundPod.GetObjectMeta().GetDeletionTimestamp() != nil {
		if _, ok := localInfo.Spec.LocalInfos[string(foundPod.ObjectMeta.UID)]; ok {
			nodeName := foundPod.Spec.NodeName
			snatIp := localInfo.Spec.LocalInfos[string(foundPod.ObjectMeta.UID)].SnatIp
			delete(localInfo.Spec.LocalInfos, string(foundPod.ObjectMeta.UID))
			log.V(1).Info("Delete being processed for", "Pod: ", request.Name)
			if len(localInfo.Spec.LocalInfos) == 0 {
				log.Info("SnatLocalInfo is deleted: ", "Update Snat Policy Status: ", nodeName)
				if utils.UpdateSnatPolicyStatus(nodeName, &snatPolicy, snatIp, r.client) {
					if snatPolicy.Status.State == aciv1.IpPortsExhausted {
						snatPolicy.Status.State = aciv1.Ready
					}
					err = r.client.Status().Update(context.TODO(), &snatPolicy)
					if err != nil {
						log.Error(err, "Policy status update failed")
						return reconcile.Result{}, err

					}

				}
			}
			return utils.UpdateLocalInfoCR(r.client, localInfo)
		}
		return reconcile.Result{}, nil
	} else if _, ok := localInfo.Spec.LocalInfos[string(foundPod.ObjectMeta.UID)]; ok {
		if resType == POD || localInfo.Spec.LocalInfos[string(foundPod.ObjectMeta.UID)].SnatScope == POD {
			if utils.MatchLabels(snatPolicy.Spec.Selector.Labels, foundPod.ObjectMeta.Labels) == false {
				log.Info("Label delete being processed for", "Pod: ", request.Name)
				nodeName := foundPod.Spec.NodeName
				snatIp := localInfo.Spec.LocalInfos[string(foundPod.ObjectMeta.UID)].SnatIp
				delete(localInfo.Spec.LocalInfos, string(foundPod.ObjectMeta.UID))
				refpolicy := false
				// check for  refrences for this any policy in localinfo
				for _, local := range localInfo.Spec.LocalInfos {
					if local.SnatPolicyName == snatPolicyName {
						refpolicy = true
					}
				}
				if !refpolicy {
					log.Info("SnatLocalInfo is deleted: ", "Update snat policy status:", nodeName)
					if snatPolicy.Status.State == aciv1.IpPortsExhausted {
						snatPolicy.Status.State = aciv1.Ready
					}
					if utils.UpdateSnatPolicyStatus(nodeName, &snatPolicy, snatIp, r.client) {
						err = r.client.Status().Update(context.TODO(), &snatPolicy)
						if err != nil {
							log.Error(err, "Policy status update failed")
							return reconcile.Result{}, err

						}

					}
				}
				return utils.UpdateLocalInfoCR(r.client, localInfo)
			}
		}
	}
	if foundPod.Status.Phase == "Running" {
		portinuse := make(map[string][]aciv1.NodePortRange)
		if len(snatPolicy.Status.SnatPortsAllocated) != 0 {
			portinuse = snatPolicy.Status.SnatPortsAllocated
		}
		portrange := aciv1.PortRange{}
		exists := false
		if snatIp == "" {
			snatIp, portrange, exists =
				utils.GetIPPortRangeForPod(r.client, foundPod.Spec.NodeName, &snatPolicy)
		} else {
			portrange, exists =
				utils.GetPortRangeForServiceIP(r.client, foundPod.Spec.NodeName, &snatPolicy, snatIp)
		}
		if exists == false && !reflect.DeepEqual(portrange, aciv1.PortRange{}) {
			var nodePortRnage aciv1.NodePortRange
			nodePortRnage.NodeName = foundPod.Spec.NodeName
			nodePortRnage.PortRange = portrange
			portinuse[snatIp] = append(portinuse[snatIp], nodePortRnage)
			snatPolicy.Status.SnatPortsAllocated = portinuse
			log.V(1).Info("Port range updated", "List of ports inuse: ", portinuse)
			err = r.client.Status().Update(context.TODO(), &snatPolicy)
			if err != nil {
				return reconcile.Result{}, err
			}
		}
		_, updated, err1 := r.addLocalInfo(&localInfo, foundPod, &snatPolicy, resType, snatIp)
		if err1 != nil {
			log.Error(err, "Adding localInfo error")
			return reconcile.Result{}, err
		}
		if updated {
			log.V(1).Info("SnatLocalInfo update for", "Pod: ", foundPod.ObjectMeta.Name)
			return utils.UpdateLocalInfoCR(r.client, localInfo)
		}
	}
	return reconcile.Result{}, nil
}

func (r *ReconcileSnatLocalInfo) handleSnatPolicyEvent(request reconcile.Request) (reconcile.Result, error) {
	log.Info("Handle update event for", "SnatPolicy", request)
	status, snatPolicyName, PolicyString, _ := GetPodNameFromReoncileRequest(request.Name)
	var snatPolicy aciv1.SnatPolicy
	var err error
	var isSnatPolicyDeleted bool
	// SnatPolicy is updated so delete the old Object
	if status == "deleted" {
		err = json.Unmarshal([]byte(PolicyString), &snatPolicy)
		if err != nil {
			log.Error(err, "Marshling string failed")
			return reconcile.Result{}, nil
		}
		isSnatPolicyDeleted = true
	} else {
		snatPolicy, err = utils.GetSnatPolicyCR(r.client, snatPolicyName)
		if err != nil && errors.IsNotFound(err) {
			log.Error(err, "not matching snatpolicy")
			return reconcile.Result{}, nil
		} else if err != nil {
			return reconcile.Result{}, err
		}
		isSnatPolicyDeleted = snatPolicy.GetDeletionTimestamp() != nil
	}
	if !isSnatPolicyDeleted {
		if snatPolicy.Status.State != aciv1.Ready {
			return reconcile.Result{}, nil
		}
		log.Info("Handle creation for", "SnatPolicyName: ", snatPolicyName)
	}
	if len(snatPolicy.Spec.SnatIp) == 0 {
		// This case will arise in case of Services
		return r.handleSnatPolicyForServices(&snatPolicy, status)
	}
	// This Policy will be applied at cluster level
	if reflect.DeepEqual(snatPolicy.Spec.Selector, aciv1.PodSelector{}) {
		return r.handleSnatPolicyForCluster(&snatPolicy, status)
	}

	// This case will hit if there is no labels specified and applied for Namespace
	if len(snatPolicy.Spec.Selector.Labels) == 0 {
		return r.handleSnatPolicyForNameSpace(&snatPolicy, status)
	}

	snatIps := utils.ExpandCIDRs(snatPolicy.Spec.SnatIp)
	if len(snatIps) == 0 {
		return reconcile.Result{}, nil
	}
	// Check before applying the order  pod > dep > ns
	ls := snatPolicy.Spec.Selector.Labels
	// list all Pods belong to the label
	selector := labels.SelectorFromSet(labels.Set(ls))
	existingPods := &corev1.PodList{}
	err = r.client.List(context.TODO(),
		&client.ListOptions{
			Namespace:     snatPolicy.Spec.Selector.Namespace,
			LabelSelector: selector,
		},
		existingPods)
	if err != nil {
		log.Error(err, "Failed to list pods", "Namespace: ", snatPolicy.Spec.Selector.Namespace)
		return reconcile.Result{}, err
	}
	//  list all the deployments
	deploymentList := &appsv1.DeploymentList{}
	err = r.client.List(context.TODO(),
		&client.ListOptions{
			Namespace:     snatPolicy.Spec.Selector.Namespace,
			LabelSelector: selector,
		},
		deploymentList)
	if err != nil {
		log.Error(err, "Failed to list deployments for", "Namespace: ", snatPolicy.Spec.Selector.Namespace)
	}
	depPods := []corev1.PodList{}
	for _, dep := range deploymentList.Items {
		log.V(1).Info("Matching lables for", "DeploymentName: ", dep.ObjectMeta.Name)
		Pods := &corev1.PodList{}
		r.client.List(context.TODO(),
			&client.ListOptions{
				Namespace:     dep.ObjectMeta.Namespace,
				LabelSelector: labels.SelectorFromSet(dep.Spec.Selector.MatchLabels),
			},
			Pods)
		if len(Pods.Items) != 0 {
			depPods = append(depPods, *Pods)
		}

	}
	nameSpaceList := &corev1.NamespaceList{}
	err = r.client.List(context.TODO(),
		&client.ListOptions{
			LabelSelector: selector,
		},
		nameSpaceList)
	if err != nil {
		return reconcile.Result{}, err
	}
	nameSpacePods := []corev1.PodList{}
	for _, name := range nameSpaceList.Items {
		if snatPolicy.Spec.Selector.Namespace != "" &&
			snatPolicy.Spec.Selector.Namespace != name.ObjectMeta.Name {
			continue
		}
		Pods := &corev1.PodList{}
		log.V(1).Info("Matching lables for", "NamespaceName: ", name.ObjectMeta.Name)
		r.client.List(context.TODO(),
			&client.ListOptions{
				Namespace: name.ObjectMeta.Name,
			},
			Pods)
		if len(Pods.Items) != 0 {
			nameSpacePods = append(nameSpacePods, *Pods)
		}
	}
	portinuse := make(map[string][]aciv1.NodePortRange)
	if !isSnatPolicyDeleted {
		if len(snatPolicy.Status.SnatPortsAllocated) != 0 {
			portinuse = snatPolicy.Status.SnatPortsAllocated
		} else {
			snatPolicy.Status.SnatPortsAllocated = portinuse
		}
		updated := false
		// Allocate the IP Port ranges for Pods listed
		updated = utils.AllocateIpPortRange(r.client, portinuse, existingPods, &snatPolicy)
		// Allocate the IP Port ranges for deployment listed
		for _, podlist := range depPods {
			if utils.AllocateIpPortRange(r.client, portinuse, &podlist, &snatPolicy) {
				updated = true
			}
		}
		// Allocate the IP Port ranges for namespace listed
		for _, podlist := range nameSpacePods {
			if utils.AllocateIpPortRange(r.client, portinuse, &podlist, &snatPolicy) {
				updated = true
			}
		}
		if len(portinuse) == 0 {
			return reconcile.Result{}, nil
		}
		if updated {
			snatPolicy.Status.SnatPortsAllocated = portinuse
			log.V(1).Info("Port range updated", "List of ports inuse: ", portinuse)
			err = r.client.Status().Update(context.TODO(), &snatPolicy)
			if err != nil {
				return reconcile.Result{}, err
			}
		}
	}
	if len(existingPods.Items) != 0 {
		_, err = r.snatPolicyUpdate(existingPods, &snatPolicy, POD, isSnatPolicyDeleted, "", false)
		if err != nil {
			return reconcile.Result{}, err
		}
	}

	for _, podlist := range depPods {
		_, err = r.snatPolicyUpdate(&podlist, &snatPolicy, DEPLOYMENT, isSnatPolicyDeleted, "", false)
		if err != nil {
			return reconcile.Result{}, err
		}
	}

	for _, podlist := range nameSpacePods {
		_, err = r.snatPolicyUpdate(&podlist, &snatPolicy, NAMESPACE, isSnatPolicyDeleted, "", false)
		if err != nil {
			return reconcile.Result{}, err
		}
	}
	return reconcile.Result{}, nil
}

func (r *ReconcileSnatLocalInfo) snatPolicyUpdate(existingPods *corev1.PodList,
	snatpolicy *aciv1.SnatPolicy, resType string, deleted bool, snatip string, forcedel bool) (reconcile.Result, error) {
	// caching the localinfo and snaip per node to update once
	localInfos := make(map[string]aciv1.SnatLocalInfo)
	snatIps := make(map[string]string)
	var localInfo aciv1.SnatLocalInfo
	poddeleted := false
	snatPolicyList := aciv1.SnatPolicyList{}
	var fristPod corev1.Pod
	if len(existingPods.Items) == 0 {
		return reconcile.Result{}, nil
	}
	err := r.client.List(context.TODO(), &client.ListOptions{Namespace: ""}, &snatPolicyList)
	if err != nil {
		log.Info("Failed to list snatpolicy")
	}
	for _, pod := range existingPods.Items {
		//Don't add snatip for host network pods
		if resType == "cluster" && pod.Spec.HostNetwork {
			continue
		}
		// This is required to sample first pod  in case if we need to enqueue for other policy
		fristPod = pod
		if _, ok := localInfos[pod.Spec.NodeName]; !ok {
			localInfo, err = utils.GetLocalInfoCR(r.client, pod.Spec.NodeName, os.Getenv("ACI_SNAT_NAMESPACE"))
			if err != nil {
				log.Error(err, "localInfo error")
				return reconcile.Result{}, err
			}
		} else {
			// This case will fall for delete case
			localInfo = localInfos[pod.Spec.NodeName]
		}
		nodeName := pod.Spec.NodeName
		snatIp := localInfo.Spec.LocalInfos[string(pod.ObjectMeta.UID)].SnatIp
		if pod.GetObjectMeta().GetDeletionTimestamp() != nil {
			if _, ok := localInfo.Spec.LocalInfos[string(pod.ObjectMeta.UID)]; ok {
				poddeleted = true
				delete(localInfo.Spec.LocalInfos, string(pod.ObjectMeta.UID))
				localInfos[nodeName] = localInfo
				if len(localInfo.Spec.LocalInfos) == 0 {
					snatIps[nodeName] = snatIp
				}
			}
		} else if deleted {
			log.V(1).Info("Deleted/label removed for", "SnatPolicyName: ", snatpolicy.ObjectMeta.Name)
			if _, ok := localInfo.Spec.LocalInfos[string(pod.ObjectMeta.UID)]; ok {
				inforesType := localInfo.Spec.LocalInfos[string(pod.ObjectMeta.UID)].SnatScope
				poddeleted = forcedel
				// forcedel will be set if any target gets deleted
				if !forcedel {
					if inforesType == POD && (resType != POD) {
						continue
					} else if inforesType == SERVICE &&
						(resType == NAMESPACE || resType == CLUSTER || resType == DEPLOYMENT) {
						continue
					} else if inforesType == DEPLOYMENT && (resType == NAMESPACE || resType == CLUSTER) {
						continue
					} else if inforesType == NAMESPACE && resType == CLUSTER {
						continue
					}
				}
				delete(localInfo.Spec.LocalInfos, string(pod.ObjectMeta.UID))
				// This case is possible when some of the POD's are deleted marked in case of Deployment
				localInfos[nodeName] = localInfo
				if snatpolicy.GetDeletionTimestamp() == nil {
					refpolicy := false
					for _, local := range localInfo.Spec.LocalInfos {
						if local.SnatPolicyName == snatpolicy.ObjectMeta.Name {
							refpolicy = true
						}
					}
					// if no reference to the policy found remove the snatIP
					if !refpolicy {
						log.V(1).Info("Snatpolicy deleted update the status for", "NodeName: ", nodeName)
						snatIps[nodeName] = snatIp
					}
				}
			}
		} else if pod.Status.Phase == corev1.PodRunning {
			_, updated, err1 := r.addLocalInfo(&localInfo, &pod, snatpolicy, resType, snatip)
			if err1 != nil {
				log.Error(err, "Adding localInfo error")
				return reconcile.Result{}, err
			}
			if updated {
				localInfos[nodeName] = localInfo
			}
		}
	}
	for _, localinfo := range localInfos {
		_, err := utils.UpdateLocalInfoCR(r.client, localinfo)
		if err != nil {
			log.Error(err, "updating localInfo error")
			return reconcile.Result{}, err
		}
	}

	if deleted {
		// in the update Old Object's policy status needs to be cleared
		var curpolicy aciv1.SnatPolicy
		curpolicy, err = utils.GetSnatPolicyCR(r.client, snatpolicy.ObjectMeta.Name)
		update := false
		// update the status to be nil for Policy delete and update case
		if curpolicy.GetDeletionTimestamp() != nil || !reflect.DeepEqual(curpolicy.Spec, snatpolicy.Spec) {
			log.V(1).Info("Update the status for", "SnatPolicyName: ", snatpolicy.ObjectMeta.Name)
			curpolicy.Status.SnatPortsAllocated = nil
			update = true
		} else {
			// this case will be for deployment and ns delete
			for nodeName, snatIp := range snatIps {
				if utils.UpdateSnatPolicyStatus(nodeName, &curpolicy, snatIp, r.client) {
					update = true
				}
			}
		}
		if update {
			log.V(1).Info("Update status", "PortsAllocated: ", curpolicy.Status.SnatPortsAllocated)
			curpolicy.Status.State = aciv1.Ready
			err = r.client.Status().Update(context.TODO(), &curpolicy)
			if err != nil {
				log.Error(err, "SnatPolicy status update failed")
				return reconcile.Result{}, err
			}
		}
	}

	request := FilterPodsPerSnatPolicy(r.client, &snatPolicyList, &fristPod)
	if len(request) != 0 && !poddeleted && deleted {
		log.Info("Enque the request again: ", "Request: ", request)
		r.handleSnatPolicyEvent(request[0])
	}
	return reconcile.Result{}, nil
}
func (r *ReconcileSnatLocalInfo) addLocalInfo(snatlocalinfo *aciv1.SnatLocalInfo, pod *corev1.Pod,
	snatpolicy *aciv1.SnatPolicy, resType string, snatIp string) (reconcile.Result, bool, error) {
	policyname := snatpolicy.GetObjectMeta().GetName()
	log.V(1).Info("Add snatlocalinfo for", "SnatPolicy name: ", policyname)
	var portrange aciv1.PortRange
	var snatip string
	var err error
	if len(snatpolicy.Spec.SnatIp) == 0 {
		snatip = snatIp
		portrange, _ = utils.GetPortRangeForServiceIP(r.client, pod.Spec.NodeName, snatpolicy, snatIp)
	} else {
		snatip, portrange, _ = utils.GetIPPortRangeForPod(r.client, pod.Spec.NodeName, snatpolicy)
	}
	if reflect.DeepEqual(portrange, aciv1.PortRange{}) {
		log.Info("IPPorts exhausted for", "SnatPolicy name: ", policyname)
		if snatpolicy.Status.State == aciv1.Ready {
			snatpolicy.Status.State = aciv1.IpPortsExhausted
			err = r.client.Status().Update(context.TODO(), snatpolicy)
			if err != nil {
				return reconcile.Result{}, false, err
			}
			return reconcile.Result{}, false, nil
		}
	}
	tempLocalInfo := aciv1.LocalInfo{
		PodName:        pod.GetObjectMeta().GetName(),
		PodNamespace:   pod.GetObjectMeta().GetNamespace(),
		SnatIp:         snatip,
		SnatPolicyName: snatpolicy.ObjectMeta.Name,
		SnatScope:      resType,
	}
	if len(snatlocalinfo.Spec.LocalInfos) == 0 && snatlocalinfo.GetObjectMeta().GetName() != pod.Spec.NodeName {
		log.Info("SnatLocalInfo CR is not present", "Creating new one: ", pod.Spec.NodeName)
		tempMap := make(map[string]aciv1.LocalInfo)
		tempMap[string(pod.ObjectMeta.UID)] = tempLocalInfo
		tempLocalInfoSpec := aciv1.SnatLocalInfoSpec{
			LocalInfos: tempMap,
		}
		snatlocalinfo, _, err = utils.CreateLocalInfoCR(r.client, tempLocalInfoSpec, pod.Spec.NodeName)
		if err != nil {
			log.Error(err, "Create localInfo Failed")
			return reconcile.Result{}, false, err
		}
	} else {
		// LocaInfo CR is already present, Append localInfo object into Spec's map  and update Locainfo
		scopechange := false
		if len(snatlocalinfo.Spec.LocalInfos) == 0 {
			tempMap := make(map[string]aciv1.LocalInfo)
			tempMap[string(pod.ObjectMeta.UID)] = tempLocalInfo
			tempLocalInfoSpec := aciv1.SnatLocalInfoSpec{
				LocalInfos: tempMap,
			}
			snatlocalinfo.Spec = tempLocalInfoSpec
			scopechange = true
		} else {
			log.V(1).Info("SnatLocalInfo is changed for", "Pod: ", pod.ObjectMeta.Name)
			if info, ok := snatlocalinfo.Spec.LocalInfos[string(pod.ObjectMeta.UID)]; ok {
				if info.SnatScope == DEPLOYMENT && (resType == POD || resType == SERVICE) {
					scopechange = true
					tempLocalInfo.SnatScope = resType
				} else if info.SnatScope == NAMESPACE && (resType == POD || resType == SERVICE || resType == DEPLOYMENT) {
					scopechange = true
					tempLocalInfo.SnatScope = resType
				} else if info.SnatScope == SERVICE && resType == POD {
					scopechange = true
					tempLocalInfo.SnatScope = resType
				} else if info.SnatScope == CLUSTER && resType != CLUSTER {
					scopechange = true
					tempLocalInfo.SnatScope = resType
				}

			} else {
				scopechange = true
			}
		}
		if scopechange {
			snatlocalinfo.Spec.LocalInfos[string(pod.ObjectMeta.UID)] = tempLocalInfo
			log.Info("SnatLocalInfo is updated: ", "SnatLocalInfos: ", snatlocalinfo.Spec.LocalInfos)
		}
		return reconcile.Result{}, scopechange, nil
	}
	return reconcile.Result{}, false, nil
}

func (r *ReconcileSnatLocalInfo) handleDeploymentEvent(request reconcile.Request) (reconcile.Result, error) {
	// revisit this code t write separate API for GetPodNameFromReoncileRequest to fix the names
	depName, namespace, slectorstring, _ := GetPodNameFromReoncileRequest(request.Name)
	dep := &appsv1.Deployment{}
	matches := false
	var snatPolicyName string
	deploymentdeleted := false
	selector := make(map[string]string)
	err := r.client.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: depName}, dep)
	if err != nil && errors.IsNotFound(err) {
		log.Info("HandleDeploymentEvent deployment not found: ", "DeploymentName: ", depName)
		deploymentdeleted = true
	} else if err != nil {
		return reconcile.Result{}, err
	}
	// Deployment delete cleanup handled as Pod Delete
	if dep.GetObjectMeta().GetDeletionTimestamp() != nil {
		log.Info("HandleDeploymentEvent as deployment is being deleted: ", "DeploymentName: ", depName)
		deploymentdeleted = true
	}
	err = json.Unmarshal([]byte(slectorstring), &selector)
	if err != nil {
		log.Error(err, "Marshling string Failed")
		return reconcile.Result{}, nil
	}
	if !deploymentdeleted {
		snatPolicyName, matches = utils.CheckMatchesLabletoPolicy(r.client, selector, namespace)
		if matches {
			if !reflect.DeepEqual(selector, dep.ObjectMeta.Labels) {
				return reconcile.Result{}, nil
			}
			log.Info("handleDeploymentEvent", "Matches labels: ", dep.ObjectMeta.Labels)
		}
		selector = dep.Spec.Selector.MatchLabels
	}
	Pods := &corev1.PodList{}
	err = r.client.List(context.TODO(),
		&client.ListOptions{
			Namespace:     namespace,
			LabelSelector: labels.SelectorFromSet(selector),
		},
		Pods)
	if err != nil {
		log.Error(err, "Failed to list pods for", "Namespace: ", namespace)
		return reconcile.Result{}, err
	}
	_, err = r.updatePods(Pods, matches, snatPolicyName, DEPLOYMENT, deploymentdeleted)

	if err != nil {
		return reconcile.Result{}, err
	}
	return reconcile.Result{}, nil
}

func (r *ReconcileSnatLocalInfo) handleNameSpaceEvent(request reconcile.Request) (reconcile.Result, error) {
	namespaceName, _, slectorstring, _ := GetPodNameFromReoncileRequest(request.Name)
	namespace := &corev1.Namespace{}
	matches := false
	var snatPolicyName string
	namespacedeleted := false
	selector := make(map[string]string)
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: namespaceName}, namespace)
	if err != nil && errors.IsNotFound(err) {
		log.Info("HandleNameSpaceEvent namespace not found", "NameSpaceName: ", namespaceName)
		namespacedeleted = true
	} else if err != nil {
		return reconcile.Result{}, err
	}
	// namespace delete cleanup handled as Pod Delete
	if namespace.GetDeletionTimestamp() != nil {
		log.Info("HandleNameSpaceEvent namespace being deleted: ", "NameSpaceName: ", namespace)
		namespacedeleted = true
	}
	if !namespacedeleted {
		err = json.Unmarshal([]byte(slectorstring), &selector)
		if err != nil {
			log.Error(err, "Marshling string Failed")
			return reconcile.Result{}, nil
		}
		snatPolicyName, matches = utils.CheckMatchesLabletoPolicy(r.client, selector, namespaceName)
		if matches {
			if !reflect.DeepEqual(selector, namespace.ObjectMeta.Labels) {
				return reconcile.Result{}, nil
			}
			log.Info("HandleNameSpaceEvent", "Matches labels: ", namespace.ObjectMeta.Labels)
		}
	}
	Pods := &corev1.PodList{}
	err = r.client.List(context.TODO(),
		&client.ListOptions{
			Namespace: namespaceName,
		},
		Pods)
	if err != nil {
		log.Error(err, "Failed to list pods for", "Namespace: ", namespaceName)
		return reconcile.Result{}, err
	}
	_, err = r.updatePods(Pods, matches, snatPolicyName, NAMESPACE, namespacedeleted)
	if err != nil {
		return reconcile.Result{}, err
	}
	return reconcile.Result{}, nil
}

func (r *ReconcileSnatLocalInfo) handleDeploymentConfigEvent(request reconcile.Request) (reconcile.Result, error) {
	// revisit this code t write separate API for GetPodNameFromReoncileRequest to fix the names
	depName, namespace, slectorstring, _ := GetPodNameFromReoncileRequest(request.Name)
	dep := &openv1.DeploymentConfig{}
	var snatPolicyName string
	deploymentdeleted := false
	selector := make(map[string]string)
	matches := false
	err := r.client.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: depName}, dep)
	if err != nil && errors.IsNotFound(err) {
		log.Info("HandleDeploymentEvent deploymentconf not found: ", "DeploymentconfName: ", depName)
		deploymentdeleted = true
	} else if err != nil {
		return reconcile.Result{}, nil
	}
	// Deployment delete cleanup handled as Pod Delete
	if dep.GetDeletionTimestamp() != nil {
		log.Info("HandleDeploymentEvent as deploymentconf is being deleted: ", "DeploymentconfName: ", depName)
		deploymentdeleted = true
	}
	err = json.Unmarshal([]byte(slectorstring), &selector)
	if err != nil {
		log.Error(err, "Marshling string Failed")
		return reconcile.Result{}, nil
	}
	if !deploymentdeleted {
		snatPolicyName, matches = utils.CheckMatchesLabletoPolicy(r.client, selector, namespace)
		if matches {
			if !reflect.DeepEqual(selector, dep.ObjectMeta.Labels) {
				return reconcile.Result{}, nil
			}
			log.Info("HandleDeploymentEvent", "Matches policy: ", snatPolicyName)
		}
		selector = dep.Spec.Selector
	}
	Pods := &corev1.PodList{}
	r.client.List(context.TODO(),
		&client.ListOptions{
			Namespace:     dep.ObjectMeta.Namespace,
			LabelSelector: labels.SelectorFromSet(selector),
		},
		Pods)
	_, err = r.updatePods(Pods, matches, snatPolicyName, DEPLOYMENT, deploymentdeleted)
	if err != nil {
		return reconcile.Result{}, err
	}
	return reconcile.Result{}, nil
}

func (r *ReconcileSnatLocalInfo) updatePods(Pods *corev1.PodList, matches bool, snatPolicyName string,
	resType string, resdeleted bool) (reconcile.Result, error) {

	if len(Pods.Items) == 0 {
		return reconcile.Result{}, nil
	}
	if matches {
		snatPolicy, err := utils.GetSnatPolicyCR(r.client, snatPolicyName)
		if err != nil && errors.IsNotFound(err) {
			log.Error(err, "not matching snatpolicy")
			return reconcile.Result{}, nil
		} else if err != nil {
			return reconcile.Result{}, err
		}
		if snatPolicy.GetDeletionTimestamp() != nil {
			return reconcile.Result{}, err
		}
		portinuse := make(map[string][]aciv1.NodePortRange)
		if snatPolicy.Status.SnatPortsAllocated != nil {
			portinuse = snatPolicy.Status.SnatPortsAllocated
		} else {
			snatPolicy.Status.SnatPortsAllocated = portinuse
		}
		updated := false
		updated = utils.AllocateIpPortRange(r.client, portinuse, Pods, &snatPolicy)
		if updated {
			log.V(1).Info("Port range updated", "List of ports inuse: ", portinuse)
			err = r.client.Status().Update(context.TODO(), &snatPolicy)
			if err != nil {
				return reconcile.Result{}, err
			}
		}
		if len(portinuse) == 0 {
			return reconcile.Result{}, nil
		}
		_, err = r.snatPolicyUpdate(Pods, &snatPolicy, resType, false, "", resdeleted)
		if err != nil {
			return reconcile.Result{}, err
		}
	} else {
		if len(Pods.Items) > 0 {
			pod := Pods.Items[0]
			snatPolicy, err := utils.GetSnatPolicyCRFromPod(r.client, &pod)
			if err != nil {
				return reconcile.Result{}, err
			}
			// return if it is empty policy
			if reflect.DeepEqual(snatPolicy, aciv1.SnatPolicy{}) {
				return reconcile.Result{}, nil
			}
			log.V(1).Info("Found pod", "Matches policy name: ", snatPolicy.ObjectMeta.Name)
			_, err1 := r.snatPolicyUpdate(Pods, &snatPolicy, resType, true, "", resdeleted)
			if err1 != nil {
				log.Error(err, "SnatPolicy update failed", "SnatPolicy: ", snatPolicy.ObjectMeta.Name)
				return reconcile.Result{}, err
			}
		}
	}

	return reconcile.Result{}, nil
}

func (r *ReconcileSnatLocalInfo) handleSnatPolicyForServices(snatPolicy *aciv1.SnatPolicy, status string) (reconcile.Result, error) {
	isSnatPolicyDeleted := snatPolicy.GetDeletionTimestamp() != nil
	// SnatPolicy is updated so delete the old Object
	if status == "deleted" {
		isSnatPolicyDeleted = true
	}
	var err error
	// Check before applying the order  pod > dep > ns
	ls := snatPolicy.Spec.Selector.Labels
	// list all Pods belong to the label
	var resType string
	selector := labels.SelectorFromSet(labels.Set(ls))
	SerivesList := &corev1.ServiceList{}
	if len(snatPolicy.Spec.Selector.Labels) == 0 {
		err = r.client.List(context.TODO(),
			&client.ListOptions{
				Namespace: snatPolicy.Spec.Selector.Namespace,
			},
			SerivesList)
		resType = NAMESPACE
	} else {
		err = r.client.List(context.TODO(),
			&client.ListOptions{
				Namespace:     snatPolicy.Spec.Selector.Namespace,
				LabelSelector: selector,
			},
			SerivesList)
		resType = SERVICE
	}
	if err != nil {
		return reconcile.Result{}, err
	}
	log.Info("Matching lables for ", "ServiceList: ", SerivesList)
	if len(SerivesList.Items) == 0 {
		return reconcile.Result{}, nil
	}
	portinuse := make(map[string][]aciv1.NodePortRange)
	for _, service := range SerivesList.Items {
		Pods := &corev1.PodList{}
		r.client.List(context.TODO(),
			&client.ListOptions{
				Namespace:     snatPolicy.Spec.Selector.Namespace,
				LabelSelector: labels.SelectorFromSet(service.Spec.Selector),
			},
			Pods)
		if len(service.Status.LoadBalancer.Ingress) == 0 {
			log.Info("No external loadbalance ip for", "Service: ", service)
			continue
		}
		snatip := service.Status.LoadBalancer.Ingress[0].IP
		if len(Pods.Items) != 0 {
			if !isSnatPolicyDeleted {
				if len(snatPolicy.Status.SnatPortsAllocated) != 0 {
					portinuse = snatPolicy.Status.SnatPortsAllocated
				} else {
					snatPolicy.Status.SnatPortsAllocated = portinuse
				}
				if utils.AllocateIpPortRangeforservice(r.client, portinuse, Pods, snatPolicy, snatip) {
					snatPolicy.Status.SnatPortsAllocated = portinuse
					log.V(1).Info("Port range updated", "List of ports inUse: ", portinuse)
					err = r.client.Status().Update(context.TODO(), snatPolicy)
					if err != nil {
						return reconcile.Result{}, err
					}
				}
			}
			_, err := r.snatPolicyUpdate(Pods, snatPolicy, resType, isSnatPolicyDeleted, snatip, false)
			if err != nil {
				log.Error(err, "SnatPolicy update failed", "SnatPolicy: ", snatPolicy.ObjectMeta.Name)
				return reconcile.Result{}, err
			}
		}
	}
	return reconcile.Result{}, nil
}

func (r *ReconcileSnatLocalInfo) handleServiceEvent(request reconcile.Request) (reconcile.Result, error) {
	// revisit this code t write separate API for GetPodNameFromReoncileRequest to fix the names
	servicename, namespace, slectorstring, _ := GetPodNameFromReoncileRequest(request.Name)
	log.V(1).Info("HandleServiceEvent for", "ServiceName: ", servicename)
	service := &corev1.Service{}
	matches := false
	var snatPolicyName string
	var snatip string
	var resType string
	servicedeleted := false
	selector := make(map[string]string)
	err := r.client.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: servicename}, service)
	if err != nil && errors.IsNotFound(err) {
		servicedeleted = true
		log.Info("HandleServiceEvent service is not found: ", "ServiceName: ", servicename)
	} else if err != nil {
		return reconcile.Result{}, nil
	}
	// Service deleted cleanup handled as Pod Delete
	if service.GetObjectMeta().GetDeletionTimestamp() != nil {
		log.V(1).Info("HandleServiceEvent  as service being deleted: ", "ServiceName: ", servicename)
		servicedeleted = true
	}
	snatPolicyList := &aciv1.SnatPolicyList{}
	if err := r.client.List(context.TODO(), &client.ListOptions{Namespace: ""}, snatPolicyList); err != nil {
		return reconcile.Result{}, nil
	}
	if servicedeleted {
		err = json.Unmarshal([]byte(slectorstring), &selector)
		if err != nil {
			log.Error(err, "Marshling string failed")
			return reconcile.Result{}, nil
		}
	} else {
		selector = service.Spec.Selector
		snatPolicyName, matches = utils.CheckMatchesLabletoPolicy(r.client, service.ObjectMeta.Labels, namespace)
		if matches {
			log.Info("HandleServiceEvent for", "Matching labels: ", service.ObjectMeta.Labels)
		} else {
			// This Case is possible when Service doesn't have labels only namespace based
			for _, item := range snatPolicyList.Items {
				if item.Status.State != aciv1.Ready {
					continue
				}
				if len(item.Spec.SnatIp) == 0 && len(item.Spec.Selector.Labels) == 0 &&
					item.Spec.Selector.Namespace == namespace {
					matches = true
					snatPolicyName = item.ObjectMeta.Name
					break
				}
			}
		}
	}
	Pods := &corev1.PodList{}
	err = r.client.List(context.TODO(),
		&client.ListOptions{
			Namespace:     namespace,
			LabelSelector: labels.SelectorFromSet(selector),
		},
		Pods)
	if err != nil {
		log.Error(err, "Listing pods failed")
		return reconcile.Result{}, nil
	}
	if len(Pods.Items) == 0 {
		return reconcile.Result{}, nil
	}
	if matches == true {
		if len(service.Status.LoadBalancer.Ingress) == 0 {
			log.Info("No external loadbalance ip for ", "Service: ", service)
			return reconcile.Result{}, nil
		}
		snatip = service.Status.LoadBalancer.Ingress[0].IP
		snatPolicy, err := utils.GetSnatPolicyCR(r.client, snatPolicyName)
		if err != nil && errors.IsNotFound(err) {
			log.Error(err, "not matching snatpolicy")
			return reconcile.Result{}, nil
		} else if err != nil {
			return reconcile.Result{}, err
		}
		if snatPolicy.GetDeletionTimestamp() != nil || len(snatPolicy.Spec.SnatIp) != 0 {
			return reconcile.Result{}, nil
		}
		portinuse := make(map[string][]aciv1.NodePortRange)
		if len(snatPolicy.Status.SnatPortsAllocated) != 0 {
			portinuse = snatPolicy.Status.SnatPortsAllocated
		} else {
			snatPolicy.Status.SnatPortsAllocated = portinuse
		}

		if utils.AllocateIpPortRangeforservice(r.client, portinuse, Pods, &snatPolicy, snatip) {
			snatPolicy.Status.SnatPortsAllocated = portinuse
			log.V(1).Info("Port range updated", "List of ports inuse: ", portinuse)
			err = r.client.Status().Update(context.TODO(), &snatPolicy)
			if err != nil {
				return reconcile.Result{}, err
			}
		}
		if len(portinuse) == 0 {
			return reconcile.Result{}, nil
		}
		if len(snatPolicy.Spec.Selector.Labels) != 0 {
			resType = SERVICE
		} else {
			resType = NAMESPACE
		}
		_, err = r.snatPolicyUpdate(Pods, &snatPolicy, resType, false, snatip, false)
		if err != nil {
			log.Error(err, "SnatPolicy update failed", "SnatPolicy: ", snatPolicy.ObjectMeta.Name)
			return reconcile.Result{}, err
		}

	} else {
		if len(Pods.Items) > 0 {
			// check the first pod get the policy info
			pod := Pods.Items[0]
			snatPolicy, err := utils.GetSnatPolicyCRFromPod(r.client, &pod)
			if err != nil {
				return reconcile.Result{}, err
			}
			// return if it is empty policy
			if reflect.DeepEqual(snatPolicy, aciv1.SnatPolicy{}) {
				return reconcile.Result{}, nil
			}
			if len(snatPolicy.Spec.Selector.Labels) != 0 {
				resType = SERVICE
			} else {
				resType = NAMESPACE
			}
			if servicedeleted {
				_, err = r.snatPolicyUpdate(Pods, &snatPolicy, resType, true, snatip, true)
			} else {
				_, err = r.snatPolicyUpdate(Pods, &snatPolicy, resType, true, snatip, false)
			}
			if err != nil {
				log.Error(err, "SnatPolicy update failed", "SnatPolicy: ", snatPolicy.ObjectMeta.Name)
				return reconcile.Result{}, err
			}
		}
	}
	return reconcile.Result{}, nil
}

func (r *ReconcileSnatLocalInfo) handleSnatPolicyForCluster(snatPolicy *aciv1.SnatPolicy, status string) (reconcile.Result, error) {
	isSnatPolicyDeleted := snatPolicy.GetDeletionTimestamp() != nil
	if status == "deleted" {
		isSnatPolicyDeleted = true
	}
	// list all Pods belong to the label
	snatIps := utils.ExpandCIDRs(snatPolicy.Spec.SnatIp)
	if len(snatIps) == 0 {
		return reconcile.Result{}, nil
	}
	Pods := &corev1.PodList{}
	err := r.client.List(context.TODO(),
		&client.ListOptions{
			Namespace: corev1.NamespaceAll,
		},
		Pods)
	portinuse := make(map[string][]aciv1.NodePortRange)
	if !isSnatPolicyDeleted {
		if len(snatPolicy.Status.SnatPortsAllocated) != 0 {
			portinuse = snatPolicy.Status.SnatPortsAllocated
		} else {
			snatPolicy.Status.SnatPortsAllocated = portinuse
		}
		if utils.AllocateIpPortRange(r.client, portinuse, Pods, snatPolicy) {
			snatPolicy.Status.SnatPortsAllocated = portinuse
			log.V(1).Info("Port range updated", "List of ports inuse: ", portinuse)
			err = r.client.Status().Update(context.TODO(), snatPolicy)
			if err != nil {
				return reconcile.Result{}, err
			}
		}
	}
	log.Info("SnatPolicy applied at Cluster for", "SnatPolicyName: ", snatPolicy.ObjectMeta.Name)
	_, err = r.snatPolicyUpdate(Pods, snatPolicy, CLUSTER, isSnatPolicyDeleted, "", false)
	if err != nil {
		log.Error(err, "SnatPolicy update failed", "SnatPolicy: ", snatPolicy.ObjectMeta.Name)
		return reconcile.Result{}, err
	}
	return reconcile.Result{}, nil
}

func (r *ReconcileSnatLocalInfo) handleSnatPolicyForNameSpace(snatPolicy *aciv1.SnatPolicy, status string) (reconcile.Result, error) {

	isSnatPolicyDeleted := snatPolicy.GetDeletionTimestamp() != nil
	// SnatPolicy is updated so delete the old Object
	if status == "deleted" {
		isSnatPolicyDeleted = true
	}
	nameSpacePods := &corev1.PodList{}
	err := r.client.List(context.TODO(),
		&client.ListOptions{
			Namespace: snatPolicy.Spec.Selector.Namespace,
		},
		nameSpacePods)
	if err != nil {
		log.Error(err, "Failed to list pods for", "Namespace: ", snatPolicy.Spec.Selector.Namespace)
		return reconcile.Result{}, nil
	}
	snatIps := utils.ExpandCIDRs(snatPolicy.Spec.SnatIp)
	if len(snatIps) == 0 {
		return reconcile.Result{}, nil
	}
	portinuse := make(map[string][]aciv1.NodePortRange)
	if !isSnatPolicyDeleted {
		if len(snatPolicy.Status.SnatPortsAllocated) != 0 {
			portinuse = snatPolicy.Status.SnatPortsAllocated
		} else {
			snatPolicy.Status.SnatPortsAllocated = portinuse
		}
		if utils.AllocateIpPortRange(r.client, portinuse, nameSpacePods, snatPolicy) {
			snatPolicy.Status.SnatPortsAllocated = portinuse
			log.V(1).Info("Port range updated", "List of ports inuse: ", portinuse)
			err = r.client.Status().Update(context.TODO(), snatPolicy)
			if err != nil {
				return reconcile.Result{}, err
			}
		}
	}
	log.Info("SnatPolicy applied at namespace for", "SnatPolicyName: ", snatPolicy.ObjectMeta.Name)
	if len(nameSpacePods.Items) != 0 {
		_, err = r.snatPolicyUpdate(nameSpacePods, snatPolicy, NAMESPACE, isSnatPolicyDeleted, "", false)
		if err != nil {
			log.Error(err, "SnatPolicy update failed", "SnatPolicy: ", snatPolicy.ObjectMeta.Name)
			return reconcile.Result{}, err
		}
	}
	return reconcile.Result{}, nil
}
