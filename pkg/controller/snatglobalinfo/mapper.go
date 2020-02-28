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
package snatglobalinfo

import (
	aciv1 "github.com/noironetworks/snat-operator/pkg/apis/aci/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var MapperLog = logf.Log.WithName("Mapper:")

type handleLocalInfosMapper struct {
	client     client.Client
	predicates []predicate.Predicate
}

// Globalinfo reconcile loop has secondary resource as LocalInfo.
// When Locainfo change happens, we are pusing request to reconcile loop of GlobalInfo CR
// Request will be of this format
// Name: "snat-localinfo-" + <locainfo CR name>
func (h *handleLocalInfosMapper) Map(obj handler.MapObject) []reconcile.Request {
	MapperLog.V(1).Info("Mapper handling the object", "SnatLocalInfo: ", obj.Object)
	if obj.Object == nil {
		return nil
	}

	localInfo, ok := obj.Object.(*aciv1.SnatLocalInfo)
	MapperLog.V(1).Info("Mapper handling the object", "SnatLocalInfo: ", localInfo)
	if !ok {
		return nil
	}

	var requests []reconcile.Request
	requests = append(requests, reconcile.Request{
		NamespacedName: types.NamespacedName{
			Namespace: localInfo.ObjectMeta.Namespace,
			Name:      "snat-localinfo$" + localInfo.ObjectMeta.Name,
		},
	})

	return requests
}

func HandleLocalInfosMapper(client client.Client, predicates []predicate.Predicate) handler.Mapper {
	return &handleLocalInfosMapper{client, predicates}
}
