# KServe Installer: Usage Guide

This guide describes how to install the full KServe stack (Istio, Knative, KServe, Cert-Manager) using the `kserve-installer` operator bundle.

**Prerequisites:**
- A Kubernetes cluster (freshly installed/reset recommended)
- `curl`
- `operator-sdk` (binary provided in `bin/` or installed system-wide)
- `helm` (for Ingress Gateway)
- `kubectl`

---

## 1. Prerequisites (Install Tools)

### Install `operator-sdk`
You need the Operator SDK CLI to run the bundle.

**Option A: Homebrew (macOS)**
```bash
brew install operator-sdk
```

**Option B: Direct Download (Linux/macOS)**
```bash
export ARCH=$(case $(uname -m) in x86_64) echo -n amd64 ;; aarch64) echo -n arm64 ;; *) echo -n $(uname -m) ;; esac)
export OS=$(uname | awk '{print tolower($0)}')
export OPERATOR_SDK_DL_URL=https://github.com/operator-framework/operator-sdk/releases/download/v1.31.0
curl -LO ${OPERATOR_SDK_DL_URL}/operator-sdk_${OS}_${ARCH}
chmod +x operator-sdk_${OS}_${ARCH} && sudo mv operator-sdk_${OS}_${ARCH} /usr/local/bin/operator-sdk
```

**Option C: Use Local Tools (Pre-installed in project)**
If you are inside the `samplekserve` directory, you can add the pre-downloaded tools to your session:
```bash
export PATH=$PWD/.tools/go/bin:$PWD/.tools:$PATH
operator-sdk version
```

---

## 2. Install OLM (Operator Lifecycle Manager)
If your cluster does not have OLM installed (e.g. a fresh Kind or Docker Desktop cluster), install it first.

```bash
curl -L https://github.com/operator-framework/operator-lifecycle-manager/releases/download/v0.28.0/install.sh | bash -s v0.28.0
```

**Verification:**
Wait for the OLM pods to be ready:
```bash
kubectl get deployment -n olm
# Ensure olm-operator, catalog-operator, and packageserver are Ready (1/1)
```

## 3. Install the KServe Installer Bundle
Run the following command to install the operator. This will also automatically resolve and install the required dependencies (Istio, Knative, Cert-Manager).

```bash
# Set PATH for local tools if needed:
export PATH=$PWD/.tools/go/bin:$PWD/.tools:$PATH

# Run the bundle
operator-sdk run bundle docker.io/akashneha/kserve-installer-bundle:v0.0.1 --timeout 10m
```

### 3.5 Grant Knative Operator Permissions (Required)
In local environments (Docker Desktop, Kind), the Knative Operator requires an explicit ClusterRoleBinding to manage its components.

```bash
kubectl create clusterrolebinding knative-operator-admin \
  --clusterrole=cluster-admin \
  --serviceaccount=default:knative-operator
```

---

**Verification:**
```bash
kubectl get csv -n default
# Ensure 'kserve-installer', 'sailoperator', 'knative-operator', and 'cert-manager' show phase: Succeeded
```

## 4. Trigger Stack Configuration
Create the `KServeStack` custom resource. This signals the operator to configure the control planes.

```bash
kubectl apply -f kserve-installer/config/samples/kserve_v1alpha1_kservestack.yaml
```

**Monitor Progress:**
```bash
kubectl get pods -n kserve -w
# Wait for controller-manager pods to be Running
```

## 5. Install Istio Ingress Gateway
The operator configures the control plane, but for Gateway API support, we install the standard Istio Ingress Gateway via Helm. We install it in the `istio-system` namespace so Knative can automatically find it.

```bash
helm upgrade --install istio-ingressgateway istio/gateway -n istio-system --create-namespace
```

## 6. Verify Installation (Sample Inference)
Deploy a sample `sklearn` model to verify end-to-end functionality.

```bash
# 1. Create a namespace for the test
kubectl create ns kserve-test

# 2. Deploy the sample InferenceService
kubectl apply -f kserve-installer/iris.yaml -n kserve-test

# 3. Port-forward the Ingress Gateway (Use 8081 to avoid conflicts)
kubectl port-forward -n istio-system svc/istio-ingressgateway 8081:80 &

# 4. Test Inference (using the predictor-specific host)
curl -v -H "Host: sklearn-iris-predictor.kserve-test.example.com" \
     -H "Content-Type: application/json" \
     http://localhost:8081/v1/models/sklearn-iris:predict \
     -d @kserve-installer/iris-input.json

# Expected Output: {"predictions":[1,1]}
```

---

## 7. Cleanup / Reset
If you need to uninstall everything and start over:

```bash
# 1. Delete the sample and namespace
kubectl delete ns kserve-test

# 2. Uninstall the operator and dependencies
operator-sdk cleanup kserve-installer

# 3. Remove the Ingress Gateway
helm uninstall istio-ingressgateway -n istio-system

# 4. (Optional) Wipe OLM for a total reset
curl -L https://github.com/operator-framework/operator-lifecycle-manager/releases/download/v0.28.0/install.sh | bash -s v0.28.0 -- --uninstall
```
