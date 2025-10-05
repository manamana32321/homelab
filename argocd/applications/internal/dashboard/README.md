# Kubernetes Dashboard

The Kubernetes Dashboard is a web-based user interface for managing Kubernetes clusters. This Helm chart deploys the official Kubernetes Dashboard with custom configurations to enhance security and usability.

## How to Install

1. Add the Helm repository and update it:

```sh
helm repo add kubernetes-dashboard https://kubernetes.github.io/dashboard/
helm repo update
```

2. Install the Kubernetes Dashboard with custom values:

```sh
helm install kubernetes-dashboard kubernetes-dashboard/kubernetes-dashboard -f values.yaml -n kubernetes-dashboard
```
