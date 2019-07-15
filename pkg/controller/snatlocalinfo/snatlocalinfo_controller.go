package snatlocalinfo

import (
	"context"
	"os"
	"reflect"
	"strings"

	"github.com/noironetworks/snat-operator/cmd/manager/utils"
	aciv1 "github.com/noironetworks/snat-operator/pkg/apis/aci/v1"
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
	if strings.HasPrefix(request.Name, "snat-policyforpod") {
		result, err := r.handlePodEvent(request)
		return result, err
	} else if strings.HasPrefix(request.Name, "snat-policy") {
		result, err := r.handleSnatPolicyEvent(request)
		return result, err
	} else if strings.HasPrefix(request.Name, "snat-deployment") {
		result, err := r.handleDeploymentEvent(request)
		return result, err
	} else if strings.HasPrefix(request.Name, "snat-namespace") {
		result, err := r.handleNameSpaceEvent(request)
		return result, err
	} else {
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
			inforesType := localInfo.Spec.LocalInfos[string(foundPod.ObjectMeta.UID)].SnatScope
			if inforesType == "pod" && (resType == "deployment" || resType == "namespace") {
				return reconcile.Result{}, nil
			} else if inforesType == "deployment" && resType == "namespace" {
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
		return r.addLocalInfo(&localInfo, foundPod, &snatPolicy, resType)
	}
	return reconcile.Result{}, nil
}
func (r *ReconcileSnatLocalInfo) handleSnatPolicyEvent(request reconcile.Request) (reconcile.Result, error) {
	log.Info("Snat Policy Update Event ", "Snat Policy: ", request)
	_, snatPolicyName, _ := utils.GetPodNameFromReoncileRequest(request.Name)
	snatPolicy, err := utils.GetSnatPolicyCR(r.client, snatPolicyName)
	if err != nil && errors.IsNotFound(err) {
		log.Error(err, "not matching snatpolicy")
		return reconcile.Result{}, nil
	} else if err != nil {
		return reconcile.Result{}, err
	}
	isSnatPolicyDeleted := snatPolicy.GetDeletionTimestamp() != nil
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
			//Namespace: snatPolicy.Spec.Selector.Namespace,
			LabelSelector: selector,
		},
		existingPods)
	if err != nil {
		return reconcile.Result{}, err
	}
	if len(existingPods.Items) != 0 {
		log.Info("Matching lables for", "POD's: ", existingPods)
	}
	//  list all the deployments
	deploymentList := &appsv1.DeploymentList{}
	err = r.client.List(context.TODO(),
		&client.ListOptions{
			LabelSelector: selector,
		},
		deploymentList)
	if err != nil {
		return reconcile.Result{}, err
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
			// Create  dummy entry
			snatIps := utils.ExpandCIDRs(snatPolicy.Spec.SnatIp)
			var nodePortRnage aciv1.NodePortRange
			portinuse[snatIps[0]] = append(portinuse[snatIps[0]], nodePortRnage)
			snatPolicy.Status.SnatPortsAllocated = portinuse
			log.Info("Port Range Updated: ", "List of Ports InUse: ", portinuse)
			err = r.client.Status().Update(context.TODO(), &snatPolicy)
			if err != nil {
				return reconcile.Result{}, err
			}

		}
		updated := false
		for _, pod := range existingPods.Items {
			snatip, portrange, exists := utils.GetIPPortRangeForPod(pod.Spec.NodeName, &snatPolicy)
			if exists == false && pod.GetObjectMeta().GetDeletionTimestamp() == nil {
				var nodePortRnage aciv1.NodePortRange
				nodePortRnage.NodeName = pod.Spec.NodeName
				nodePortRnage.PortRange = portrange
				portinuse[snatip] = append(portinuse[snatip], nodePortRnage)
				updated = true
			}
		}
		for _, podlist := range depPods {
			for _, pod := range podlist.Items {
				snatip, portrange, exists := utils.GetIPPortRangeForPod(pod.Spec.NodeName, &snatPolicy)
				if exists == false && pod.GetObjectMeta().GetDeletionTimestamp() == nil {
					var nodePortRnage aciv1.NodePortRange
					nodePortRnage.NodeName = pod.Spec.NodeName
					nodePortRnage.PortRange = portrange
					portinuse[snatip] = append(portinuse[snatip], nodePortRnage)
					updated = true
				}
			}
		}
		for _, podlist := range mameSpacePods {
			for _, pod := range podlist.Items {
				snatip, portrange, exists := utils.GetIPPortRangeForPod(pod.Spec.NodeName, &snatPolicy)
				if exists == false && pod.GetObjectMeta().GetDeletionTimestamp() == nil {
					var nodePortRnage aciv1.NodePortRange
					nodePortRnage.NodeName = pod.Spec.NodeName
					nodePortRnage.PortRange = portrange
					portinuse[snatip] = append(portinuse[snatip], nodePortRnage)
					updated = true
				}
			}
		}
		if updated {
			snatPolicy.Status.SnatPortsAllocated = portinuse
			log.Info("Port Range Updated: ", "List of Ports InUse: ", portinuse)
			err = r.client.Status().Update(context.TODO(), &snatPolicy)
			if err != nil {
				return reconcile.Result{}, err
			}
		}
		snatPolicy, err = utils.GetSnatPolicyCR(r.client, snatPolicyName)
		if !reflect.DeepEqual(snatPolicy.Status.SnatPortsAllocated, portinuse) {
			result := reconcile.Result{}
			result.Requeue = true
			return result, nil
		}
	}
	if len(existingPods.Items) != 0 {
		_, err = r.snatPolicyUpdate(existingPods, &snatPolicy, "pod", isSnatPolicyDeleted)
		if err != nil {
			return reconcile.Result{}, err
		}
	}

	for _, podlist := range depPods {
		_, err = r.snatPolicyUpdate(&podlist, &snatPolicy, "deployment", isSnatPolicyDeleted)
		if err != nil {
			return reconcile.Result{}, err
		}
	}

	for _, podlist := range mameSpacePods {
		_, err = r.snatPolicyUpdate(&podlist, &snatPolicy, "namespace", isSnatPolicyDeleted)
		if err != nil {
			return reconcile.Result{}, err
		}
	}

	return reconcile.Result{}, err
}
func (r *ReconcileSnatLocalInfo) snatPolicyUpdate(existingPods *corev1.PodList,
	snatpolicy *aciv1.SnatPolicy, resType string, deleted bool) (reconcile.Result, error) {
	for _, pod := range existingPods.Items {
		if pod.GetObjectMeta().GetDeletionTimestamp() != nil {
			continue
		}
		localInfo, err := utils.GetLocalInfoCR(r.client, pod.Spec.NodeName, os.Getenv("ACI_SNAT_NAMESPACE"))
		if err != nil && errors.IsNotFound(err) {
			log.Error(err, "localInfo error")
			return reconcile.Result{}, nil
		}
		if deleted {
			log.Info("nat Policy Deleted: ", "Snat Policy", deleted)
			if _, ok := localInfo.Spec.LocalInfos[string(pod.ObjectMeta.UID)]; ok {
				nodeName := pod.Spec.NodeName
				snatIp := localInfo.Spec.LocalInfos[string(pod.ObjectMeta.UID)].SnatIp
				inforesType := localInfo.Spec.LocalInfos[string(pod.ObjectMeta.UID)].SnatScope
				if inforesType == "pod" && (resType == "deployment" || resType == "namespace") {
					continue
				} else if inforesType == "deployment" && resType == "namespace" {
					continue
				}
				delete(localInfo.Spec.LocalInfos, string(pod.ObjectMeta.UID))
				if len(localInfo.Spec.LocalInfos) == 0 {
					_, err = utils.UpdateSnatPolicyStatus(nodeName, snatpolicy.ObjectMeta.Name, snatIp, r.client)
					if err != nil {
						log.Error(err, "Policy Status Update Failed")
						return reconcile.Result{}, err
					}
				}
				_, err = utils.UpdateLocalInfoCR(r.client, localInfo)
				if err != nil {
					log.Error(err, "updating localInfo error")
					return reconcile.Result{}, nil
				}
			}
		} else if pod.Status.Phase == corev1.PodRunning {
			_, err = r.addLocalInfo(&localInfo, &pod, snatpolicy, resType)
			if err != nil {
				log.Error(err, "Adding localInfo error")
				return reconcile.Result{}, err
			}
		}

	}
	return reconcile.Result{}, nil
}
func (r *ReconcileSnatLocalInfo) addLocalInfo(snatlocalinfo *aciv1.SnatLocalInfo, pod *corev1.Pod,
	snatpolicy *aciv1.SnatPolicy, resType string) (reconcile.Result, error) {
	policyname := snatpolicy.GetObjectMeta().GetName()
	log.Info("localinfo", "Snat Policy NAME ### ", policyname)
	snatip, _, _ := utils.GetIPPortRangeForPod(pod.Spec.NodeName, snatpolicy)
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
		return utils.CreateLocalInfoCR(r.client, tempLocalInfoSpec, pod.Spec.NodeName)
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
			scopechange = true
		} else {
			if info, ok := snatlocalinfo.Spec.LocalInfos[string(pod.ObjectMeta.UID)]; ok {
				if info.SnatScope == "deployment" && resType == "pod" {
					scopechange = true
					tempLocalInfo.SnatScope = "pod"
				} else if info.SnatScope == "namespace" && resType == "pod" {
					scopechange = true
					tempLocalInfo.SnatScope = "pod"
				} else if info.SnatScope == "namespace" && resType == "deployment" {
					scopechange = true
					tempLocalInfo.SnatScope = "deployment"
				}
			} else {
				tempLocalInfo.SnatScope = resType
				scopechange = true
			}
		}

		if scopechange {
			return utils.UpdateLocalInfoCR(r.client, *snatlocalinfo)
		}
	}
	return reconcile.Result{}, nil

}

func (r *ReconcileSnatLocalInfo) handleDeploymentEvent(request reconcile.Request) (reconcile.Result, error) {
	// revisit this code t write separate API for GetPodNameFromReoncileRequest to fix the names
	depName, namespace, resType := utils.GetPodNameFromReoncileRequest(request.Name)
	dep := &appsv1.Deployment{}
	matches := false
	var snatPolicyName string
	err := r.client.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: depName}, dep)
	if err != nil && errors.IsNotFound(err) {
		log.Error(err, "no matching deployment")
		return reconcile.Result{}, nil
	} else if err != nil {
		return reconcile.Result{}, err
	}
	// Deployment delete cleanup handled as Pod Delete
	if dep.GetDeletionTimestamp() != nil {
		log.Info("handleDeploymentEvent", "deployment deleted", dep)
		return reconcile.Result{}, nil
	}
	snatPolicyList := &aciv1.SnatPolicyList{}
	if err := r.client.List(context.TODO(), &client.ListOptions{Namespace: ""}, snatPolicyList); err != nil {
		return reconcile.Result{}, nil
	}
	for _, item := range snatPolicyList.Items {
		if utils.MatchLabels(item.Spec.Selector.Labels, dep.ObjectMeta.Labels) {
			log.Info("handleDeploymentEvent", "Labels Matches", item.Spec.Selector.Labels)
			matches = true
			snatPolicyName = item.ObjectMeta.Name
			break
		}
	}
	Pods := &corev1.PodList{}
	r.client.List(context.TODO(),
		&client.ListOptions{
			Namespace:     dep.ObjectMeta.Namespace,
			LabelSelector: labels.SelectorFromSet(dep.Spec.Selector.MatchLabels),
		},
		Pods)
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
		for _, pod := range Pods.Items {
			snatip, portrange, exists := utils.GetIPPortRangeForPod(pod.Spec.NodeName, &snatPolicy)
			if exists == false && pod.GetObjectMeta().GetDeletionTimestamp() == nil {
				var nodePortRnage aciv1.NodePortRange
				nodePortRnage.NodeName = pod.Spec.NodeName
				nodePortRnage.PortRange = portrange
				portinuse[snatip] = append(portinuse[snatip], nodePortRnage)
				updated = true
			}
		}
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
		_, err = r.snatPolicyUpdate(Pods, &snatPolicy, resType, false)
		if err != nil {
			return reconcile.Result{}, err
		}
	} else {
		for _, item := range snatPolicyList.Items {
			snatPolicy, err := utils.GetSnatPolicyCR(r.client, item.ObjectMeta.Name)
			if err != nil && errors.IsNotFound(err) {
				log.Error(err, "no matching snatpolicy")
				return reconcile.Result{}, nil
			} else if err != nil {
				log.Error(err, " Get snatpolicy failed")
				return reconcile.Result{}, err
			}
			_, err = r.snatPolicyUpdate(Pods, &snatPolicy, resType, true)
			if err != nil {
				return reconcile.Result{}, err
			}
		}
	}
	return reconcile.Result{}, nil
}

func (r *ReconcileSnatLocalInfo) handleNameSpaceEvent(request reconcile.Request) (reconcile.Result, error) {
	namespaceName, _, resType := utils.GetPodNameFromReoncileRequest(request.Name)
	namespace := &corev1.Namespace{}
	matches := false
	var snatPolicyName string
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: namespaceName}, namespace)
	if err != nil && errors.IsNotFound(err) {
		log.Error(err, "not matching namespace")
		return reconcile.Result{}, nil
	} else if err != nil {
		return reconcile.Result{}, err
	}
	// namespace delete cleanup handled as Pod Delete
	if namespace.GetDeletionTimestamp() != nil {
		log.Info("handleNameSpaceEvent", "namespace deleted", namespace)
		return reconcile.Result{}, nil
	}
	snatPolicyList := &aciv1.SnatPolicyList{}
	if err := r.client.List(context.TODO(), &client.ListOptions{Namespace: ""}, snatPolicyList); err != nil {
		return reconcile.Result{}, nil
	}
	for _, item := range snatPolicyList.Items {
		if utils.MatchLabels(item.Spec.Selector.Labels, namespace.ObjectMeta.Labels) {
			log.Info("handleNameSpaceEvent", "Labels Matches", item.Spec.Selector.Labels)
			matches = true
			snatPolicyName = item.ObjectMeta.Name
			break
		}
	}
	Pods := &corev1.PodList{}
	r.client.List(context.TODO(),
		&client.ListOptions{
			Namespace: namespaceName,
		},
		Pods)
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
		for _, pod := range Pods.Items {
			snatip, portrange, exists := utils.GetIPPortRangeForPod(pod.Spec.NodeName, &snatPolicy)
			if exists == false && pod.GetObjectMeta().GetDeletionTimestamp() == nil {
				var nodePortRnage aciv1.NodePortRange
				nodePortRnage.NodeName = pod.Spec.NodeName
				nodePortRnage.PortRange = portrange
				portinuse[snatip] = append(portinuse[snatip], nodePortRnage)
				updated = true
			}
		}
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
		_, err = r.snatPolicyUpdate(Pods, &snatPolicy, resType, false)
		if err != nil {
			return reconcile.Result{}, err
		}
	} else {
		for _, item := range snatPolicyList.Items {
			snatPolicy, err := utils.GetSnatPolicyCR(r.client, item.ObjectMeta.Name)
			if err != nil && errors.IsNotFound(err) {
				log.Error(err, "not matching snatpolicy")
				return reconcile.Result{}, nil
			} else if err != nil {
				return reconcile.Result{}, err
			}
			_, err = r.snatPolicyUpdate(Pods, &snatPolicy, resType, true)
			if err != nil {
				return reconcile.Result{}, err
			}
		}
	}
	return reconcile.Result{}, nil
}
