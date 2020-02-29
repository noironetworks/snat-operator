// Copyright 2019 Cisco Systems, Inc.
//
// Licensed under the Apache License, Version 2.0 (the License);
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an AS IS BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
// implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package utils

import (
	"context"

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
		UtilLog.Info("Invalid incoming snatpolicy can't have more than one label")
		return
	}

	if len(snatPolicyList.Items) > 1 {
		cr_labels := cr.Spec.Selector.Labels
		cr_ns := cr.Spec.Selector.Namespace
		for _, item := range snatPolicyList.Items {
			if item.Status.State != aciv1.Failed {
				if cr.ObjectMeta.Name != item.ObjectMeta.Name {
					for _, val := range item.Spec.SnatIp {
						_, net1, _ := parseIP(val)
						for _, ip := range cr.Spec.SnatIp {
							_, net2, err := parseIP(ip)
							if err != nil {
								UtilLog.Error(err, "Invalid incoming Snatpolicy")
								v.Validated = false
								return
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
			_, _, err := parseIP(ip)
			if err != nil {
				UtilLog.Error(err, "Invalid incoming Snatpolicy")
				v.Validated = false
				return
			}
		}
	}
}
