package utils

import (
	"context"
	"os"
	"strings"

	// nodeinfo "github.com/noironetworks/aci-containers/pkg/nodeinfo/apis/aci.nodeinfo/v1"

	aciv1 "github.com/noironetworks/snat-operator/pkg/apis/aci/v1"
	"github.com/prometheus/common/log"
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
func GetPodNameFromReoncileRequest(requestName string) (string, string) {

	temp := strings.Split(requestName, "-")
	if len(temp) != 4 {
		UtilLog.Info("Length should be 4", "input string:", requestName, "lengthGot", len(temp))
		return "", ""
	}
	snatPolicyName, podName := temp[2], temp[3]
	return podName, snatPolicyName
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
			UtilLog.Info("Nodeinfo object found", "For NodeName:", item.ObjectMeta.Name)
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
		log.Info("LocalIfo not present ", "foundLocalIfo:", nodeName)
		return aciv1.SnatLocalInfo{}, nil
	} else if err != nil {
		return aciv1.SnatLocalInfo{}, err
	}

	return *foundLocalIfo, nil
}

// Given a policyName, return SnatPolicy CR object if present
func GetSnatPolicyCR(c client.Client, policyName string) (aciv1.SnatPolicy, error) {

	foundSnatPolicy := &aciv1.SnatPolicy{}
	err := c.Get(context.TODO(), types.NamespacedName{Name: policyName, Namespace: os.Getenv("ACI_SNAT_NAMESPACE")}, foundSnatPolicy)
	if err != nil && errors.IsNotFound(err) {
		log.Info("SnatPolicy not present", "foundSnatPolicy:", policyName)
		return aciv1.SnatPolicy{}, err
	} else if err != nil {
		return aciv1.SnatPolicy{}, err
	}

	return *foundSnatPolicy, nil
}

// createSnatLocalInfoCR Creates a SnatLocalInfo CR
func CreateLocalInfoCR(c client.Client, localInfoSpec aciv1.SnatLocalInfoSpec, nodeName string) (reconcile.Result, error) {

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
		return reconcile.Result{}, err
	}
	log.Info("Created localinfo object", "SnatLocalInfo", obj)
	return reconcile.Result{}, nil
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
func GetIPPortRangeForPod(NodeName string, snatPolicyName string, c client.Client) (string, aciv1.PortRange, bool, error) {
	log.Info("SnatPolicy Info", "Snatpolicy Name", snatPolicyName)
	foundSnatPolicy, err := GetSnatPolicyCR(c, snatPolicyName)
	if err != nil {
		log.Error(err, "not matching snatpolicy", snatPolicyName)
		return "", aciv1.PortRange{}, false, err
	}
	snatPortsAllocated := foundSnatPolicy.Status.SnatPortsAllocated
	snatIps := ExpandCIDRs(foundSnatPolicy.Spec.SnatIp)
	var portRange aciv1.PortRange
	portRange.Start = MIN_PORT
	portRange.End = MAX_PORT
	var currPortRange []aciv1.PortRange
	currPortRange = append(currPortRange, portRange)
	expandedsnatports := ExpandPortRanges(currPortRange, PORTPERNODES)
	if len(snatPortsAllocated) == 0 {
		return snatIps[0], expandedsnatports[0], false, nil
	}
	for _, v := range snatIps {
		if _, ok := snatPortsAllocated[v]; ok {
			//  Check ports for this IP exhaused, then check for next IP
			if len(snatPortsAllocated[v]) < len(expandedsnatports) {
				for _, val := range snatPortsAllocated[v] {
					if val.NodeName == NodeName {
						return v, val.PortRange, true, nil
					}
				}
				m := map[int]int{}
				for _, Val1 := range snatPortsAllocated[v] {
					m[Val1.PortRange.Start] = Val1.PortRange.End
				}
				for i, Val2 := range expandedsnatports {
					if _, ok := m[Val2.Start]; !ok {
						var nodePortRange aciv1.NodePortRange
						nodePortRange.NodeName = NodeName
						nodePortRange.PortRange = expandedsnatports[i]
						snatPortsAllocated[v] = append(snatPortsAllocated[v], nodePortRange)
						return v, expandedsnatports[i], false, nil
					}
				}
			}
		}
	}
	return "", aciv1.PortRange{}, false, nil
}
func UpdateSnatPolicyStatus(NodeName string, snatPolicyName string, snatIp string, c client.Client) (reconcile.Result, error) {
	foundSnatPolicy, err := GetSnatPolicyCR(c, snatPolicyName)
	if err != nil {
		log.Error(err, "not matching snatpolicy", snatPolicyName)
		return reconcile.Result{}, nil
	}
	if _, ok := foundSnatPolicy.Status.SnatPortsAllocated[snatIp]; ok {
		nodePortRange := foundSnatPolicy.Status.SnatPortsAllocated[snatIp]
		for i, val := range nodePortRange {
			if val.NodeName == NodeName {
				nodePortRange[i] = nodePortRange[len(nodePortRange)-1]
				nodePortRange = nodePortRange[:len(nodePortRange)-1]
				break
			}
		}
		foundSnatPolicy.Status.SnatPortsAllocated[snatIp] = nodePortRange
		err = c.Status().Update(context.TODO(), &foundSnatPolicy)
		if err != nil {
			return reconcile.Result{}, err
		}
	}
	return reconcile.Result{}, nil
}
