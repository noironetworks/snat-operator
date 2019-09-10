package utils

import (
	"context"
	"os"
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

// Given a reconcile request name, it extracts out pod name by omiiting snat-policy- from it
// eg: snat-policy-foo-podname -> podname, foo
func GetPodNameFromReoncileRequest(requestName string) (string, string, string) {

	temp := strings.Split(requestName, "$")
	if len(temp) != 4 {
		UtilLog.Info("Length should be 4", "input string:", requestName, "lengthGot", len(temp))
		return "", "", ""
	}
	name, resName, resType := temp[1], temp[2], temp[3]
	return resName, name, resType
}

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
func CreateLocalInfoCR(c client.Client, localInfoSpec aciv1.SnatLocalInfoSpec, nodeName string) (*aciv1.SnatLocalInfo, reconcile.Result, error) {

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
func DeleteLocalInfoCR(c client.Client, localInfoSpec aciv1.SnatLocalInfoSpec, nodeName string) (reconcile.Result, error) {

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

// Get IP and port for pod for which notification has come to reconcile loop
func GetIPPortRangeForPod(NodeName string, snatpolicy *aciv1.SnatPolicy) (string, aciv1.PortRange, bool) {
	log.Info("Get Port Range For", "Node name: ", NodeName)
	snatPortsAllocated := snatpolicy.Status.SnatPortsAllocated
	snatIps := ExpandCIDRs(snatpolicy.Spec.SnatIp)
	var portRange aciv1.PortRange
	portRange.Start = MIN_PORT
	portRange.End = MAX_PORT
	var currPortRange []aciv1.PortRange
	currPortRange = append(currPortRange, portRange)
	expandedsnatports := ExpandPortRanges(currPortRange, PORTPERNODES)
	if len(snatPortsAllocated) == 0 {
		return snatIps[0], expandedsnatports[0], false
	}
	for _, v := range snatIps {
		if _, ok := snatPortsAllocated[v]; ok {
			//  Check ports for this IP exhaused, then check for next IP
			if len(snatPortsAllocated[v]) < len(expandedsnatports) {
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
		}
		return v, expandedsnatports[0], false
	}
	return "", aciv1.PortRange{}, false
}
func UpdateSnatPolicyStatus(NodeName string, snatpolicy *aciv1.SnatPolicy, snatIp string, c client.Client) bool {
	if snatpolicy.GetDeletionTimestamp() != nil {
		return false
	}
	if _, ok := snatpolicy.Status.SnatPortsAllocated[snatIp]; ok {
		nodePortRange := snatpolicy.Status.SnatPortsAllocated[snatIp]
		for i, val := range nodePortRange {
			if NodeName == val.NodeName {
				nodePortRange[i] = nodePortRange[len(nodePortRange)-1]
				nodePortRange = nodePortRange[:len(nodePortRange)-1]
				snatpolicy.Status.SnatPortsAllocated[snatIp] = nodePortRange
				return true
			}
		}
	}
	return false
}

func GetPortRangeForServiceIP(NodeName string, snatpolicy *aciv1.SnatPolicy, snatIp string) (aciv1.PortRange, bool) {
	snatPortsAllocated := snatpolicy.Status.SnatPortsAllocated
	var portRange aciv1.PortRange
	portRange.Start = MIN_PORT
	portRange.End = MAX_PORT
	var currPortRange []aciv1.PortRange
	currPortRange = append(currPortRange, portRange)
	expandedsnatports := ExpandPortRanges(currPortRange, PORTPERNODES)
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
func CheckMatchesLabletoPolicy(snatPolicyList *aciv1.SnatPolicyList, labels map[string]string, namespace string) (string, bool) {
	matches := false
	var snatPolicyName string
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
func AllocateIpPortRange(portInuse map[string][]aciv1.NodePortRange, podList *corev1.PodList,
	snatPolicy *aciv1.SnatPolicy) bool {
	updated := false
	emptyportrange := aciv1.PortRange{}
	for _, pod := range podList.Items {
		if pod.Spec.HostNetwork {
			continue
		}
		snatip, portrange, exists := GetIPPortRangeForPod(pod.Spec.NodeName, snatPolicy)
		if exists == false && pod.GetObjectMeta().GetDeletionTimestamp() == nil &&
			portrange != emptyportrange {
			var nodePortRnage aciv1.NodePortRange
			nodePortRnage.NodeName = pod.Spec.NodeName
			nodePortRnage.PortRange = portrange
			portInuse[snatip] = append(portInuse[snatip], nodePortRnage)
			updated = true
		}
	}
	return updated
}

// Allocate ip port range for services
func AllocateIpPortRangeforservice(portInuse map[string][]aciv1.NodePortRange, podList *corev1.PodList,
	snatPolicy *aciv1.SnatPolicy, snatip string) bool {
	updated := false
	for _, pod := range podList.Items {
		portrange, exists := GetPortRangeForServiceIP(pod.Spec.NodeName, snatPolicy, snatip)
		if exists == false && pod.GetObjectMeta().GetDeletionTimestamp() == nil {
			var nodePortRnage aciv1.NodePortRange
			nodePortRnage.NodeName = pod.Spec.NodeName
			nodePortRnage.PortRange = portrange
			portInuse[snatip] = append(portInuse[snatip], nodePortRnage)
			updated = true
		}
	}
	return updated
}
