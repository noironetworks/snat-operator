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
	case strings.HasPrefix(request.Name, "snat-policyforservicepod"):
		result, err := r.handlePodForServiceEvent(request)
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
	podName, snatPolicyName, resType := utils.GetPodNameFromReoncileRequest(request.Name)

	// Query this pod using k8s client
	foundPod := &corev1.Pod{}
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: podName, Namespace: request.Namespace}, foundPod)

	if err != nil && errors.IsNotFound(err) {
		log.Info("Pod deleted", "PodName:", request.Name)
		return reconcile.Result{}, nil
	} else if err != nil {
		return reconcile.Result{}, err
	}
	log.Info("POD found:", "Pod name", foundPod.ObjectMeta.Name)
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
		log.Info("Local Info to be deleted: ", "Pod UUID", string(foundPod.ObjectMeta.Name))
		if _, ok := localInfo.Spec.LocalInfos[string(foundPod.ObjectMeta.UID)]; ok {
			nodeName := foundPod.Spec.NodeName
			snatIp := localInfo.Spec.LocalInfos[string(foundPod.ObjectMeta.UID)].SnatIp
			delete(localInfo.Spec.LocalInfos, string(foundPod.ObjectMeta.UID))
			log.Info("Local Info to be deleted: ", "Update Snat Policy Status###", len(localInfo.Spec.LocalInfos))
			if len(localInfo.Spec.LocalInfos) == 0 {
				log.Info("Local Info to be deleted: ", "Update Snat Policy Status###", nodeName)
				_, err = utils.UpdateSnatPolicyStatus(nodeName, snatPolicyName, snatIp, r.client)
				if err != nil {
					log.Error(err, "Policy Status Update Failed")
					return reconcile.Result{}, err
				}
			}
			return utils.UpdateLocalInfoCR(r.client, localInfo)
		}
		return reconcile.Result{}, nil
	}
	// Check for the case where label might have been removed
	if _, ok := localInfo.Spec.LocalInfos[string(foundPod.ObjectMeta.UID)]; ok {
		if resType == "pod" || localInfo.Spec.LocalInfos[string(foundPod.ObjectMeta.UID)].SnatScope == "pod" {
			if utils.MatchLabels(snatPolicy.Spec.Selector.Labels, foundPod.ObjectMeta.Labels) == false {
				nodeName := foundPod.Spec.NodeName
				snatIp := localInfo.Spec.LocalInfos[string(foundPod.ObjectMeta.UID)].SnatIp
				delete(localInfo.Spec.LocalInfos, string(foundPod.ObjectMeta.UID))
				if len(localInfo.Spec.LocalInfos) == 0 {
					_, err = utils.UpdateSnatPolicyStatus(nodeName, snatPolicyName, snatIp, r.client)
					if err != nil {
						log.Error(err, "Policy Status Update Failed")
						return reconcile.Result{}, err
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
		snatip, portrange, exists := utils.GetIPPortRangeForPod(foundPod.Spec.NodeName, &snatPolicy)
		if exists == false {
			var nodePortRnage aciv1.NodePortRange
			nodePortRnage.NodeName = foundPod.Spec.NodeName
			nodePortRnage.PortRange = portrange
			portinuse[snatip] = append(portinuse[snatip], nodePortRnage)
			snatPolicy.Status.SnatPortsAllocated = portinuse
			log.Info("Port Range Updated: ", "List of Ports InUse: ", portinuse)
			err = r.client.Status().Update(context.TODO(), &snatPolicy)
			if err != nil {
				return reconcile.Result{}, err
			}
		}
		_, err = r.addLocalInfo(&localInfo, foundPod, &snatPolicy, resType, "")
		if err != nil {
			log.Error(err, "Adding localInfo error")
			return reconcile.Result{}, err
		}
		return utils.UpdateLocalInfoCR(r.client, localInfo)
	}
	return reconcile.Result{}, nil
}
func (r *ReconcileSnatLocalInfo) handleSnatPolicyEvent(request reconcile.Request) (reconcile.Result, error) {
	log.Info("Snat Policy Update Event ", "Snat Policy: ", request)
	status, snatPolicyName, PolicyString := utils.GetPodNameFromReoncileRequest(request.Name)
	var snatPolicy aciv1.SnatPolicy
	var err error
	var isSnatPolicyDeleted bool
	// SnatPolicy is updated so delete the old Object
	if status == "deleted" {
		err = json.Unmarshal([]byte(PolicyString), &snatPolicy)
		if err != nil {
			log.Error(err, "Marshling string Failed")
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
	if snatPolicy.Status.State != aciv1.Ready {
		return reconcile.Result{}, nil
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
	ls := make(map[string]string)

	for _, label := range snatPolicy.Spec.Selector.Labels {
		ls[label.Key] = label.Value
	}

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
		log.Info("Failed to list Pods: ", "Namespace: ", snatPolicy.Spec.Selector.Namespace)
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
		log.Info("Failed to list deployments: ", "Namespace: ", snatPolicy.Spec.Selector.Namespace)
		//return reconcile.Result{}, err
	}
	depPods := []corev1.PodList{}
	for _, dep := range deploymentList.Items {
		log.Info("Matching lables for", "Depolymet: ", dep)
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
	nameSpacePods := &corev1.PodList{}
	nameSpaceList := &corev1.NamespaceList{}
	err = r.client.List(context.TODO(),
		&client.ListOptions{
			LabelSelector: selector,
		},
		nameSpaceList)
	if err != nil {
		return reconcile.Result{}, err
	}
	mameSpacePods := []corev1.PodList{}
	for _, name := range nameSpaceList.Items {
		Pods := &corev1.PodList{}
		log.Info("Matching lables for", "namespace: ", name.ObjectMeta.Name)
		r.client.List(context.TODO(),
			&client.ListOptions{
				Namespace: name.ObjectMeta.Name,
			},
			Pods)
		if len(Pods.Items) != 0 {
			mameSpacePods = append(mameSpacePods, *Pods)
		}
	}
	portinuse := make(map[string][]aciv1.NodePortRange)
	if !isSnatPolicyDeleted {
		if len(snatPolicy.Status.SnatPortsAllocated) != 0 {
			portinuse = snatPolicy.Status.SnatPortsAllocated
		} else {
			utils.CreatePolicyStatus(portinuse, snatIps[0], &snatPolicy)
		}
		updated := false
		updated = utils.AllocateIpPortRange(portinuse, existingPods, &snatPolicy)
		for _, podlist := range depPods {
			if utils.AllocateIpPortRange(portinuse, &podlist, &snatPolicy) {
				updated = true
			}
		}
		if utils.AllocateIpPortRange(portinuse, nameSpacePods, &snatPolicy) {
			updated = true
		}
		if updated {
			snatPolicy.Status.SnatPortsAllocated = portinuse
			log.Info("Port Range Updated: ", "List of Ports InUse: ", portinuse)
			err = r.client.Status().Update(context.TODO(), &snatPolicy)
			if err != nil {
				return reconcile.Result{}, err
			}
		}
	}
	if len(existingPods.Items) != 0 {
		_, err = r.snatPolicyUpdate(existingPods, &snatPolicy, POD, isSnatPolicyDeleted, "")
		if err != nil {
			return reconcile.Result{}, err
		}
	}

	for _, podlist := range depPods {
		_, err = r.snatPolicyUpdate(&podlist, &snatPolicy, DEPLOYMENT, isSnatPolicyDeleted, "")
		if err != nil {
			return reconcile.Result{}, err
		}
	}
	if len(nameSpacePods.Items) != 0 {
		_, err = r.snatPolicyUpdate(nameSpacePods, &snatPolicy, NAMESPACE, isSnatPolicyDeleted, "")
		if err != nil {
			return reconcile.Result{}, err
		}
	}
	return reconcile.Result{}, nil
}

func (r *ReconcileSnatLocalInfo) snatPolicyUpdate(existingPods *corev1.PodList,
	snatpolicy *aciv1.SnatPolicy, resType string, deleted bool, snatip string) (reconcile.Result, error) {
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
		log.Info("SnatPolicyList ", "Failed to List: ", snatPolicyList)
	}
	for _, pod := range existingPods.Items {
		if resType == "cluster" && pod.Spec.HostNetwork {
			continue
		}
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
			log.Info("Snat Policy Deleted: ", "Snat Policy", deleted)
			if _, ok := localInfo.Spec.LocalInfos[string(pod.ObjectMeta.UID)]; ok {
				inforesType := localInfo.Spec.LocalInfos[string(pod.ObjectMeta.UID)].SnatScope
				if inforesType == POD && (resType != POD) {
					continue
				} else if inforesType == SERVICE && (resType != SERVICE) {
					continue
				} else if inforesType == DEPLOYMENT && (resType == NAMESPACE || resType == CLUSTER) {
					continue
				} else if inforesType == NAMESPACE && resType == CLUSTER {
					continue
				}
				delete(localInfo.Spec.LocalInfos, string(pod.ObjectMeta.UID))
				// This case is possible when some of the POD's are deleted marked in case of Deployment
				localInfos[nodeName] = localInfo
				if len(localInfo.Spec.LocalInfos) == 0 {
					snatIps[nodeName] = snatIp
				}
			}
		} else if pod.Status.Phase == corev1.PodRunning {
			_, err = r.addLocalInfo(&localInfo, &pod, snatpolicy, resType, snatip)
			if err != nil {
				log.Error(err, "Adding localInfo error")
				return reconcile.Result{}, err
			}
			localInfos[nodeName] = localInfo
		}
	}
	if poddeleted {
		for nodeName, snatIp := range snatIps {
			_, err := utils.UpdateSnatPolicyStatus(nodeName, snatpolicy.ObjectMeta.Name, snatIp, r.client)
			if err != nil {
				log.Error(err, "Policy Status Update Failed")
				return reconcile.Result{}, err
			}
		}
	} else if deleted {
		// in the update Old Object's policy status needs to be cleared
		var curpolicy aciv1.SnatPolicy
		curpolicy, err = utils.GetSnatPolicyCR(r.client, snatpolicy.ObjectMeta.Name)
		if err == nil && curpolicy.GetDeletionTimestamp() == nil {
			curpolicy.Status.SnatPortsAllocated = nil
			err = r.client.Status().Update(context.TODO(), &curpolicy)
			if err != nil {
				return reconcile.Result{}, err
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
	request := FilterPodsPerSnatPolicy(r.client, &snatPolicyList, &fristPod)
	if len(request) != 0 && !poddeleted && deleted {
		log.Info("Enque the request again: ", "There is policy match for pod:#####", request)
		r.handleSnatPolicyEvent(request[0])
	}
	return reconcile.Result{}, nil
}
func (r *ReconcileSnatLocalInfo) addLocalInfo(snatlocalinfo *aciv1.SnatLocalInfo, pod *corev1.Pod,
	snatpolicy *aciv1.SnatPolicy, resType string, snatIp string) (reconcile.Result, error) {
	policyname := snatpolicy.GetObjectMeta().GetName()
	log.Info("localinfo", "Snat Policy NAME ### ", policyname)
	var snatip string
	var err error
	if len(snatpolicy.Spec.SnatIp) == 0 {
		snatip = snatIp
	} else {
		snatip, _, _ = utils.GetIPPortRangeForPod(pod.Spec.NodeName, snatpolicy)
	}
	tempLocalInfo := aciv1.LocalInfo{
		PodName:        pod.GetObjectMeta().GetName(),
		PodNamespace:   pod.GetObjectMeta().GetNamespace(),
		SnatIp:         snatip,
		SnatPolicyName: snatpolicy.ObjectMeta.Name,
		SnatScope:      resType,
	}
	if len(snatlocalinfo.Spec.LocalInfos) == 0 && snatlocalinfo.GetObjectMeta().GetName() != pod.Spec.NodeName {
		log.Info("LocalInfo CR is not present", "Creating new one", pod.Spec.NodeName)
		tempMap := make(map[string]aciv1.LocalInfo)
		tempMap[string(pod.ObjectMeta.UID)] = tempLocalInfo
		tempLocalInfoSpec := aciv1.SnatLocalInfoSpec{
			LocalInfos: tempMap,
		}
		snatlocalinfo, _, err = utils.CreateLocalInfoCR(r.client, tempLocalInfoSpec, pod.Spec.NodeName)
		if err != nil {
			log.Error(err, "Create localInfo Failed")
			return reconcile.Result{}, err
		}

	} else {
		// LocaInfo CR is already present, Append localInfo object into Spec's map  and update Locainfo
		scopechange := false
		log.Info("LocalInfo is updated", "Updating the  Spec ####", snatlocalinfo.Spec.LocalInfos)
		if len(snatlocalinfo.Spec.LocalInfos) == 0 {
			tempMap := make(map[string]aciv1.LocalInfo)
			tempMap[string(pod.ObjectMeta.UID)] = tempLocalInfo
			tempLocalInfoSpec := aciv1.SnatLocalInfoSpec{
				LocalInfos: tempMap,
			}
			snatlocalinfo.Spec = tempLocalInfoSpec
		} else {
			log.Info("LocalInfo is updated", "Updating the  Spec For  ####", pod.ObjectMeta.Name)
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
			log.Info("LocalInfo is updated", "Updated 	the  Spec ####", snatlocalinfo.Spec.LocalInfos)
		}
	}
	return reconcile.Result{}, nil
}

func (r *ReconcileSnatLocalInfo) handleDeploymentEvent(request reconcile.Request) (reconcile.Result, error) {
	// revisit this code t write separate API for GetPodNameFromReoncileRequest to fix the names
	depName, namespace, slectorstring := utils.GetPodNameFromReoncileRequest(request.Name)
	dep := &appsv1.Deployment{}
	matches := false
	var snatPolicyName string
	deploymentdeleted := false
	selector := make(map[string]string)
	err := r.client.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: depName}, dep)
	if err != nil && errors.IsNotFound(err) {
		log.Info("handleDeploymentEvent", "deployment not found", depName)
		deploymentdeleted = true
	} else if err != nil {
		return reconcile.Result{}, err
	}
	// Deployment delete cleanup handled as Pod Delete
	if dep.GetObjectMeta().GetDeletionTimestamp() != nil {
		log.Info("handleDeploymentEvent", "deployment deleted", dep)
		deploymentdeleted = true
	}
	snatPolicyList := &aciv1.SnatPolicyList{}
	if err := r.client.List(context.TODO(), &client.ListOptions{Namespace: ""}, snatPolicyList); err != nil {
		return reconcile.Result{}, nil
	}

	if deploymentdeleted {
		err = json.Unmarshal([]byte(slectorstring), &selector)
		if err != nil {
			log.Error(err, "Marshling string Failed")
			return reconcile.Result{}, nil
		}
	} else {
		selector = dep.Spec.Selector.MatchLabels
		snatPolicyName, matches = utils.CheckMatchesLabletoPolicy(snatPolicyList, dep.ObjectMeta.Labels, namespace)
		if matches {
			log.Info("handleDeploymentEvent", "Labels Matches", dep.ObjectMeta.Labels)
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
		log.Error(err, "Pods list failed for deployment")
		return reconcile.Result{}, err
	}
	_, err = r.updatePods(Pods, matches, snatPolicyName, snatPolicyList, DEPLOYMENT)

	if err != nil {
		return reconcile.Result{}, err
	}
	return reconcile.Result{}, nil
}

func (r *ReconcileSnatLocalInfo) handleNameSpaceEvent(request reconcile.Request) (reconcile.Result, error) {
	namespaceName, _, resType := utils.GetPodNameFromReoncileRequest(request.Name)
	namespace := &corev1.Namespace{}
	matches := false
	var snatPolicyName string
	namespacedeleted := false
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: namespaceName}, namespace)
	if err != nil && errors.IsNotFound(err) {
		log.Info("handleNameSpaceEvent", "namespace not found", namespaceName)
		namespacedeleted = true
	} else if err != nil {
		return reconcile.Result{}, err
	}
	// namespace delete cleanup handled as Pod Delete
	if namespace.GetDeletionTimestamp() != nil {
		log.Info("handleNameSpaceEvent", "namespace deleted", namespace)
		namespacedeleted = true
	}
	snatPolicyList := &aciv1.SnatPolicyList{}
	if err := r.client.List(context.TODO(), &client.ListOptions{Namespace: ""}, snatPolicyList); err != nil {
		return reconcile.Result{}, nil
	}
	if !namespacedeleted {
		snatPolicyName, matches = utils.CheckMatchesLabletoPolicy(snatPolicyList, namespace.ObjectMeta.Labels, namespaceName)
		if matches {
			log.Info("handleNameSpaceEvent", "Labels Matches", namespace.ObjectMeta.Labels)
		}
	}
	Pods := &corev1.PodList{}
	err = r.client.List(context.TODO(),
		&client.ListOptions{
			Namespace: namespaceName,
		},
		Pods)
	if err != nil {
		log.Error(err, "Pods list failed for namespace")
		return reconcile.Result{}, err
	}
	_, err = r.updatePods(Pods, matches, snatPolicyName, snatPolicyList, resType)
	if err != nil {
		return reconcile.Result{}, err
	}
	return reconcile.Result{}, nil
}

func (r *ReconcileSnatLocalInfo) handleDeploymentConfigEvent(request reconcile.Request) (reconcile.Result, error) {
	// revisit this code t write separate API for GetPodNameFromReoncileRequest to fix the names
	depName, namespace, slectorstring := utils.GetPodNameFromReoncileRequest(request.Name)
	dep := &openv1.DeploymentConfig{}
	var snatPolicyName string
	deploymentdeleted := false
	selector := make(map[string]string)
	matches := false
	err := r.client.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: depName}, dep)
	if err != nil && errors.IsNotFound(err) {
		log.Error(err, "no matching deployment")
		deploymentdeleted = true
	} else if err != nil {
		return reconcile.Result{}, nil
	}
	// Deployment delete cleanup handled as Pod Delete
	if dep.GetDeletionTimestamp() != nil {
		log.Info("handleDeploymentEvent", "deployment deleted", dep)
		deploymentdeleted = true
	}
	snatPolicyList := &aciv1.SnatPolicyList{}
	if err := r.client.List(context.TODO(), &client.ListOptions{Namespace: ""}, snatPolicyList); err != nil {
		return reconcile.Result{}, nil
	}
	if deploymentdeleted {
		err = json.Unmarshal([]byte(slectorstring), &selector)
		if err != nil {
			log.Error(err, "Marshling string Failed")
			return reconcile.Result{}, nil
		}
	} else {
		selector = dep.Spec.Selector
		snatPolicyName, matches = utils.CheckMatchesLabletoPolicy(snatPolicyList, dep.ObjectMeta.Labels, namespace)
		if matches {
			log.Info("handleDeploymentEvent", "Labels Matches", dep.ObjectMeta.Labels)
		}
	}
	Pods := &corev1.PodList{}
	r.client.List(context.TODO(),
		&client.ListOptions{
			Namespace:     dep.ObjectMeta.Namespace,
			LabelSelector: labels.SelectorFromSet(dep.Spec.Selector),
		},
		Pods)
	_, err = r.updatePods(Pods, matches, snatPolicyName, snatPolicyList, DEPLOYMENT)
	if err != nil {
		return reconcile.Result{}, err
	}
	return reconcile.Result{}, nil
}

func (r *ReconcileSnatLocalInfo) updatePods(Pods *corev1.PodList, matches bool, snatPolicyName string,
	snatPolicyList *aciv1.SnatPolicyList, resType string) (reconcile.Result, error) {

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
		}
		updated := false
		updated = utils.AllocateIpPortRange(portinuse, Pods, &snatPolicy)
		log.Info("Port Range Updated: ", "List of Ports InUse: ", portinuse)
		if updated {
			err = r.client.Status().Update(context.TODO(), &snatPolicy)
			if err != nil {
				return reconcile.Result{}, err
			}
		}
		snactPolicyCheck, err := utils.GetSnatPolicyCR(r.client, snatPolicyName)
		if !reflect.DeepEqual(snactPolicyCheck.Status.SnatPortsAllocated, portinuse) {
			result := reconcile.Result{}
			result.Requeue = true
			return result, nil
		}
		_, err = r.snatPolicyUpdate(Pods, &snatPolicy, resType, false, "")
		if err != nil {
			return reconcile.Result{}, err
		}
	} else {
		for _, item := range snatPolicyList.Items {
			 if item.Status.State != aciv1.Ready {
				continue;
			}
			snatPolicy, err := utils.GetSnatPolicyCR(r.client, item.ObjectMeta.Name)
			if err != nil && errors.IsNotFound(err) {
				log.Error(err, "not matching snatpolicy")
				continue
			} else if err != nil {
				return reconcile.Result{}, err
			}
			_, err = r.snatPolicyUpdate(Pods, &snatPolicy, resType, true, "")
			if err != nil {
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
	ls := make(map[string]string)
	for _, label := range snatPolicy.Spec.Selector.Labels {
		ls[label.Key] = label.Value
	}
	// list all Pods belong to the label
	selector := labels.SelectorFromSet(labels.Set(ls))
	SerivesList := &corev1.ServiceList{}
	if len(snatPolicy.Spec.Selector.Labels) == 0 {
		err = r.client.List(context.TODO(),
			&client.ListOptions{
				Namespace: snatPolicy.Spec.Selector.Namespace,
			},
			SerivesList)
	} else {
		err = r.client.List(context.TODO(),
			&client.ListOptions{
				Namespace:     snatPolicy.Spec.Selector.Namespace,
				LabelSelector: selector,
			},
			SerivesList)
	}
	if err != nil {
		return reconcile.Result{}, err
	}
	if len(SerivesList.Items) == 0 {
		return reconcile.Result{}, nil
	}
	portinuse := make(map[string][]aciv1.NodePortRange)
	for _, service := range SerivesList.Items {
		log.Info("Matching lables for", "Services: ", service)
		Pods := &corev1.PodList{}
		r.client.List(context.TODO(),
			&client.ListOptions{
				Namespace:     service.ObjectMeta.Namespace,
				LabelSelector: labels.SelectorFromSet(service.Spec.Selector),
			},
			Pods)
		if len(service.Status.LoadBalancer.Ingress) == 0 {
			log.Info("No external Loadbalance IP for ", "Service: ", service)
			continue
		}
		snatip := service.Status.LoadBalancer.Ingress[0].IP
		if len(Pods.Items) != 0 {
			if !isSnatPolicyDeleted {
				if len(snatPolicy.Status.SnatPortsAllocated) != 0 {
					portinuse = snatPolicy.Status.SnatPortsAllocated
				} else {
					utils.CreatePolicyStatus(portinuse, snatip, snatPolicy)
				}
				if utils.AllocateIpPortRangeforservice(portinuse, Pods, snatPolicy, snatip) {
					snatPolicy.Status.SnatPortsAllocated = portinuse
					log.Info("Port Range Updated: ", "List of Ports InUse: ", portinuse)
					err = r.client.Status().Update(context.TODO(), snatPolicy)
					if err != nil {
						return reconcile.Result{}, err
					}
				}
			}
			_, err := r.snatPolicyUpdate(Pods, snatPolicy, SERVICE, isSnatPolicyDeleted, snatip)
			if err != nil {
				return reconcile.Result{}, err
			}
		}
	}
	return reconcile.Result{}, nil
}
func (r *ReconcileSnatLocalInfo) handlePodForServiceEvent(request reconcile.Request) (reconcile.Result, error) {
	// Podname: name of the pod for which loop was triggered
	// PolicyName: name of the snatPolicy for respective pod
	podName, snatPolicyName, snatIp := utils.GetPodNameFromReoncileRequest(request.Name)

	// Query this pod using k8s client
	foundPod := &corev1.Pod{}
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: podName, Namespace: request.Namespace}, foundPod)

	if err != nil && errors.IsNotFound(err) {
		log.Info("Pod deleted", "PodName:", request.Name)
		return reconcile.Result{}, nil
	} else if err != nil {
		return reconcile.Result{}, err
	}
	log.Info("POD found:", "Pod name", foundPod.ObjectMeta.Name)
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
		log.Info("Local Info to be deleted: ", "Pod UUID", string(foundPod.ObjectMeta.Name))
		if _, ok := localInfo.Spec.LocalInfos[string(foundPod.ObjectMeta.UID)]; ok {
			nodeName := foundPod.Spec.NodeName
			snatIp := localInfo.Spec.LocalInfos[string(foundPod.ObjectMeta.UID)].SnatIp
			inforesType := localInfo.Spec.LocalInfos[string(foundPod.ObjectMeta.UID)].SnatScope
			if inforesType == "pod" {
				return reconcile.Result{}, nil
			}
			delete(localInfo.Spec.LocalInfos, string(foundPod.ObjectMeta.UID))
			log.Info("Local Info to be deleted: ", "Update Snat Policy Status###", len(localInfo.Spec.LocalInfos))
			if len(localInfo.Spec.LocalInfos) == 0 {
				log.Info("Local Info to be deleted: ", "Update Snat Policy Status###", nodeName)
				_, err = utils.UpdateSnatPolicyStatus(nodeName, snatPolicyName, snatIp, r.client)
				if err != nil {
					log.Error(err, "Policy Status Update Failed")
					return reconcile.Result{}, err
				}
			}
			return utils.UpdateLocalInfoCR(r.client, localInfo)
		}
		return reconcile.Result{}, nil
	}
	if foundPod.Status.Phase == "Running" {
		portinuse := make(map[string][]aciv1.NodePortRange)
		if len(snatPolicy.Status.SnatPortsAllocated) != 0 {
			portinuse = snatPolicy.Status.SnatPortsAllocated
		}
		portrange, exists := utils.GetPortRangeForServiceIP(foundPod.Spec.NodeName, &snatPolicy, snatIp)
		if exists == false {
			var nodePortRnage aciv1.NodePortRange
			nodePortRnage.NodeName = foundPod.Spec.NodeName
			nodePortRnage.PortRange = portrange
			portinuse[snatIp] = append(portinuse[snatIp], nodePortRnage)
			snatPolicy.Status.SnatPortsAllocated = portinuse
			log.Info("Port Range Updated: ", "List of Ports InUse: ", portinuse)
			err = r.client.Status().Update(context.TODO(), &snatPolicy)
			if err != nil {
				return reconcile.Result{}, err
			}
		}
		_, err = r.addLocalInfo(&localInfo, foundPod, &snatPolicy, SERVICE, snatIp)
		if err != nil {
			log.Error(err, "Adding localInfo error")
			return reconcile.Result{}, err
		}
		return utils.UpdateLocalInfoCR(r.client, localInfo)
	}
	return reconcile.Result{}, nil
}
func (r *ReconcileSnatLocalInfo) handleServiceEvent(request reconcile.Request) (reconcile.Result, error) {
	// revisit this code t write separate API for GetPodNameFromReoncileRequest to fix the names
	servicename, namespace, slectorstring := utils.GetPodNameFromReoncileRequest(request.Name)
	log.Info("handleServiceEvent", "name", servicename)
	service := &corev1.Service{}
	matches := false
	var snatPolicyName string
	var snatip string
	servicedeleted := false
	selector := make(map[string]string)
	err := r.client.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: servicename}, service)
	if err != nil && errors.IsNotFound(err) {
		servicedeleted = true
		log.Info("handleServiceEvent", "Service deleted", service)
	} else if err != nil {
		return reconcile.Result{}, nil
	}
	// Service deleted cleanup handled as Pod Delete
	if service.GetObjectMeta().GetDeletionTimestamp() != nil {
		log.Info("handleServiceEvent", "Service deleted", service)
		servicedeleted = true
	}
	snatPolicyList := &aciv1.SnatPolicyList{}
	if err := r.client.List(context.TODO(), &client.ListOptions{Namespace: ""}, snatPolicyList); err != nil {
		return reconcile.Result{}, nil
	}
	if servicedeleted {
		err = json.Unmarshal([]byte(slectorstring), &selector)
		if err != nil {
			log.Error(err, "Marshling string Failed")
			return reconcile.Result{}, nil
		}
	} else {
		selector = service.Spec.Selector
		snatPolicyName, matches = utils.CheckMatchesLabletoPolicy(snatPolicyList, service.ObjectMeta.Labels, namespace)
		if matches {
			log.Info("handleServiceEvent", "Labels Matches", service.ObjectMeta.Labels)
		}
	}
	log.Info("handleServiceEvent:", "Selector###", selector)
	Pods := &corev1.PodList{}
	err = r.client.List(context.TODO(),
		&client.ListOptions{
			Namespace:     namespace,
			LabelSelector: labels.SelectorFromSet(selector),
		},
		Pods)
	if err != nil {
		log.Error(err, "Listing Pods Failed")
		return reconcile.Result{}, nil
	}
	if len(Pods.Items) == 0 {
		log.Info("handleServiceEvent:", "length Zero###", Pods)
		return reconcile.Result{}, nil
	}
	if matches == true {
		if len(service.Status.LoadBalancer.Ingress) == 0 {
			log.Info("No external Loadbalance IP for ", "Service: ", service)
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
		if snatPolicy.GetDeletionTimestamp() != nil {
			return reconcile.Result{}, err
		}

		portinuse := make(map[string][]aciv1.NodePortRange)
		if len(snatPolicy.Status.SnatPortsAllocated) != 0 {
			portinuse = snatPolicy.Status.SnatPortsAllocated
		} else {
			utils.CreatePolicyStatus(portinuse, snatip, &snatPolicy)
		}

		if utils.AllocateIpPortRangeforservice(portinuse, Pods, &snatPolicy, snatip) {
			snatPolicy.Status.SnatPortsAllocated = portinuse
			log.Info("Port Range Updated: ", "List of Ports InUse: ", portinuse)
			err = r.client.Status().Update(context.TODO(), &snatPolicy)
			if err != nil {
				return reconcile.Result{}, err
			}
		}
		if servicedeleted {
			_, err = r.snatPolicyUpdate(Pods, &snatPolicy, SERVICE, true, snatip)
		} else {
			_, err = r.snatPolicyUpdate(Pods, &snatPolicy, SERVICE, false, snatip)
		}
		if err != nil {
			log.Info("SnatPolicy Update For: ", "Failed: ", snatPolicy)
			return reconcile.Result{}, err
		}

	} else {
		for _, item := range snatPolicyList.Items {
			 if item.Status.State != aciv1.Ready {
                                continue;
                        }
			snatPolicy, err := utils.GetSnatPolicyCR(r.client, item.ObjectMeta.Name)
			if err != nil && errors.IsNotFound(err) {
				log.Error(err, "not matching snatpolicy")
				continue
			} else if err != nil {
				return reconcile.Result{}, err
			}
			_, err = r.snatPolicyUpdate(Pods, &snatPolicy, SERVICE, true, snatip)
			if err != nil {
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
			utils.CreatePolicyStatus(portinuse, snatIps[0], snatPolicy)
		}
		if utils.AllocateIpPortRange(portinuse, Pods, snatPolicy) {
			snatPolicy.Status.SnatPortsAllocated = portinuse
			log.Info("Port Range Updated: ", "List of Ports InUse: ", portinuse)
			err = r.client.Status().Update(context.TODO(), snatPolicy)
			if err != nil {
				return reconcile.Result{}, err
			}
		}
	}
	_, err = r.snatPolicyUpdate(Pods, snatPolicy, CLUSTER, isSnatPolicyDeleted, "")
	if err != nil {
		return reconcile.Result{}, err
	}
	return reconcile.Result{}, nil
}

func (r *ReconcileSnatLocalInfo) handleSnatPolicyForNameSpace(snatPolicy *aciv1.SnatPolicy, status string) (reconcile.Result, error) {
	 if snatPolicy.Status.State != aciv1.Ready {
                return reconcile.Result{}, nil
        }
	nameSpacePods := &corev1.PodList{}
	isSnatPolicyDeleted := snatPolicy.GetDeletionTimestamp() != nil
	// SnatPolicy is updated so delete the old Object
	if status == "deleted" {
		isSnatPolicyDeleted = true
	}
	err := r.client.List(context.TODO(),
		&client.ListOptions{
			Namespace: snatPolicy.Spec.Selector.Namespace,
		},
		nameSpacePods)
	if err != nil {
		log.Info("Failed to List pods: ", "For namespace: ", snatPolicy.Spec.Selector.Namespace)
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
			utils.CreatePolicyStatus(portinuse, snatIps[0], snatPolicy)
		}
		if utils.AllocateIpPortRange(portinuse, nameSpacePods, snatPolicy) {
			snatPolicy.Status.SnatPortsAllocated = portinuse
			log.Info("Port Range Updated: ", "List of Ports InUse: ", portinuse)
			err = r.client.Status().Update(context.TODO(), snatPolicy)
			if err != nil {
				return reconcile.Result{}, err
			}
		}
	}
	if len(nameSpacePods.Items) != 0 {
		_, err = r.snatPolicyUpdate(nameSpacePods, snatPolicy, NAMESPACE, isSnatPolicyDeleted, "")
		if err != nil {
			return reconcile.Result{}, err
		}
	}
	return reconcile.Result{}, nil
}
