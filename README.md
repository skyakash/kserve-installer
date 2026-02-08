# KServe Installer

The **KServe Installer** is a Kubernetes operator designed to provide a "one-click" deployment experience for the complete KServe serving stack. It automates the installation and configuration of KServe and its core dependencies: **Istio**, **Knative**, and **Cert-Manager**.

## 🚀 Key Features
- **Deterministic Installation**: Bundles specific, tested versions of all dependencies.
- **Dependency Resolution**: Automatically manages OLM subscriptions for Cert-Manager, Istio (Sail Operator), and Knative.
- **Automatic Configuration**: Provisioning of namespaces (`istio-system`, `knative-serving`) and control plane resources via a single Custom Resource.
- **Production Ready**: Configures Gateway API support and standard Ingress Gateways out of the box.

## 📁 Project Structure
- `api/v1alpha1/`: Definition of the `KServeStack` API.
- `internal/controller/`: Reconciliation logic for bootstrapping the stack.
- `internal/controller/manifests/`: Embedded YAML manifests for all components.
- `config/`: Operator and RBAC configuration files.
- `bundle/`: OLM Bundle metadata for distribution.

## 📖 Documentation
Choose the guide that best fits your needs:

- **[Usage Guide](kserve-installer/usage_guide.md)**: Steps for end-users to install the stack on a fresh cluster using the OLM bundle.
- **[Packaging Guide](kserve-installer/packaging_guide.md)**: Developer guide for building, modifying, and publishing the operator.
- **[Manual Installation Guide](kserve-installer/manual_install_guide.md)**: Alternative steps for users who prefer manual `kubectl` / `helm` installation without the meta-operator.

## ⚡ Quick Start
If you already have `operator-sdk` and `OLM` installed, you can deploy the stack with:

```bash
# 1. Run the bundle
operator-sdk run bundle docker.io/akashneha/kserve-installer-bundle:v0.0.1

# 2. Grant Knative Operator permissions (Local Cluster/Docker Desktop fix)
kubectl create clusterrolebinding knative-operator-admin \
  --clusterrole=cluster-admin \
  --serviceaccount=default:knative-operator

# 3. Apply the stack configuration
kubectl apply -f kserve-installer/config/samples/kserve_v1alpha1_kservestack.yaml
```

## License
Copyright 2026. Licensed under the Apache License, Version 2.0.

