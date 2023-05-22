# Getting Started With Weave Policy Agent

## Prerequisites

- [Kubernetes Cluster](https://kubernetes.io/) >= v1.20
- [Flux](https://fluxcd.io/flux/installation/) >= v0.36.0 (optional)
- [Cert Manager](https://cert-manager.io/docs/installation/) >= v1.5.0
- [kubectl](https://kubernetes.io/docs/tasks/tools/#kubectl)
- [helm](https://helm.sh/docs/intro/install/) (Optional)
- [kustomize](https://kubectl.docs.kubernetes.io/installation/kustomize/) (optional)

## Installing Weave Policy Agent

This document will cover two approaches for installing the agent. The first is using Flux and Helm Releases as part of a GitOps ecosystem where Flux will take care of the installation, while the second is using Helm to install the agent directly on the cluster.
For both scenarios, the examples used installs the agent with only the admission mode enabled and Kubernetes events configured as the sink for the violations. 

### Using HelmRelease and Flux

To install the Weave Policy Agent using `Flux`, create a `HelmRepository` and `HelmRelease` that reference the agent, and add them to your cluster's repository in a location reconcilable by flux. 

<details>
  <summary>Click to expand HelmRepository </summary>

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
  <summary>Click to expand HelmRelease </summary>

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

Once the `HelmRepository` and `HelmRelease` are reconciled by `Flux`, you should find the Policy Agent installed on your cluster.

Check installation status using the below commands, you should expect to see the success of HelmRelease installation and the pod of the agent running

```bash
flux get helmrelease -A
kubectl get pods -n policy-system
```

### Using Helm

Create `policy-system` namespace to install the chart in

  ```bash
  kubectl create ns policy-system
  ```

Add the Weave Policy Agent helm chart

  ```bash
  helm repo add policy-agent https://weaveworks.github.io/policy-agent/
  ```

Install the helm chart

  ```bash
  helm install policy-agent policy-agent/policy-agent -n policy-system
  ```

Check installation status using the below command, you should expect the pod of the agent running

```bash
kubectl get pods -n policy-system
```

## Installing Policies

The [Policy CRD](../helm/crds/pac.weave.works_policies.yaml) is used to define policies which are then consumed and used by the agent to validate entities.

It uses OPA Rego Language to evaluate the entities.

### Installing Policies Using Flux

To install the default policies, create a `kustomization` to reference the default policies from the policy agent repository and push it to your cluster's repository.

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

### Installing Policies Using Kustomize

You can use kustomize to install the default policies from the Policy Agent repository by applying this kustomization directly to your Kubernetes cluster.

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

You can verify the installation by running the following command. If the installation is successful, the output will show a list of all the default policies.

```bash
kubectl get policies
```

## Explore Violations
Now that you have the agent and policies in place, it's time to watch the agent block violating resources. [One](link to replica policy) of the default policies blocks deployments that have replicas lower than 2. 
You can create or update a Deployment and set the `replicas` value to 1. The agent will block it because it violates the policy.
If you don't have a violating Deployment, you can use the Deployment below as an example.
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

The agent's admission controller will block the deployment and show the output below.

    <details>
    <summary>Click to expand the admission controller response for the violation </summary>

    ```bash
    Error from server (==================================================================
    ==================================================================
    Policy	: weave.policies.containers-minimum-replica-count
    Entity	: deployment/nginx-deployment in namespace: default
    Occurrences:
    - Replica count must be greater than or equal to '2'; found '1'.
    ): error when creating "deployment.yaml": admission webhook "admission.agent.weaveworks" denied the request: 
    ==================================================================
    Policy	: weave.policies.containers-minimum-replica-count
    Entity	: deployment/nginx-deployment in namespace: default
    Occurrences:
    - Replica count must be greater than or equal to '2'; found '1'.
    ```

    </details>

Since Kubernetes events are configured as a sink for the admission mode, you can use kubectl to list the violatons.

    ```bash
    kubectl get events --field-selector type=Warning,reason=PolicyViolation -A
    ```

- To view the violating events by using WeaveGitOps UI

    ![WeaveGitOps UI](imgs/violations.png)

## Fix & Exclude

To fix the violation, each policy has a `how_to_solve` section and it's used by the admission controller to make a suggestion for you to how to fix the violation in your resource `yaml` file. The following example for Minimum Replica Count Policy
  
  ![how to solve](./imgs/how-to-solve.png)

  ```bash
  Policy	: weave.policies.containers-minimum-replica-count
  Entity	: deployment/nginx-deployment in namespace: default
  Occurrences:
  - Replica count must be greater than or equal to '2'; found '1'.
  ```

To prevent the agent from scanning certain namespaces and stop deployments, you can add these namespaces to `excludeNamespaces` in the Policy Agent helm chart values file.

To prevent a certain policy from running in a specific namespace, you can add these namespaces to the policy's `exclude_namespaces` parameter, either by a direct modification to the policy file or by using `kustomize` overlays.

## References

- [HelmRepository](https://fluxcd.io/flux/components/source/helmrepositories/)
- [HelmRelease](https://fluxcd.io/flux/components/helm/helmreleases/)

## FAQ
