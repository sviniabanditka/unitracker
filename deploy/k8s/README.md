# Tracker — k3s deployment

Deploys to the same cluster as `scheduler.sviniabanditka.com` etc.: traefik ingress + cert-manager (`letsencrypt-prod`) + private registry `registry.sviniabanditka.com`.

DNS A record for `tracker.sviniabanditka.com` must already point at the cluster.

## Layout

| File | Purpose |
|---|---|
| `namespace.yaml` | namespace `tracker` |
| `pvc.yaml` | 5Gi RWO PVC for `/data` (SQLite + snapshots). Single-replica only. |
| `secret.example.yaml` | template; copy to `secret.yaml`, fill in, apply (gitignored) |
| `deployment.yaml` | Service (ClusterIP :80 → :8080) + Deployment (1 replica, Recreate strategy) |
| `ingress.yaml` | traefik ingress + TLS via cert-manager |
| `kustomization.yaml` | aggregates the public manifests |

## First-time deploy

```bash
# 1. Build & push image
./deploy/build-push.sh

# 2. Create the namespace + PVC
kubectl apply -f deploy/k8s/namespace.yaml
kubectl apply -f deploy/k8s/pvc.yaml

# 3. Create the secret (one-shot; do not commit it)
cp deploy/k8s/secret.example.yaml deploy/k8s/secret.yaml
$EDITOR deploy/k8s/secret.yaml
kubectl apply -f deploy/k8s/secret.yaml

# 4. Deployment + Service + Ingress
kubectl apply -k deploy/k8s

# 5. Watch rollout
kubectl -n tracker rollout status deploy/tracker
kubectl -n tracker get ingress
```

cert-manager will issue a Let's Encrypt cert into the `tracker-tls` secret on first request to the ingress. The first HTTPS hit may take ~30s while the cert is being issued.

## Updates

Pushes to `master` (or `main`) trigger `.github/workflows/deploy.yml`, which:

1. builds + pushes `registry.sviniabanditka.com/tracker:<sha>` and `:latest`,
2. SSHes into `ssh.sviniabanditka.com` and runs `kubectl set image deployment/tracker app=...:<sha>` followed by `rollout status`.

Required GitHub secrets (Settings → Secrets and variables → Actions):

| Name | Used for |
|---|---|
| `REGISTRY_USERNAME` | login to `registry.sviniabanditka.com` |
| `REGISTRY_PASSWORD` | login to `registry.sviniabanditka.com` |
| `SSH_PRIVATE_KEY`   | root SSH key for `ssh.sviniabanditka.com` |

For a manual hot-fix deploy without going through CI:

```bash
./deploy/build-push.sh
kubectl -n tracker rollout restart deploy/tracker
kubectl -n tracker rollout status deploy/tracker
```

`imagePullPolicy: Always` + `rollout restart` is enough because we push the `:latest` tag.

## Notes

- **Single replica is intentional.** The app uses SQLite on a `ReadWriteOnce` PVC — multiple pods would corrupt the DB. Keep `replicas: 1` and `strategy: Recreate`.
- **Backups**: the app's built-in snapshot job writes into `/data/backups`. Pulling them out: `kubectl -n tracker cp tracker/<pod>:/data/backups ./backups`.
- **First admin**: `INITIAL_ADMIN_USERNAME` / `INITIAL_ADMIN_PASSWORD` from the secret are only used when the `users` table is empty, so changing them later has no effect — rotate via the in-app UI.
- **`COOKIE_SECURE=true`** is set in the deployment because we always serve over HTTPS through the ingress.
