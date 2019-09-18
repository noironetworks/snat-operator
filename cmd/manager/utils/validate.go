package utils

import (
	"context"
	"net"

	aciv1 "github.com/noironetworks/snat-operator/pkg/apis/aci/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Validator defines validator struct
type Validator struct {
	Validated    bool
	ErrorMessage string
}

// // Validate validates SnatSubnet Custom Resource
func (v *Validator) ValidateSnatIP(cr *aciv1.SnatPolicy, c client.Client) {
	v.Validated = true
	snatPolicyList := &aciv1.SnatPolicyList{}
	err := c.List(context.TODO(), &client.ListOptions{Namespace: ""}, snatPolicyList)
	if err != nil {
		UtilLog.Error(err, "failed to list existing Snatpolicies")
		v.Validated = false
	}

	if len(cr.Spec.Selector.Labels) > 1 {
		v.Validated = false
		UtilLog.Info("Invalid incoming Snatpolicy: cant have more than one label")
		return
	}

	if len(snatPolicyList.Items) > 1 {
		cr_labels := cr.Spec.Selector.Labels
		cr_ns := cr.Spec.Selector.Namespace
		for _, item := range snatPolicyList.Items {
			if item.Status.State != aciv1.Failed {
				if cr.ObjectMeta.Name != item.ObjectMeta.Name {

					// check if IP is repeated or invalid
					for _, val := range item.Spec.SnatIp {
						_, net1, err1 := net.ParseCIDR(val)
						if err1 != nil {
							ip_temp1 := net.ParseIP(val)
							if ip_temp1.To4() != nil {
								val = val + "/32"
								_, net1, _ = net.ParseCIDR(val)
							} else {
								val = val + "/128"
								_, net1, _ = net.ParseCIDR(val)
							}
						}
						for _, ip := range cr.Spec.SnatIp {
							_, net2, err := net.ParseCIDR(ip)
							if err != nil {
								ip_temp := net.ParseIP(ip)
								if ip_temp != nil && ip_temp.To4() != nil {
									ip = ip + "/32"
									_, net2, _ = net.ParseCIDR(ip)
								} else if ip_temp != nil && ip_temp.To16() != nil {
									ip = ip + "/128"
									_, net2, _ = net.ParseCIDR(ip)
								} else {
									UtilLog.Error(err, "Invalid incoming Snatpolicy")
									v.Validated = false
									return
								}
							}
							if net2.Contains(net1.IP) || net1.Contains(net2.IP) {
								UtilLog.Error(err, "SnatIP's are conflicting across the policies")
								v.Validated = false
								return
							}
						}
					}

					// check if labels are repeated
					item_labels := item.Spec.Selector.Labels
					for key, crLabel := range cr_labels {
						if _, ok := item_labels[key]; ok {
							if crLabel == item_labels[key] {
								UtilLog.Info("Label already exists")
								v.Validated = false
								return
							}
						}
					}

					// if no labels, diff IP and
					// same namespace- reject
					item_ns := item.Spec.Selector.Namespace
					if (len(item_labels) == 0) && (len(cr_labels) == 0) && (cr_ns == item_ns){
						v.Validated = false
						return
					}
				}
			}
		}
	} else {
		for _, ip := range cr.Spec.SnatIp {
			_, _, err := net.ParseCIDR(ip)
			if err != nil {
				ip_temp := net.ParseIP(ip)
				if ip_temp != nil && ip_temp.To4() != nil {
					ip = ip + "/32"
				} else if ip_temp != nil && ip_temp.To16() != nil {
					ip = ip + "/128"
				} else {
					v.Validated = false
					UtilLog.Error(err, "Invalid incoming Snatpolicy")
					return
				}
			}
		}
	}
}
