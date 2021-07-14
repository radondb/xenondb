/*
Copyright 2021 zhyass.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"fmt"
	"reflect"

	"github.com/go-logr/logr"
	"github.com/presslabs/controller-util/syncer"
	apiv1alpha1 "github.com/radondb/radondb-mysql-kubernetes/api/v1alpha1"
	"github.com/radondb/radondb-mysql-kubernetes/mysqlbackup"
	backupSyncer "github.com/radondb/radondb-mysql-kubernetes/mysqlbackup/syncer"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// MysqlBackupReconciler reconciles a MysqlBackup object
type MysqlBackupReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
	//Opt      *options.Options
}

//+kubebuilder:rbac:groups=mysql.radondb.com,resources=mysqlbackups,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=serviceaccounts,verbs=get;list;watch;create;update
//+kubebuilder:rbac:groups=mysql.radondb.com,resources=mysqlbackups/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=mysql.radondb.com,resources=mysqlbackups/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the MysqlBackup object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.7.2/pkg/reconcile
func (r *MysqlBackupReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx).WithName("controllers").WithName("MysqlBackup")

	// your logic here
	// Fetch the MysqlBackup instance
	backup := mysqlbackup.New(&apiv1alpha1.MysqlBackup{})
	err := r.Get(context.TODO(), req.NamespacedName, backup.Unwrap())
	if err != nil {
		if errors.IsNotFound(err) {
			// Object not found, return.  Created objects are automatically garbage collected.
			// For additional cleanup logic use finalizers.
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}
	// Set defaults on backup
	r.Scheme.Default(backup.Unwrap())

	// save the backup for later check for diff
	savedBackup := backup.Unwrap().DeepCopy()
	log.Info(fmt.Sprintf("saveBackup %v", savedBackup))
	// cluster name should be specified for a backup
	// if len(backup.Spec.ClusterName) == 0 {
	// 	return reconcile.Result{}, fmt.Errorf("cluster name is not specified")
	// }
	//TODO:
	syncers := []syncer.Interface{
		backupSyncer.NewJobSyncer(r.Client, r.Scheme, backup),
	}

	if err = r.sync(context.TODO(), syncers); err != nil {
		return reconcile.Result{}, err
	}
	if err = r.updateBackup(savedBackup, backup); err != nil {
		return reconcile.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *MysqlBackupReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&apiv1alpha1.MysqlBackup{}).
		Complete(r)
}
func (r *MysqlBackupReconciler) updateBackup(savedBackup *apiv1alpha1.MysqlBackup, backup *mysqlbackup.MysqlBackup) error {
	log := log.Log.WithName("controllers").WithName("MysqlBackup")
	if !reflect.DeepEqual(savedBackup, backup.Unwrap()) {
		if err := r.Update(context.TODO(), backup.Unwrap()); err != nil {
			return err
		}
	}
	if !reflect.DeepEqual(savedBackup.Status, backup.Unwrap().Status) {

		log.Info("update backup object status")
		if err := r.Status().Update(context.TODO(), backup.Unwrap()); err != nil {
			log.Error(err, fmt.Sprintf("update status backup %s/%s", backup.Name, backup.Namespace),
				"backupStatus", backup.Status)
			return err
		}
	}
	return nil
}
func (r *MysqlBackupReconciler) sync(ctx context.Context, syncers []syncer.Interface) error {
	for _, s := range syncers {
		if err := syncer.Sync(ctx, s, r.Recorder); err != nil {
			return err
		}
	}
	return nil
}