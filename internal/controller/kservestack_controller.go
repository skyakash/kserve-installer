/*
Copyright 2026.

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

package controller

import (
	"context"
	"embed"
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	kservev1alpha1 "github.com/akashdeo/kserve-installer/api/v1alpha1"
)

// KServeStackReconciler reconciles a KServeStack object
type KServeStackReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=kserve.kserve.example.com,resources=kservestacks,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=kserve.kserve.example.com,resources=kservestacks/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=kserve.kserve.example.com,resources=kservestacks/finalizers,verbs=update
//+kubebuilder:rbac:groups="*",resources="*",verbs="*"

//go:embed manifests/*.yaml
var content embed.FS

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *KServeStackReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	l := log.FromContext(ctx)

	// Fetch the KServeStack instance
	kserveStack := &kservev1alpha1.KServeStack{}
	if err := r.Get(ctx, req.NamespacedName, kserveStack); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Apply manifests in order
	manifests := []string{
		"manifests/namespaces.yaml",               // Create required namespaces
		"manifests/knative-rbac.yaml",             // Grant Knative Operator permissions
		"manifests/kserve.yaml",                   // Install KServe Control Plane (CRDs + Controller)
		"manifests/istio.yaml",                    // Configure Istio (via Sail Operator)
		"manifests/knative-serving.yaml",          // Configure Knative (via Knative Operator)
		"manifests/kserve-cluster-resources.yaml", // Install Default Runtimes
	}

	for _, m := range manifests {
		l.Info("Applying manifest", "file", m)
		if err := r.applyManifest(ctx, m); err != nil {
			l.Error(err, "Failed to apply manifest", "file", m)
			return ctrl.Result{}, err
		}
	}

	l.Info("All manifests applied successfully")
	return ctrl.Result{}, nil
}

func (r *KServeStackReconciler) applyManifest(ctx context.Context, fileName string) error {
	data, err := content.ReadFile(fileName)
	if err != nil {
		return fmt.Errorf("failed to read embedded file %s: %w", fileName, err)
	}

	// Split YAML documents (some files like kserve.yaml contain multiple docs)
	docs := strings.Split(string(data), "\n---")

	for _, doc := range docs {
		if strings.TrimSpace(doc) == "" {
			continue
		}

		// Decode YAML to Unstructured
		dec := yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)
		obj := &unstructured.Unstructured{}
		_, _, err := dec.Decode([]byte(doc), nil, obj)
		if err != nil {
			return fmt.Errorf("failed to decode YAML in %s: %w", fileName, err)
		}

		// Apply using Server-Side Apply
		// We set the field manager to "kserve-installer" to own the changes
		if err := r.Patch(ctx, obj, client.Apply, client.ForceOwnership, client.FieldOwner("kserve-installer")); err != nil {
			return fmt.Errorf("failed to apply object %s/%s: %w", obj.GetKind(), obj.GetName(), err)
		}
	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *KServeStackReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kservev1alpha1.KServeStack{}).
		Complete(r)
}
