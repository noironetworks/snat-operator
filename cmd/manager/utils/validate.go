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
		UtilLog.Error(err, "failed to list existing Snatpoliceis")
		v.Validated = false
	}
	for _, item := range snatPolicyList.Items {
		if cr.ObjectMeta.Name != item.ObjectMeta.Name {
			for _, val := range item.Spec.SnatIp {
				_, net1, _ := net.ParseCIDR(val)
				for _, ip := range cr.Spec.SnatIp {
					_, net2, err := net.ParseCIDR(ip)
					if err != nil {
						UtilLog.Error(err, "failed to list existing Snatpoliceis")
						v.Validated = false
					}
					if net2.Contains(net1.IP) || net1.Contains(net2.IP) {
						UtilLog.Error(err, "SnatIP's are conflicting across the policies")
						v.Validated = false
					}
				}
			}
		}
	}

}
