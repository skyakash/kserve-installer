# KServe Installer Operator Guide

This guide details how to build, package, and use the custom `kserve-installer` operator to deploy the KServe stack (Istio, Knative, Cert-Manager, KServe) via OLM.

## 1. Prerequisites
- `docker` (or podman)
- `make`
- A container registry you can push to (e.g., Docker Hub: `docker.io/<username>`).

## 2. Generate and Build
From the `kserve-installer` directory:

```bash
# Set your image registry (REPLACE THIS)
export IMG_REPO="docker.io/your-username"
export VERSION="v0.0.1"

# 1. Build and Push the Operator Image
make docker-build docker-push IMG=$IMG_REPO/kserve-installer:$VERSION

# 2. Build and Push the Bundle Image (OLM Artifact)
make bundle docker-build bundle-build bundle-push \
  IMG=$IMG_REPO/kserve-installer:$VERSION \
  BUNDLE_IMG=$IMG_REPO/kserve-installer-bundle:$VERSION
```

## 3. Local Development Loop (No Image Build)

For rapid iteration without building/pushing images, you can run the controller process locally against your current cluster context.

**Wait**: Ensure your cluster has OLM and dependencies installed first (or rely on the code to attempt partial installs, though CRD prerequisites must be present).
*Actually, for dependencies managed by OLM, running locally won't install them automatically unless you manually apply the Subscriptions. This is a limitation of local execution.*

**Recommended Command:**
```bash
make install run
```

This installs the CRDs and runs the controller binary on your machine.

## 4. Test OLM Bundle (Simulation)

To test the full OLM experience (including dependency resolution) without submitting to a public catalog, you can "host" the bundle yourself in any container registry (Docker Hub, Quay, or ttl.sh for ephemeral testing).

### Option A: Using ttl.sh (Ephemeral / No Login)
Great for quick, disposable tests. Images last 1 hour.

```bash
# 1. Build and Push (Anonymous)
export USER_ID=$(uuidgen | tr '[:upper:]' '[:lower:]')
export IMG=ttl.sh/${USER_ID}/kserve-installer:1h
export BUNDLE_IMG=ttl.sh/${USER_ID}/kserve-installer-bundle:1h

make docker-build docker-push IMG=$IMG
make bundle docker-build bundle-build bundle-push IMG=$IMG BUNDLE_IMG=$BUNDLE_IMG

# 2. Run Bundle
operator-sdk run bundle $BUNDLE_IMG
```

### Option B: Using Docker Hub (Personal)
Requires `docker login`.

```bash
export IMG=docker.io/<your-username>/kserve-installer:v0.0.1
export BUNDLE_IMG=docker.io/<your-username>/kserve-installer-bundle:v0.0.1

# Build & Push
make docker-build docker-push IMG=$IMG
make bundle docker-build bundle-build bundle-push IMG=$IMG BUNDLE_IMG=$BUNDLE_IMG

# Run Bundle
operator-sdk run bundle $BUNDLE_IMG
```

## 5. Trigger Installation
Once the operator is installed and running, create an instance of `KServeStack` to trigger the actual component configuration (applying the detailed YAMLs).

```yaml
apiVersion: kserve.kserve.example.com/v1alpha1
kind: KServeStack
metadata:
  name: kservestack-sample
spec:
  # Add spec fields if we parameterized anything in the future
```

Apply this CR:
```bash
kubectl apply -f config/samples/kserve_v1alpha1_kservestack.yaml
```

The operator will then:
1.  Install Istio (via Sail Operator).
2.  Install Knative Serving (via Knative Operator).
3.  Install KServe Control Plane.
4.  Install Default Serving Runtimes.

## 7. Manual Build & Publish (Deep Dive)

If you prefer to understand exactly what `make` is doing, or need to run these steps manually (e.g. in a CI/CD pipeline without Make), here is the step-by-step breakdown.

### Step 0: Set Variables
Define your image names once to re-use them.
```bash
export IMG=docker.io/akashneha/kserve-installer:v0.0.1
export BUNDLE_IMG=docker.io/akashneha/kserve-installer-bundle:v0.0.1
```

### Step 1: Docker Login
Authenticate to your registry so you can push images.
```bash
docker login
# Enter username (akashneha) and password
```

### Step 2: Build and Push the Manager Image
The "Manager" is the Go binary (the controller) that runs in the cluster and executes your logic.
```bash
# 1. Compile and Build Docker Image
docker build -t $IMG .

# 2. Push to Registry
docker push $IMG
```

### Step 3: Generate Bundle Manifests
The "Bundle" is a collection of YAMLs (CRDs, CSV, RBAC) that OLM uses to install your operator. This step generates those YAMLs in the `bundle/` directory.
`kustomize` is used to template the YAMLs, and `operator-sdk` wraps it all.

```bash
# 1. Generate core manifests (config/crd, config/rbac)
operator-sdk generate kustomize manifests -q

# 2. Set the controller image in the Kustomize config
cd config/manager && kustomize edit set image controller=$IMG && cd ../..

# 3. Build the final manifest and generate the bundle metadata
kustomize build config/manifests | operator-sdk generate bundle \
  -q --overwrite --version 0.0.1 --channels alpha --default-channel alpha
```

*Note: This updates the `bundle/manifests` and `bundle/metadata` directories.*

### Step 4: Build and Push the Bundle Image
The bundle itself must be packaged as a Docker image so OLM can pull it.
```bash
# 1. Build Bundle Image (using the generated Dockerfile)
docker build -f bundle.Dockerfile -t $BUNDLE_IMG .

# 2. Push Bundle Image
docker push $BUNDLE_IMG
```

### Step 5: Validate
Check that your bundle is valid.
```bash
operator-sdk bundle validate ./bundle
```

### Step 6: Install OLM (Prerequisite)

If you just reset your cluster, you must re-install OLM first.

```bash
curl -L https://github.com/operator-framework/operator-lifecycle-manager/releases/download/v0.28.0/install.sh | bash -s v0.28.0
```

### Step 7: Install (Run)
```bash
# 1. Run the bundle
operator-sdk run bundle $BUNDLE_IMG

# 2. Grant Knative Operator Permissions (CRITICAL for local clusters)
kubectl create clusterrolebinding knative-operator-admin \
  --clusterrole=cluster-admin \
  --serviceaccount=default:knative-operator
```

---

## 8. Best Practices & Troubleshooting

### Bundled Namespaces
The operator currently manages the creation of `istio-system`, `knative-serving`, and `cert-manager`. If you modify the manifests to add more namespaces, ensure they are added to `internal/controller/manifests/namespaces.yaml` and included in the `manifests` slice in `kservestack_controller.go`.

### Bundle Image Push (Mac/Apple Silicon)
When pushing bundle images from Apple Silicon, OLM may fail to pull them if `docker buildx` adds provenance attestations. Always use:
```bash
docker build --provenance=false -f bundle.Dockerfile -t $BUNDLE_IMG .
```

### Manifest Ordering
The reconciliation loop in `kservestack_controller.go` is order-dependent. Ensure namespaces and CRDs are applied before the Custom Resources (CRs) that depend on them.
