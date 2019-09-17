package utils

import (
	"context"
	"os"
	"reflect"
	"strconv"
	"strings"

	// nodeinfo "github.com/noironetworks/aci-containers/pkg/nodeinfo/apis/aci.nodeinfo/v1"

	aciv1 "github.com/noironetworks/snat-operator/pkg/apis/aci/v1"
	"github.com/prometheus/common/log"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	MAX_PORT = 65000
	MIN_PORT = 5000
)
const PORTPERNODES = 3000

// Get nodeinfo object matching given name of the node
// Optimization can be done here:
// if we know namespace of this nodeinfo object then we can type request.NamespacedName{Name: , Namespace:}
// in Get call and directly get the object instead of doing List and iterating.
// But for that namespace has to be knowen. We can push aci-containers-system / kube-system inserted as ENV var
// in this container then we can refer to that.
func GetNodeInfoCRObject(c client.Client, nodeName string) (aciv1.NodeInfo, error) {
	nodeinfoList := &aciv1.NodeInfoList{}
	err := c.List(context.TODO(), &client.ListOptions{Namespace: ""}, nodeinfoList)
	if err != nil && errors.IsNotFound(err) {
		UtilLog.Error(err, "Cound not find nodeinfo object")
		return aciv1.NodeInfo{}, err
	}

	for _, item := range nodeinfoList.Items {
		if item.ObjectMeta.Name == nodeName {
			UtilLog.Info("Nodeinfo object found", "For NodeName:", item)
			return item, nil
		}
	}
	return aciv1.NodeInfo{}, err

}

// Given a reconcile request name, it extracts out node name by omiiting node-event- from it
func GetNodeNameFromReoncileRequest(requestName string) string {
	if strings.HasPrefix(requestName, "node-event-") {
		return requestName[len("node-event-"):]
	}
	return requestName
}

// Given a nodeName, return LocalInfo CR object if present
func GetLocalInfoCR(c client.Client, nodeName, namespace string) (aciv1.SnatLocalInfo, error) {

	foundLocalIfo := &aciv1.SnatLocalInfo{}
	err := c.Get(context.TODO(), types.NamespacedName{Name: nodeName, Namespace: namespace}, foundLocalIfo)
	if err != nil && errors.IsNotFound(err) {
		log.Info("LocalInfo not present ", "foundLocalInfo:", nodeName)
		return aciv1.SnatLocalInfo{}, nil
	} else if err != nil {
		return aciv1.SnatLocalInfo{}, err
	}

	return *foundLocalIfo, nil
}

// Given a policyName, return SnatPolicy CR object if present
func GetSnatPolicyCR(c client.Client, policyName string) (aciv1.SnatPolicy, error) {

	foundSnatPolicy := &aciv1.SnatPolicy{}
	err := c.Get(context.TODO(), types.NamespacedName{Name: policyName, Namespace: os.Getenv("WATCH_NAMESPACE")}, foundSnatPolicy)
	if err != nil && errors.IsNotFound(err) {
		log.Info("SnatPolicy not present", "foundSnatPolicy:", policyName)
		return aciv1.SnatPolicy{}, err
	} else if err != nil {
		return aciv1.SnatPolicy{}, err
	}

	return *foundSnatPolicy, nil
}

// createSnatLocalInfoCR Creates a SnatLocalInfo CR
func CreateLocalInfoCR(c client.Client, localInfoSpec aciv1.SnatLocalInfoSpec,
	nodeName string) (*aciv1.SnatLocalInfo, reconcile.Result, error) {

	obj := &aciv1.SnatLocalInfo{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nodeName,
			Namespace: os.Getenv("ACI_SNAT_NAMESPACE"),
		},
		Spec: localInfoSpec,
	}
	err := c.Create(context.TODO(), obj)
	if err != nil {
		log.Error(err, "failed to create a snat locainfo cr")
		return obj, reconcile.Result{}, err
	}
	log.Info("Created localinfo object", "SnatLocalInfo", obj)
	return obj, reconcile.Result{}, nil
}

// Delete SnatLocalInfoCR Creates a SnatLocalInfo CR
func DeleteLocalInfoCR(c client.Client, localInfoSpec aciv1.SnatLocalInfoSpec,
	nodeName string) (reconcile.Result, error) {

	obj := &aciv1.SnatLocalInfo{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nodeName,
			Namespace: os.Getenv("ACI_SNAT_NAMESPACE"),
		},
		Spec: localInfoSpec,
	}
	err := c.Delete(context.TODO(), obj)
	if err != nil {
		log.Error(err, "failed to Delete a snat locainfo cr")
		return reconcile.Result{}, err
	}
	log.Info("Deleted localinfo object", "SnatLocalInfo", obj)
	return reconcile.Result{}, nil
}

// UpdateSnatLocalInfoCR Updates a SnatLocalInfo CR
func UpdateLocalInfoCR(c client.Client, localInfo aciv1.SnatLocalInfo) (reconcile.Result, error) {

	err := c.Update(context.TODO(), &localInfo)
	if err != nil {
		log.Error(err, "failed to update a snat locainfo cr")
		return reconcile.Result{}, err
	}
	log.Info("Updated localinfo object", "SnatLocalInfo", localInfo)
	return reconcile.Result{}, nil
}

// createSnatGlobalInfoCR Creates a SnatGlobalInfo CR
func CreateSnatGlobalInfoCR(c client.Client, globalInfoSpec aciv1.SnatGlobalInfoSpec) (reconcile.Result, error) {
	obj := &aciv1.SnatGlobalInfo{
		ObjectMeta: metav1.ObjectMeta{
			Name:      os.Getenv("ACI_SNAGLOBALINFO_NAME"),
			Namespace: os.Getenv("ACI_SNAT_NAMESPACE"),
		},
		Spec: globalInfoSpec,
	}
	err := c.Create(context.TODO(), obj)
	if err != nil {
		log.Error(err, "failed to create a snat global cr")
		return reconcile.Result{}, err
	}
	log.Info("Created globalInfo object", "SnatGlobalInfo", obj)
	return reconcile.Result{}, nil
}

// UpdateSnatGlobalInfoCR Updates a SnatGlobalInfo CR
func UpdateGlobalInfoCR(c client.Client, globalInfo aciv1.SnatGlobalInfo) (reconcile.Result, error) {

	err := c.Update(context.TODO(), &globalInfo)
	if err != nil {
		log.Error(err, "failed to update a snat globalInfo cr")
		return reconcile.Result{}, err
	}
	log.Info("Updated globalInfo object", "SnatGlobalinfo", globalInfo)
	return reconcile.Result{}, nil
}

// Get portRange from snat-operator-config configMap
func getPortRangeFromConfigMap(c client.Client) (aciv1.PortRange, int) {
	cMap := &corev1.ConfigMap{}
	err := c.Get(context.Background(), client.ObjectKey{
		Namespace: "aci-containers-system",
		Name:      "snat-operator-config",
	}, cMap)
	var resultPortRange aciv1.PortRange
	resultPortRange.Start = MIN_PORT
	resultPortRange.End = MAX_PORT
	if err != nil {
		return resultPortRange, PORTPERNODES
	}
	data := cMap.Data
	start, err1 := strconv.Atoi(data["start"])
	end, err2 := strconv.Atoi(data["end"])
	portsPerNode, err3 := strconv.Atoi(data["ports-per-node"])
	if err1 != nil || err2 != nil || err3 != nil ||
		start < 5000 || end > 65000 || start > end || portsPerNode > end-start+1 {
		return resultPortRange, PORTPERNODES
	}
	resultPortRange.Start = start
	resultPortRange.End = end
	return resultPortRange, portsPerNode
}

// Get IP and port for pod for which notification has come to reconcile loop
func GetIPPortRangeForPod(c client.Client, NodeName string,
	snatpolicy *aciv1.SnatPolicy) (string, aciv1.PortRange, bool) {
	log.Info("Get Port Range For", "Node name: ", NodeName)
	snatPortsAllocated := snatpolicy.Status.SnatPortsAllocated
	snatIps := ExpandCIDRs(snatpolicy.Spec.SnatIp)
	portRange, portsPerNode := getPortRangeFromConfigMap(c)
	var currPortRange []aciv1.PortRange
	currPortRange = append(currPortRange, portRange)
	expandedsnatports := ExpandPortRanges(currPortRange, portsPerNode)
	if len(snatPortsAllocated) == 0 {
		return snatIps[0], expandedsnatports[0], false
	}
	for _, v := range snatIps {
		if _, ok := snatPortsAllocated[v]; ok {
			//  Check ports for this IP exhaused, then check for next IP
			if len(snatPortsAllocated[v]) <= len(expandedsnatports) {
				for _, val := range snatPortsAllocated[v] {
					if val.NodeName == NodeName {
						return v, val.PortRange, true
					}
				}
				m := map[int]int{}
				for _, Val1 := range snatPortsAllocated[v] {
					m[Val1.PortRange.Start] = Val1.PortRange.End
				}
				for i, Val2 := range expandedsnatports {
					if _, ok := m[Val2.Start]; !ok {
						log.Info("Created New Port Range for new NodeName ", "SnatGlobalInfo", expandedsnatports[i])
						return v, expandedsnatports[i], false
					}
				}
			}
			continue
		}
		return v, expandedsnatports[0], false
	}
	return "", aciv1.PortRange{}, false
}

// Update the SnatPolicy Status when the there is no localinfo present for snatIp
func UpdateSnatPolicyStatus(NodeName string, snatpolicy *aciv1.SnatPolicy,
	snatIp string, c client.Client) bool {
	if _, ok := snatpolicy.Status.SnatPortsAllocated[snatIp]; ok {
		nodePortRange := snatpolicy.Status.SnatPortsAllocated[snatIp][:]
		for i, val := range nodePortRange {
			if NodeName == val.NodeName {
				nodePortRange[i] = nodePortRange[len(nodePortRange)-1]
				nodePortRange = nodePortRange[:len(nodePortRange)-1]
				snatpolicy.Status.SnatPortsAllocated[snatIp] = nodePortRange
				if len(nodePortRange) == 0 {
					delete(snatpolicy.Status.SnatPortsAllocated, snatIp)
				}
				return true
			}
		}
		return true
	}
	return false
}

// Get the Portrange for the Node
func GetPortRangeForServiceIP(c client.Client, NodeName string,
	snatpolicy *aciv1.SnatPolicy, snatIp string) (aciv1.PortRange, bool) {
	snatPortsAllocated := snatpolicy.Status.SnatPortsAllocated
	portRange, portsPerNode := getPortRangeFromConfigMap(c)
	var currPortRange []aciv1.PortRange
	currPortRange = append(currPortRange, portRange)
	expandedsnatports := ExpandPortRanges(currPortRange, portsPerNode)
	if _, ok := snatPortsAllocated[snatIp]; ok {
		nodePortRange := snatPortsAllocated[snatIp]
		if len(nodePortRange) == 0 {
			return expandedsnatports[0], false
		}
		for _, val := range nodePortRange {
			if val.NodeName == NodeName {
				return val.PortRange, true
			}
		}
		m := map[int]int{}
		for _, Val1 := range snatPortsAllocated[snatIp] {
			m[Val1.PortRange.Start] = Val1.PortRange.End
		}
		for i, Val2 := range expandedsnatports {
			if _, ok := m[Val2.Start]; !ok {
				log.Info("Created New Port Range for new NodeName ", "SnatGlobalInfo", expandedsnatports[i])
				return expandedsnatports[i], false
			}
		}
		return aciv1.PortRange{}, false
	}
	return expandedsnatports[0], false
}

// Check target labels matches any snatpolicy labels
func CheckMatchesLabletoPolicy(c client.Client, labels map[string]string,
	namespace string) (string, bool) {
	matches := false
	var snatPolicyName string
	snatPolicyList := &aciv1.SnatPolicyList{}
	if err := c.List(context.TODO(), &client.ListOptions{Namespace: ""}, snatPolicyList); err != nil {
		return snatPolicyName, matches
	}
	for _, item := range snatPolicyList.Items {
		if item.Status.State != aciv1.Ready {
			continue
		}
		if MatchLabels(item.Spec.Selector.Labels, labels) {
			log.Info("Matches", "Labels ", item.Spec.Selector.Labels)
			if item.Spec.Selector.Namespace != "" && item.Spec.Selector.Namespace == namespace {
				matches = true
				snatPolicyName = item.ObjectMeta.Name
				break
			} else if item.Spec.Selector.Namespace == "" {
				matches = true
				snatPolicyName = item.ObjectMeta.Name
				break
			}
		}
	}
	return snatPolicyName, matches
}

// Allocate ip port range
func AllocateIpPortRange(c client.Client, portInuse map[string][]aciv1.NodePortRange,
	podList *corev1.PodList, snatPolicy *aciv1.SnatPolicy) bool {
	updated := false
	visited := make(map[string]bool)
	for _, pod := range podList.Items {
		if pod.Spec.HostNetwork {
			continue
		}
		if visited[pod.Spec.NodeName] {
			continue
		}
		snatip, portrange, exists := GetIPPortRangeForPod(c, pod.Spec.NodeName, snatPolicy)
		if reflect.DeepEqual(portrange, aciv1.PortRange{}) {
			return updated
		}
		if exists == false && pod.GetObjectMeta().GetDeletionTimestamp() == nil {
			var nodePortRnage aciv1.NodePortRange
			nodePortRnage.NodeName = pod.Spec.NodeName
			nodePortRnage.PortRange = portrange
			portInuse[snatip] = append(portInuse[snatip], nodePortRnage)
			updated = true
			visited[pod.Spec.NodeName] = true
		}
	}
	return updated
}

// Allocate ip port range for services
func AllocateIpPortRangeforservice(c client.Client, portInuse map[string][]aciv1.NodePortRange,
	podList *corev1.PodList, snatPolicy *aciv1.SnatPolicy, snatip string) bool {
	updated := false
	visited := make(map[string]bool)
	for _, pod := range podList.Items {
		if visited[pod.Spec.NodeName] {
			continue
		}
		portrange, exists := GetPortRangeForServiceIP(c, pod.Spec.NodeName, snatPolicy, snatip)
		if reflect.DeepEqual(portrange, aciv1.PortRange{}) {
			return updated
		}
		if exists == false && pod.GetObjectMeta().GetDeletionTimestamp() == nil {
			var nodePortRnage aciv1.NodePortRange
			nodePortRnage.NodeName = pod.Spec.NodeName
			nodePortRnage.PortRange = portrange
			portInuse[snatip] = append(portInuse[snatip], nodePortRnage)
			updated = true
			visited[pod.Spec.NodeName] = true
		}
	}
	return updated
}

// Get Policy matches the pod which is present it local Info
func GetSnatPolicyCRFromPod(c client.Client, pod *corev1.Pod) (aciv1.SnatPolicy, error) {
	foundSnatPolicy := &aciv1.SnatPolicy{}
	localInfo, err := GetLocalInfoCR(c, pod.Spec.NodeName, os.Getenv("ACI_SNAT_NAMESPACE"))
	if err == nil {
		if len(localInfo.Spec.LocalInfos) > 0 {
			if _, ok := localInfo.Spec.LocalInfos[string(pod.ObjectMeta.UID)]; ok {
				snatPolicyName := localInfo.Spec.LocalInfos[string(pod.ObjectMeta.UID)].SnatPolicyName
				*foundSnatPolicy, err = GetSnatPolicyCR(c, snatPolicyName)
				if err != nil && errors.IsNotFound(err) {
					log.Error(err, "not matching snatpolicy")
					return *foundSnatPolicy, nil
				} else if err != nil {
					return *foundSnatPolicy, err
				}
			}
		}
	}
	return *foundSnatPolicy, nil
}
