# Getting Started With Weave Policy Agent

## Prerequisites

- [Kubernetes Cluster](https://kubernetes.io/) (v1.20 or newer)
- [Flux](https://fluxcd.io/flux/installation/) (v0.36.0 or newer) (optional)
- [Cert Manager](https://cert-manager.io/docs/installation/) (v1.5.0) or newer
- [kubectl](https://kubernetes.io/docs/tasks/tools/#kubectl)
- [helm](https://helm.sh/docs/intro/install/) (Optional)
- [kustomize](https://kubectl.docs.kubernetes.io/installation/kustomize/) (optional)

## Installing Weave Policy Agent

Weave Policy Agent can be installed using 2 methods (By using HelmRelease and Flux, By using Helm)

### Using HelmRelease and Flux

To install Weave Policy Agent using `Flux` it requires the `HelmRepository` and `HelmRelease` for the Weave Policy Agent  to be applied into your cluster by adding them to your cluster repository in a location that's readable by flux

<details>
  <summary>Click to expand Weave Policy Agent HelmRepository </summary>

```yaml
apiVersion: source.toolkit.fluxcd.io/v1beta2
kind: HelmRepository
metadata:
creationTimestamp: null
name: policy-agent
namespace: flux-system
spec:
interval: 1m0s
timeout: 1m0s
url: https://weaveworks.github.io/policy-agent/
status: {}
```
</details>

<details>
  <summary>Click to expand Weave Policy Agent HelmRelease </summary>

```yaml
apiVersion: helm.toolkit.fluxcd.io/v2beta1
kind: HelmRelease
metadata:
  name: policy-agent
  namespace: flux-system
spec:
  chart:
    spec:
      chart: policy-agent
      sourceRef:
        apiVersion: source.toolkit.fluxcd.io/v1beta2
        kind: HelmRepository
        name: policy-agent
        namespace: flux-system
      version: 2.3.0
  interval: 10m0s
  targetNamespace: policy-system
  values:
    caCertificate: ""
    certificate: ""
    config:
      accountId: ""
      admission:
        enabled: true
        sinks:
          k8sEventsSink:
            enabled: true
      audit:
        enabled: false
      clusterId: ""
    excludeNamespaces:
    - kube-system
    failurePolicy: Fail
    image: weaveworks/policy-agent
    key: ""
    persistence:
      enabled: false
    useCertManager: true
status: {}
```
</details>

### Using Helm

- Create `policy-system` namespace to install the chart in

    ```bash
    kubectl create ns policy-system
    ```

- Add the Weave Policy Agent helm chart

    ```bash
    helm repo add policy-agent https://weaveworks.github.io/policy-agent/
    ```

- Install the helm chart

    ```bash
    helm install policy-agent policy-agent/policy-agent -n policy-system
    ```

## References

- [HelmRepository](https://fluxcd.io/flux/components/source/helmrepositories/)
- [HelmRelease](https://fluxcd.io/flux/components/helm/helmreleases/)

## Installing Policies

### Using Flux

To install default policies create a `kustomization` to reference the default policies from the policy agent repository

<details>
  <summary>Click to expand Policies kustomization </summary>

```yaml
apiVersion: source.toolkit.fluxcd.io/v1
kind: GitRepository
metadata:
  name: policies
  namespace: default
spec:
  interval: 5m
  url: https://github.com/weaveworks/policy-agent/
  ref:
    branch: open-source-policy-agent # TODO: change to master
---
apiVersion: kustomize.toolkit.fluxcd.io/v1
kind: Kustomization
metadata:
  name: policies
  namespace: default
spec:
  interval: 10m
  targetNamespace: default
  sourceRef:
    kind: GitRepository
    name: policies
  path: "./policies"
  prune: true
  timeout: 1m
```
</details>

### Using Kustomize

You can use kustomize to install the default policies from the Policy Agent repository by applying this kustomization

<details>
  <summary>Click to expand the default policies kustomization </summary>

```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
- github.com/weaveworks/policy-agent/policies
```

</details>

### Verify Policies Installation

By using the following command

```bash
kubectl get policies
```

## Explore Violations

- Sync a new or existing service that has one or more violations and watch it getting blocked by the Policy Agent 
- If you don’t have a violating service and want to test the agent out, you can apply this violating service as an example

    <details>
    <summary>Click to expand violating deployment </summary>

    ```yaml
    apiVersion: apps/v1
    kind: Deployment
    metadata:
    name: nginx-deployment
    namespace: default
    labels:
        app: nginx
    spec:
    replicas: 1
    selector:
        matchLabels:
        app: nginx
    template:
        metadata:
        labels:
            app: nginx
        spec:
        containers:
        - name: nginx
            image: nginx:1.14.2
            ports:
            - containerPort: 80
    ```

    </details>

- The violation of the admission controller should look like this

    <details>
    <summary>Click to expand the admission controller response for the violation </summary>

    ```bash
    Error from server (==================================================================
    Policy	: weave.policies.controller-serviceaccount-tokens-automount
    Entity	: deployment/nginx-deployment in namespace: default
    Occurrences:
    - 'automountServiceAccountToken' must be set; found '{"containers": [{"image": "nginx:1.14.2", "imagePullPolicy": "IfNotPresent", "name": "nginx", "ports": [{"containerPort": 80, "protocol": "TCP"}], "resources": {}, "terminationMessagePath": "/dev/termination-log", "terminationMessagePolicy": "File"}], "dnsPolicy": "ClusterFirst", "restartPolicy": "Always", "schedulerName": "default-scheduler", "securityContext": {}, "terminationGracePeriodSeconds": 30}'
    ): error when creating "deployment.yaml": admission webhook "admission.agent.weaveworks" denied the request: ==================================================================
    Policy	: weave.policies.controller-serviceaccount-tokens-automount
    Entity	: deployment/nginx-deployment in namespace: default
    Occurrences:
    - 'automountServiceAccountToken' must be set; found '{"containers": [{"image": "nginx:1.14.2", "imagePullPolicy": "IfNotPresent", "name": "nginx", "ports": [{"containerPort": 80, "protocol": "TCP"}], "resources": {}, "terminationMessagePath": "/dev/termination-log", "terminationMessagePolicy": "File"}], "dnsPolicy": "ClusterFirst", "restartPolicy": "Always", "schedulerName": "default-scheduler", "securityContext": {}, "terminationGracePeriodSeconds": 30}'

    ```

    </details>

- To view the violating events by using `kubectl`

    ```bash
    kubectl get events --field-selector type=Warning,reason=PolicyViolation -A
    ```

- To view the violating events by using WeaveGitOps UI

    // TODO: add screenshot

## Fix & Exclude

// TODO:
Mention of how to resolve section in policies → [[screenshot of policies UI]] 
Mention that users can excludeNamespaces, because usually there are certain namespaces that we are ok with having violations and we don’t want to stop deployments from happening 


## FAQs
// TODO:
I use WeaveGitOps to view violations 
