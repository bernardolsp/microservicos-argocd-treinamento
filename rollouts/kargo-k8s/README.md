# Kargo-K8s: Deployment Normal para Kargo

Este diretório contém manifests de **Deployment normal** (não Rollout) para uso com Kargo.

## Estrutura

```
kargo-k8s/
├── base/                      # Recursos base
│   ├── deployment.yaml       # Deployment comum (kind: Deployment)
│   ├── service.yaml          # Service
│   ├── ingress.yaml          # Ingress
│   └── kustomization.yaml    # Kustomize base
└── overlays/                 # Overlays por ambiente
    ├── dev/                  # Desenvolvimento
    ├── staging/              # Staging
    └── prod/                 # Produção
```

## Diferença: Rollout vs Deployment

| Aspecto | Rollout (Argo Rollouts) | Deployment (K8s nativo) |
|---------|------------------------|-------------------------|
| **Kind** | `argoproj.io/Rollout` | `apps/v1/Deployment` |
| **Canary** | Nativo | Via ArgoCD Sync |
| **Blue/Green** | Nativo | Via ArgoCD Sync |
| **Uso com Kargo** | Sim | Sim (este exemplo) |

## Como funciona com Kargo?

1. **Kargo detecta** nova imagem na Warehouse
2. **Kargo atualiza** o overlay (ex: `overlays/dev/kustomization.yaml`) com nova tag
3. **ArgoCD faz sync** do novo commit Git
4. **Kubernetes faz rolling update** do Deployment

## Uso Manual

```bash
# Aplicar em dev
kubectl apply -k overlays/dev

# Aplicar em staging
kubectl apply -k overlays/staging

# Aplicar em prod
kubectl apply -k overlays/prod
```

## Uso com ArgoCD

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: demo-app-dev
  annotations:
    kargo.akuity.io/authorized-stage: kargo-demo:dev
spec:
  project: default
  source:
    repoURL: https://github.com/bernardolsp/microservicos-argocd-treinamento
    targetRevision: HEAD
    path: rollouts/kargo-k8s/overlays/dev
  destination:
    server: https://kubernetes.default.svc
    namespace: demo-app-dev
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
```

## Como Kargo atualiza a imagem

O Kargo usa o step `kustomize-set-image` para atualizar a imagem no Git:

```yaml
# No Stage spec
promotionTemplate:
  spec:
    steps:
    - uses: git-clone
      config:
        repoURL: https://github.com/bernardolsp/microservicos-argocd-treinamento
        checkout:
        - branch: main
          path: ./src
        - branch: env/dev
          create: true
          path: ./out
    - uses: kustomize-set-image
      config:
        images:
        - image: devlopesbernardo/version-app
          tag: "${{ imageFrom(devlopesbernardo/version-app).Tag }}"
```

Isso atualiza o kustomization.yaml do overlay com a nova tag da imagem.

## Versões disponíveis

- `v1.0.0` - Comportamento normal
- `v2.0.0` - Com erros (50% de falhas)
- `v3.0.0` - Lento (alta latência)

Use estas versões para testar o pipeline de promoção do Kargo.
