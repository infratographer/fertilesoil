# Fertilesoil Helm Chart

A Helm chart for things (update me!)

![Version: 1.0.0](https://img.shields.io/badge/Version-1.0.0-informational?style=flat-square) ![AppVersion: 1.0](https://img.shields.io/badge/AppVersion-1.0-informational?style=flat-square)

## Requirements

Kubernetes: `>=1.21`

| Repository | Name | Version |
|------------|------|---------|
| https://charts.bitnami.com/bitnami | common | 2.1.1 |

## Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| audit | dict | `{"auditImage":{"pullPolicy":"IfNotPresent","registry":"ghcr.io","repository":"metal-toolbox/audittail","tag":"v0.5.1"},"enabled":true,"init":{"resources":{"limits":{"cpu":"100m","memory":"128Mi"},"requests":{"cpu":"100m","memory":"128Mi"}}},"resources":{"limits":{"cpu":"500m","memory":"1Gi"},"requests":{"cpu":"100m","memory":"128Mi"}}}` | configures metal-toolbox/audittail |
| audit.auditImage | dict | `{"pullPolicy":"IfNotPresent","registry":"ghcr.io","repository":"metal-toolbox/audittail","tag":"v0.5.1"}` | Infomation about the audittail image |
| audit.enabled | bool | `true` | toggles audittail |
| audit.init.resources | dict | `{"limits":{"cpu":"100m","memory":"128Mi"},"requests":{"cpu":"100m","memory":"128Mi"}}` | Resources for the audittail init container |
| audit.resources | dict | `{"limits":{"cpu":"500m","memory":"1Gi"},"requests":{"cpu":"100m","memory":"128Mi"}}` | Resources for the audittail container |
| autoscaling.enabled | bool | `false` |  |
| clusterInfo.fqdn | string | `"example.com"` |  |
| deployment | object | `{"affinity":{},"enabled":true,"image":{"pullPolicy":"Always","registry":"quay.io","repository":"equinixmetal/CHANGEME","tag":"75-7dc6d7b6"},"nodeAffinityPreset":{"key":"","type":"","values":[]},"podAffinityPreset":"","podAntiAffinityPreset":"soft","podDisruptionBudget":{"enabled":false,"maxUnavailable":1,"minAvailable":null},"ports":[{"containerPort":8000,"name":"http"}],"replicas":2,"resources":{"limits":{"cpu":2,"memory":"4Gi"},"requests":{"cpu":2,"memory":"4Gi"}}}` | whether or not to include a deployment |
| deployment.affinity | dict | `{}` | affinity deployment Affinity for pod assignment ref: https://kubernetes.io/docs/concepts/configuration/assign-pod-node/#affinity-and-anti-affinity Note: podAffinityPreset, podAntiAffinityPreset, and nodeAffinityPreset will be ignored when it's set |
| deployment.enabled | bool | `true` | A toggle which controls the creation of a deployment |
| deployment.image | object | `{"pullPolicy":"Always","registry":"quay.io","repository":"equinixmetal/CHANGEME","tag":"75-7dc6d7b6"}` | Image map |
| deployment.image.pullPolicy | string | `"Always"` | Image pullPolicy |
| deployment.image.registry | string | `"quay.io"` | Image registry |
| deployment.image.repository | string | `"equinixmetal/CHANGEME"` | Image repository |
| deployment.image.tag | string | `"75-7dc6d7b6"` | Image tag |
| deployment.nodeAffinityPreset | object | `{"key":"","type":"","values":[]}` | nodeAffinityPreset ref: https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/#node-affinity |
| deployment.nodeAffinityPreset.key | string | `""` | key deployment Node label key to match Ignored if `affinity` is set. E.g. key: "kubernetes.io/e2e-az-name" |
| deployment.nodeAffinityPreset.values | array | `[]` | values deployment Node label values to match. Ignored if `affinity` is set. E.g. values:   - e2e-az1   - e2e-az2 |
| deployment.podAffinityPreset | string | `""` | podAffinityPreset Pod affinity preset ref: https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/#inter-pod-affinity-and-anti-affinity |
| deployment.podAntiAffinityPreset | string | `"soft"` | podAntiAffinityPreset deployment Pod anti-affinity preset. Ignored if `affinity` is set. Allowed values: `soft` or `hard` ref: https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/#inter-pod-affinity-and-anti-affinity |
| deployment.podDisruptionBudget.enabled | bool | `false` | enable PodDisruptionBudget ref: https://kubernetes.io/docs/concepts/workloads/pods/disruptions/ |
| deployment.podDisruptionBudget.maxUnavailable | int | `1` | Maximum number/percentage of pods that may be made unavailable |
| deployment.podDisruptionBudget.minAvailable | string | `nil` | Minimum number/percentage of pods that should remain scheduled. When it's set, maxUnavailable must be disabled by `maxUnavailable: null` |
| deployment.ports | array | `[{"containerPort":8000,"name":"http"}]` | container port config |
| deployment.ports[0].containerPort | int | `8000` | Port number |
| deployment.ports[0].name | string | `"http"` | Port name |
| deployment.replicas | int | `2` | Number of nginx-ingress pods to load balance between |
| deployment.resources | dict | `{"limits":{"cpu":2,"memory":"4Gi"},"requests":{"cpu":2,"memory":"4Gi"}}` | resource limits & requests ref: https://kubernetes.io/docs/user-guide/compute-resources/ |
| ingress | object | `{"annotations":{},"className":"nginx","enabled":true,"host":"CHANGEME.{{ .Values.clusterInfo.fqdn }}","labels":{},"tls":{"enabled":true,"secretName":"your-ingress-tls"}}` | ref: https://kubernetes.io/docs/concepts/services-networking/ingress/#what-is-ingress |
| ingress.annotations | dict | `{}` | Custom Ingress annotations |
| ingress.className | string | `"nginx"` | options are typically nginx or nginx-external, if omited the cluster default is used |
| ingress.enabled | bool | `true` | Set to true to generate Ingress resource |
| ingress.host | tpl/string | CHANGEME.{{ .Values.clusterInfo.fqdn }} | Set custom host name. (DNS name convention) |
| ingress.labels | dict | `{}` | Custom Ingress labels |
| ingress.tls.enabled | bool | `true` | Set to true to enable HTTPS |
| ingress.tls.secretName | string | `"your-ingress-tls"` | You must provide a secret name where the TLS cert is stored |
| securityContext | object | `{"capabilities":{"drop":["ALL"]},"readOnlyRootFilesystem":true,"runAsNonRoot":true,"runAsUser":1000}` | Security context to be added to the deployment |
| service.port | int | `80` |  |
| serviceMonitor | object | `{"annotations":{},"enabled":true,"interval":"10s","labels":{},"scrapeTimeout":"10s"}` | ServiceMonitor is how you get metrics into prometheus! |
| serviceMonitor.annotations | object | `{}` | Annotations to add to ServiceMonitor |
| serviceMonitor.enabled | bool | `true` | Set to true to create a default ServiceMonitor for your application |
| serviceMonitor.interval | string | `"10s"` | Interval for scrape metrics. |
| serviceMonitor.labels | object | `{}` | Labels to add to ServiceMonitor |
| serviceMonitor.scrapeTimeout | string | `"10s"` | time out interval when scraping metrics |
