## How to setup Argo CD to a new cluster

### 1. Argo CD

```
kubectl create namespace argocd
kubectl apply -n argocd -f https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml
```

### 2. Argo CD Image Updater

```
kubectl apply -n argocd -f https://raw.githubusercontent.com/argoproj-labs/argocd-image-updater/stable/manifests/install.yaml
```

### 3. Setup Ingress

To access Argo CD via ingress, internal argocd-server should be edited to allow insecure connections.

```
kubectl -n argocd edit deployment argocd-server
```

```yaml
args:
- /usr/local/bin/argocd-server
- --insecure  # Add this line
```

and then apply the ingress manifest:

```
kubectl apply -f ingress.yaml
```

### 4. Login (Authentik OIDC)

Login is declared in [values.yaml](values.yaml) (`configs.cm.oidc.config`, `configs.rbac`) and
[blueprint-argocd.yaml](../authentik/manifests/blueprint-argocd.yaml). Only the `authentik Admins`
group can log in, and it maps to `role:admin`. Anyone else gets no permissions (`policy.default: ""`).

Lockout recovery: edit `argocd-rbac-cm` / `argocd-cm` with kubectl after turning off auto-sync of the `argocd` app.

### 5. AI account token

`claude` is a local account (`accounts.claude: apiKey`) with `role:readonly` + application sync.
The account and its permissions are declared, but the token is signed by the Argo CD server, so it is issued once by hand
(no expiry; re-issue if `argocd-secret`'s `server.secretkey` changes):
UI → Settings → Accounts → `claude` → Tokens → Generate New.
There is no OIDC client for `argocd login --sso`, so humans use the UI.

Put the token in each worktree's `.envrc.local` (amang, homelab):

```bash
export ARGOCD_SERVER=argocd.json-server.win
export ARGOCD_AUTH_TOKEN=<token>
export ARGOCD_OPTS=--grpc-web
```

## How to test a change

By default, applications/argocd.yaml tries to sync all applications to main branch's yaml manifests.

To test a change, you should disable the auto-sync feature of the application, and then `kubectl apply -f` your modified yaml manifests to the cluster.

```yaml
# applications/argocd.yaml

spec:
  syncPolicy: null  # Disable auto-sync
    # prune: true
    # selfHeal: true
```
