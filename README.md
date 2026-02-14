# Highway Config

GitOps-driven Kubernetes platform that bootstraps ArgoCD via Terraform and continuously deploys a hardened, observable nginx workload — with CI guardrails for secrets, IaC, and CIS benchmarks baked in.

## Architecture

```
┌──────────────────────────────────────────────────────────────┐
│  GitHub Actions CI ("Platform Guardrails")                   │
│                                                              │
│  TruffleHog ➜ Checkov ➜ kube-bench ➜ Terraform Plan/Apply   │
└──────────────────────┬───────────────────────────────────────┘
                       │
                       ▼
┌──────────────────────────────────────────────────────────────┐
│  Terraform (IaC Bootstrap)                                   │
│                                                              │
│  • Creates "highway" namespace                               │
│  • Installs ArgoCD via Helm                                  │
│  • Deploys ArgoCD Application CR (points at this repo)       │
└──────────────────────┬───────────────────────────────────────┘
                       │
                       ▼
┌──────────────────────────────────────────────────────────────┐
│  ArgoCD (Continuous Delivery)                                │
│                                                              │
│  • Watches charts/nginx-app on HEAD                          │
│  • Watches charts/observability on HEAD                      │
│  • Auto-sync with prune + self-heal                          │
└──────────┬───────────────────────────────┬───────────────────┘
           │                               │
           ▼                               ▼
┌─────────────────────────────┐ ┌─────────────────────────────┐
│  nginx workload             │ │  Observability Stack         │
│  (highway namespace)        │ │  (highway namespace)         │
│                             │ │                              │
│  Hardened: digest-pinned    │ │  • Prometheus (metrics)      │
│  image, non-root, resource  │ │  • Grafana (dashboards)      │
│  limits, probes, NodePort   │ │  • nginx-prometheus-exporter │
│                             │ │    sidecar on nginx pods     │
│  Exports metrics via        │ │  • ServiceMonitor (scrape    │
│  nginx-prometheus-exporter  │◀┤    config, 30s interval)     │
│  sidecar on :9113           │ │                              │
└─────────────────────────────┘ └─────────────────────────────┘
```

## Tech Stack

| Tool | Role | Why |
|------|------|-----|
| **Terraform** | Infrastructure bootstrap | Declarative provisioning of namespaces, ArgoCD, and the root Application CR |
| **Helm** | Kubernetes packaging | Templated charts for the ArgoCD app-of-apps pattern, nginx workload, and observability stack |
| **ArgoCD** | GitOps delivery | Auto-syncs desired state from this repo into the cluster with prune and self-heal |
| **Prometheus** | Metrics collection | Scrapes nginx metrics via ServiceMonitor; backed by kube-prometheus-stack |
| **Grafana** | Dashboards & visualization | Pre-configured with Prometheus data source for cluster and application metrics |
| **GitHub Actions** | CI pipeline | Runs security scans, IaC validation, integration tests, and CIS benchmarks on every push/PR |
| **Terratest** | Infrastructure testing | Go-based integration tests that verify the deployed service returns 200 |
| **Checkov** | Static analysis | Scans Terraform and Kubernetes manifests against CIS and best-practice policies |
| **TruffleHog** | Secret detection | Filesystem and git-history scanning to prevent credential leaks |
| **kube-bench** | CIS benchmarks | Runs the CIS Kubernetes Benchmark against the cluster and uploads the report |

## Observability

The platform ships a full monitoring stack deployed as a separate ArgoCD Application:

| Component | Detail |
|-----------|--------|
| **kube-prometheus-stack** | Prometheus + Grafana deployed via `charts/observability/` (v81.6.7) |
| **nginx-prometheus-exporter** | Sidecar container (`nginx/nginx-prometheus-exporter:1.4.1`) injected into the nginx Deployment, exposing metrics on `:9113` |
| **stub_status endpoint** | ConfigMap-driven nginx config exposes `/stub_status` on `:8081` for the exporter to scrape |
| **ServiceMonitor** | Prometheus auto-discovers the nginx metrics port with a 30-second scrape interval |
| **ServerSideApply** | Enabled on the observability Application CR to handle large CRDs that exceed the 256 KB annotation limit |

## Security Posture

| Hardening measure | Implementation |
|-------------------|----------------|
| Image pinning by digest | `nginxinc/nginx-unprivileged@sha256:7581...` — prevents tag-mutation attacks |
| Non-root execution | Container runs as UID `10001` using the `nginx-unprivileged` base image |
| Resource limits | CPU `200m` / memory `256Mi` caps prevent noisy-neighbour issues |
| Liveness & readiness probes | HTTP checks on `:8080` ensure only healthy pods receive traffic |
| Checkov policy scans | Terraform + Kubernetes frameworks scanned in CI; kube-bench job skips are annotated with justifications |
| Secret scanning | TruffleHog scans both filesystem and git history on every push |
| CIS Kubernetes Benchmark | kube-bench job runs in CI, report uploaded as artifact |

## Project Structure

```
highway-config/
├── .github/workflows/ci.yaml   # CI pipeline — scans, plans, applies, tests
├── bash/
│   ├── tf-plan-setup.sh         # Terraform plan wrapper (local + CI stages)
│   └── tf-apply-setup.sh        # Terraform apply wrapper (auto-approve + CD)
├── charts/
│   ├── argocd-app/              # Helm chart: ArgoCD Application CRs
│   │   └── templates/
│   │       ├── main.yaml        #   Points ArgoCD at charts/nginx-app
│   │       └── observability.yaml #  Points ArgoCD at charts/observability
│   ├── nginx-app/               # Helm chart: hardened nginx deployment
│   │   ├── templates/           #   Deployment, Service, ServiceMonitor,
│   │   │                        #   ConfigMap, HPA, Ingress, etc.
│   │   └── values.yaml          #   Image digest, security context, metrics config
│   └── observability/           # Helm chart: kube-prometheus-stack wrapper
│       ├── Chart.yaml           #   Depends on kube-prometheus-stack v81.6.7
│       └── values.yaml          #   Prometheus retention, Grafana settings
├── templates/                   # Raw K8s manifests (reference / pre-Helm)
├── terraform/
│   └── main.tf                  # Bootstraps namespace + ArgoCD + root app
├── test/
│   ├── kube-bench-job.yaml      # CIS benchmark Job manifest
│   └── terratest/
│       └── terraform_test.go    # Integration test: HTTP 200 from nginx
└── Makefile                     # Dev shortcuts: scan, plan, apply, clean
```

## Getting Started

### Prerequisites

- [minikube](https://minikube.sigs.k8s.io/) (or any local K8s cluster)
- [Terraform](https://www.terraform.io/) >= 1.14
- [Helm](https://helm.sh/) >= 3
- [Go](https://go.dev/) >= 1.21 (for Terratest)

### Quick Start

```bash
# Start a local cluster
minikube start

# Run security scans (TruffleHog + Checkov)
make scan

# Plan and apply infrastructure (creates namespace, ArgoCD, root app)
make tf-local

# Render Helm templates locally for debugging
make helm-render

# Tear everything down
make clean
```

### Other Make Targets

| Target | Description |
|--------|-------------|
| `make scan` | Run TruffleHog and Checkov locally |
| `make tf-plan-local` | Terraform init + plan (saves numbered `.tfplan`) |
| `make tf-apply-local` | Terraform apply (interactive approval) |
| `make tf-apply-local-auto` | Terraform apply (auto-approve latest plan) |
| `make tf-local` | Plan then auto-apply in one step |
| `make helm-render` | Render Helm templates locally for debugging |
| `make clean` | Terraform destroy |

## CI Pipeline

The **Platform Guardrails** workflow (`.github/workflows/ci.yaml`) runs on every push and PR:

1. **Secret Scan** — TruffleHog checks the repo for verified and unknown credential leaks
2. **Checkov Scan** — Static analysis of Terraform and Kubernetes manifests
3. **kube-bench** — Spins up a Kind cluster, runs the CIS Kubernetes Benchmark, and uploads the JSON report as a build artifact
4. **Terratest** — Deploys the full stack on a Kind cluster and verifies HTTP 200 from the nginx service
5. **Terraform Plan** — `terraform init` → `fmt` → `validate` → `plan`
6. **Terraform Apply** — Runs only on merged PRs to `main`
