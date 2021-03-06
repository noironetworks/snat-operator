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
package snatpolicy

import (
	"context"
	"os"
	"time"

	"github.com/go-logr/logr"
	"github.com/noironetworks/snat-operator/cmd/manager/utils"
	aciv1 "github.com/noironetworks/snat-operator/pkg/apis/aci/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const snatPolicyFinalizer = "finalizer.snatpolicy.aci.snat"

var log = logf.Log.WithName("controller_snatpolicy")

// Add creates a new SnatPolicy Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileSnatPolicy{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("snatpolicy-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource SnatPolicy
	err = c.Watch(&source.Kind{Type: &aciv1.SnatPolicy{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileSnatPolicy implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileSnatPolicy{}

// ReconcileSnatPolicy reconciles a SnatPolicy object
type ReconcileSnatPolicy struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a SnatPolicy object and makes changes based on the state read
// and what is in the SnatPolicy.Spec
func (r *ReconcileSnatPolicy) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling SnatPolicy")

	// Fetch the SnatPolicy instance
	instance := &aciv1.SnatPolicy{}
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

	// Check if the snatpolicy cr was marked to be deleted
	isSnatPolicyToBeDeleted := instance.GetDeletionTimestamp() != nil
	if isSnatPolicyToBeDeleted {
		if utils.Contains(instance.GetFinalizers(), snatPolicyFinalizer) {
			// Run finalization logic for snatPolicyFinalizer. If the
			// finalization logic fails, don't remove the finalizer so
			// that we can retry during the next reconciliation.

			if err := r.finalizeSnatPolicy(reqLogger, instance); err != nil {
				return reconcile.Result{}, err
			}

			// Remove snatPolicyFinalizer. Once all finalizers have been
			// removed, the object will be deleted.
			instance.SetFinalizers(utils.Remove(instance.GetFinalizers(), snatPolicyFinalizer))
			err := r.client.Update(context.TODO(), instance)
			if err != nil {
				return reconcile.Result{}, err
			}
		}
		return reconcile.Result{}, nil
	}

	validator := utils.Validator{}
	validator.ValidateSnatIP(instance, r.client)
	if !validator.Validated {
		instance.Status.State = aciv1.Failed
		reqLogger.Info("Policy failed")
		r.client.Status().Update(context.TODO(), instance)
		return reconcile.Result{}, err
	}
	if instance.Status.State != aciv1.IpPortsExhausted {
		instance.Status.State = aciv1.Ready
		err = r.client.Status().Update(context.TODO(), instance)
		if err != nil {
			return reconcile.Result{}, err
		}
	}
	log.Info("Policy successfully applied", "SnatPolicy:", request.Name)

	// Add finalizer for this CR
	if !utils.Contains(instance.GetFinalizers(), snatPolicyFinalizer) {
		if err := r.addFinalizer(instance); err != nil {
			return reconcile.Result{}, err
		}
	}
	// If snatIP resource is using any of the IP in snatSubnet, then check that and send appropriate error
	// return r.handleSnatSubnetUpdate(*instance)
	return reconcile.Result{}, nil
}

// Add finalizer string to snatpolicy resource to run cleanup logic on delete
func (r *ReconcileSnatPolicy) addFinalizer(m *aciv1.SnatPolicy) error {
	log.Info("Adding Finalizer for the SnatPolicy")
	m.SetFinalizers(append(m.GetFinalizers(), snatPolicyFinalizer))

	// Update CR
	err := r.client.Update(context.TODO(), m)
	if err != nil {
		log.Error(err, "Failed to update SnatPolicy with finalizer")
		return err
	}
	return nil
}

// Cleanup steps to be done when snatPolicy resource is getting deleted.
func (r *ReconcileSnatPolicy) finalizeSnatPolicy(reqLogger logr.Logger, m *aciv1.SnatPolicy) error {
	if len(m.Status.SnatPortsAllocated) != 0 {
		//Finlizer will wait untill the snatlocalinfo controller deletes the policy related info.
		// If it is still not deleted finalizer will check for snatpolicy refrence and deletes it after 2 secs.
		// Error condtion it retries every 2 sec.
		time.Sleep(time.Second * 2)
		for _, portslist := range m.Status.SnatPortsAllocated {
			for _, nodeinfo := range portslist {
				localInfo, _ := utils.GetLocalInfoCR(r.client, nodeinfo.NodeName, os.Getenv("ACI_SNAT_NAMESPACE"))
				update := false
				for uid, local := range localInfo.Spec.LocalInfos {
					if local.SnatPolicyName == m.ObjectMeta.Name {
						delete(localInfo.Spec.LocalInfos, uid)
						update = true
					}
				}
				if update {
					_, err := utils.UpdateLocalInfoCR(r.client, localInfo)
					if err != nil {
						reqLogger.Error(err, "Could not update localinfo for", "node Name", nodeinfo.NodeName)
						return err
					}
				}
			}
		}
	}
	return nil
}
