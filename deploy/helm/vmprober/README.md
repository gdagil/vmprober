# vmprober Helm chart

Deploys [vmprober](https://github.com/gdagil/vmprober) — network probe monitoring
for Prometheus & VictoriaMetrics — to Kubernetes.

## Install

```bash
# From a checkout of the repo:
helm install vmprober ./deploy/helm/vmprober

# Override the config and image tag:
helm install vmprober ./deploy/helm/vmprober \
  --set image.tag=latest-alpine \
  --set-file config=my-config.yaml   # (or edit values.yaml `config:`)
```

Verify it:

```bash
helm test vmprober          # runs the bundled /health connectivity check
kubectl port-forward svc/vmprober 8429:8429
# open http://localhost:8429/  (dashboard), /metrics, /health, /ready
```

## Key values

| Key | Default | Notes |
|-----|---------|-------|
| `replicaCount` | `1` | **Keep at 1 in push mode** — vmprober dedups per-process, so >1 replica double-pushes metrics. |
| `image.repository` | `ghcr.io/gdagil/vmprober` | |
| `image.tag` | `""` | Empty → `"<appVersion>-alpine"`. Published flavors: `-alpine`, `-scratch`. |
| `image.pullPolicy` | `IfNotPresent` | |
| `imagePullSecrets` | `[]` | For a private registry. |
| `service.type` / `service.port` | `ClusterIP` / `8429` | |
| `config` | minimal pull-only config | Rendered into a ConfigMap, mounted at `/etc/vmprober/config.yaml`. |
| `serviceMonitor.enabled` | `false` | Set `true` to scrape `/metrics` via Prometheus Operator. |
| `resources` | 100m/64Mi → 500m/256Mi | |
| `podSecurityContext` / `securityContext` | non-root, read-only rootfs, drop ALL caps | **ICMP probes need `NET_RAW`** — add it here if you enable icmp targets. |

## Config changes roll the pods

vmprober loads its config once at startup (no in-process hot-reload). The chart
adds a `checksum/config` pod annotation, so changing `.Values.config` and running
`helm upgrade` triggers a rollout automatically.

## ServiceMonitor

With `serviceMonitor.enabled=true`, a `ServiceMonitor` (Prometheus Operator CRD)
scrapes the `http` port at `pull.path` (`/metrics`). Requires the Prometheus
Operator CRDs to be installed in the cluster.
