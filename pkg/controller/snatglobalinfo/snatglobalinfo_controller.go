package snatglobalinfo

import (
	"context"
	"os"
	"strings"

	"github.com/noironetworks/snat-operator/cmd/manager/utils"
	aciv1 "github.com/noironetworks/snat-operator/pkg/apis/aci/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/uuid"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_snatglobalinfo")

// Add creates a new SnatGlobalInfo Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileSnatGlobalInfo{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("snatglobalinfo-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource SnatGlobalInfo
	err = c.Watch(&source.Kind{Type: &aciv1.SnatGlobalInfo{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	err = c.Watch(&source.Kind{Type: &aciv1.SnatLocalInfo{}}, &handler.EnqueueRequestsFromMapFunc{ToRequests: HandleLocalInfosMapper(mgr.GetClient(), []predicate.Predicate{})})
	if err != nil {
		return err
	}
	return nil
}

// blank assignment to verify that ReconcileSnatGlobalInfo implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileSnatGlobalInfo{}

// ReconcileSnatGlobalInfo reconciles a SnatGlobalInfo object
type ReconcileSnatGlobalInfo struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileSnatGlobalInfo) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling SnatGlobalInfo")
	if strings.Contains(request.Name, "snat-localinfo$") {
		localInfoName := request.Name[len("snat-localinfo$"):]
		result, err := r.handleLocalinfoEvent(localInfoName)
		return result, err
	} else {
		// Fetch the SnatGlobalInfo instance
		instance := &aciv1.SnatGlobalInfo{}
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

// This function handles Localinfo events which are triggering snatGlobalInfo's reconcile loop
func (r *ReconcileSnatGlobalInfo) handleLocalinfoEvent(name string) (reconcile.Result, error) {

	// Fetch the SnatLocainfo instance
	instance := &aciv1.SnatLocalInfo{}
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: name, Namespace: os.Getenv("ACI_SNAT_NAMESPACE")}, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			isSnatLocalInfoDeleted := instance.GetDeletionTimestamp() != nil
			if isSnatLocalInfoDeleted {
				//delete(globalInfo.Spec.GlobalInfos, instance.ObjectMeta.Name)
				log.Info("Deleted LOCAL CR for Node: ", "Updating the GlobalInfo: ", instance.ObjectMeta.Name)
				//return utils.UpdateGlobalInfoCR(r.client, *globalInfo)
			}
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}
	isSnatLocalInfoDeleted := instance.GetDeletionTimestamp() != nil
	if isSnatLocalInfoDeleted {
		//delete(globalInfo.Spec.GlobalInfos, instance.ObjectMeta.Name)
		log.Info("Deleted LOCAL CR2 for Node #####", "Updating the GlobalInfo", instance.ObjectMeta.Name)
		//return utils.UpdateGlobalInfoCR(r.client, *globalInfo)
	}

	// Create  get the local ip -> Snat Policy refrences
	localips := make(map[string][]string)
	var snatip string
	for _, v := range instance.Spec.LocalInfos {
		localips[v.SnatIp] = append(localips[v.SnatIp], v.SnatPolicyName)
		snatip = v.SnatIp
	}
	nodeinfo, err := utils.GetNodeInfoCRObject(r.client, instance.ObjectMeta.Name)
	if err != nil {
		log.Error(err, "Failed to Get NodeInfo ")
		return reconcile.Result{}, err
	}
	// Get SnatGlobalInfo instance
	globalInfo := &aciv1.SnatGlobalInfo{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: os.Getenv("ACI_SNAGLOBALINFO_NAME"),
		Namespace: os.Getenv("ACI_SNAT_NAMESPACE")}, globalInfo)
	if err != nil {
		if errors.IsNotFound(err) {
			// Create SnatGlobalInfo Object
			snatPolicy, err := utils.GetSnatPolicyCR(r.client, localips[snatip][0])
			if err != nil && errors.IsNotFound(err) {
				log.Error(err, "not matching snatpolicy")
				return reconcile.Result{}, nil
			} else if err != nil {
				return reconcile.Result{}, err
			}
			_, portrange, _ := utils.GetIPPortRangeForPod(instance.ObjectMeta.Name, &snatPolicy)
			if err != nil {
				return reconcile.Result{}, err
			}
			globalInfos := []aciv1.GlobalInfo{}
			portlist := []aciv1.PortRange{}
			portlist = append(portlist, portrange)
			// get Mac Addres
			for snatIp, _ := range localips {
				temp := aciv1.GlobalInfo{
					MacAddress: nodeinfo.Spec.Macaddress,
					PortRanges: portlist,
					SnatIp:     snatIp,
					SnatIpUid:  string(uuid.NewUUID()),
					Protocols:  []string{"tcp", "udp", "icmp"},
				}

				globalInfos = append(globalInfos, temp)
			}
			tempMap := make(map[string][]aciv1.GlobalInfo)
			tempMap[instance.ObjectMeta.Name] = globalInfos
			log.Info("Global CR is not present", "Creating new one", tempMap)
			spec := aciv1.SnatGlobalInfoSpec{
				GlobalInfos: tempMap,
			}
			return utils.CreateSnatGlobalInfoCR(r.client, spec)
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	} else {
		// update snatGlobalInfo object
		// GlobalInfo CR is already present, Append GlobalInfo object into Spec's map  and update Globalinfo
		var globalInfos []aciv1.GlobalInfo
		if globalInfo.Spec.GlobalInfos[instance.ObjectMeta.Name] != nil {
			globalInfos = append(globalInfos, globalInfo.Spec.GlobalInfos[instance.ObjectMeta.Name]...)
		}
		update := false
		// check if local info for ip deleted then update the Global Info
		for i, v := range globalInfos {
			if len(localips[v.SnatIp]) == 0 {
				globalInfos[i] = globalInfos[len(globalInfos)-1]
				globalInfos = globalInfos[:len(globalInfos)-1]
				update = true
				log.Info("Update Global GR for Deleted LOCAL CR #####", "Creating new one", globalInfos)
			}
		}
		// Check for any addition in local Info
		for snatIp, _ := range localips {
			found := false
			for _, v := range globalInfos {
				if snatIp == v.SnatIp {
					found = true
				}
			}
			if found == false {
				snatPolicy, err := utils.GetSnatPolicyCR(r.client, localips[snatip][0])
				if err != nil && errors.IsNotFound(err) {
					log.Error(err, "not matching snatpolicy")
					return reconcile.Result{}, nil
				} else if err != nil {
					return reconcile.Result{}, err
				}
				_, portrange, _ := utils.GetIPPortRangeForPod(instance.ObjectMeta.Name, &snatPolicy)
				if err != nil {
					log.Error(err, "Update Global CR for getting PortsRage  FAILED#####", portrange)
					return reconcile.Result{}, err
				}
				log.Info("Update Global CR for getting PortsRage  #####", "Portrage:", portrange)
				portlist := []aciv1.PortRange{}
				portlist = append(portlist, portrange)
				temp := aciv1.GlobalInfo{
					MacAddress: nodeinfo.Spec.Macaddress,
					PortRanges: portlist,
					SnatIp:     snatIp,
					SnatIpUid:  string(uuid.NewUUID()),
					Protocols:  []string{"tcp", "udp", "icmp"},
				}
				globalInfos = append(globalInfos, temp)
				update = true
			}
		}
		if update {
			log.Info("Update for LOCAL CR is Received", "calling Global CR Update", globalInfo)
			if len(globalInfos) == 0 {
				delete(globalInfo.Spec.GlobalInfos, instance.ObjectMeta.Name)
			} else {
				if globalInfo.Spec.GlobalInfos == nil {
					tempMap := make(map[string][]aciv1.GlobalInfo)
					tempMap[instance.ObjectMeta.Name] = globalInfos
					globalInfo.Spec.GlobalInfos = tempMap
				} else {
					globalInfo.Spec.GlobalInfos[instance.ObjectMeta.Name] = globalInfos
				}
			}
			return utils.UpdateGlobalInfoCR(r.client, *globalInfo)
		}
	}
	return reconcile.Result{}, nil
}
