# Policy Configuration

## Goal

Using the same policy with more than one configurations (Parameters, Targets, .etc)
So new CRD is introduced That will target specific resources/apps in the same or different namespaces, contains the custom configuration for policies.

## Schema

Policy Config CRD Schema as the following

```yaml
apiVersion: pac.weave.works/v2beta2 
kind: PolicyConfig    # policy config resource kind 
metadata:       
name: my-config       # policy config name
spec:
match:                # matches (targets of the policy config)
    # should be only one from the following
    namespaces:                 # add one or more name spaces
    - dev
    - prod
    apps:        # add one or more apps [HelmRelease, Kustomization]
    - kind: HelmRelease
        name: my-app            # app name
        namespace: flux-system  # app namespace [if empty will match in any namespace]
    resources:   # add one or more resources [Deployment, ReplicaSet, ..]
    - kind: Deployment          
        name: my-deployment     # resource name
        namespace: default      # resource namespace [if empty will match in any namespace]
config:        # config for policies [one or more]
    weave.policies.containers-minimum-replica-count:   
    parameters:
        replica_count: 3
```

## Priority of enforcing multiple configs for the same target [from low to high]

- Policy configs which targets the namespace.
- Policy config which targets the applications.
- Policy config which targets the kubernetes resource.

### Example

```yaml
apiVersion: pac.weave.works/v2beta2
kind: PolicyConfig
metadata:
  name: my-config-1
spec:
  match:
    namespaces:
    - default
  config:
    weave.policies.containers-minimum-replica-count:
      parameters:
        replica_count: 2
        owner: test
---

apiVersion: pac.weave.works/v2beta2
kind: PolicyConfig
metadata:
  name: my-config-2
spec:
  match:
    apps:
    - kind: Kustomization
      name: app-a
  config:
    weave.policies.containers-minimum-replica-count:
      parameters:
        replica_count: 3

---

apiVersion: pac.weave.works/v2beta2
kind: PolicyConfig
metadata:
  name: my-config-3
spec:
  match:
    apps:
    - kind: Kustomization
      name: app-a
      namespace: flux-system
  config:
    weave.policies.containers-minimum-replica-count:
      parameters:
        replica_count: 4
        owner: test2
```

- In the previous example when you apply the 3 configurations
app-a will be affected by only `my-config-3` as it matches the kind, name and namespace all together.

- If `my-config-3` is not existed, so will be affected my `my-config-2` as it's matched by kind and name

- If `my-config-2` is not existed `my-config-1` will take over then

## Priority of enforcing multiple configs for the same application/resource with/without namespace [from low to high]

- Policy configs without the namespace
- Policy configs with the namespace

### Example

```yaml
apiVersion: pac.weave.works/v2beta2
kind: PolicyConfig
metadata:
  name: my-config-4
spec:
  match:
    resources:
    - kind: Deployment
      name: deployment-1
  config:
    weave.policies.containers-minimum-replica-count:
      parameters:
        replica_count: 5

---

apiVersion: pac.weave.works/v2beta2
kind: PolicyConfig
metadata:
  name: my-config-5
spec:
  match:
    resources:
    - kind: Deployment
      name: deployment-1
      namespace: default
  config:
    weave.policies.containers-minimum-replica-count:
      parameters:
        replica_count: 6
```

- In the previous example when you apply the 2 configurations
app-a will be affected by only `my-config-5` as it matches the name and namespace all together. 

- If not existed `my-config-4` will take over

## Possible senarios for using PolicyConfig

Refer to design document [here](https://www.notion.so/weaveworks/Policy-Configuration-4864364188664656ba41f62ecb31945c#52cab387b78448c7a723f6fe449050a8)

## How to use / test against local kind cluster

- Create kind cluster and apply cert-manager

    ```
    kind create cluster --name name

    kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.9.1/cert-manager.yaml
    ```

- Build agent binary

    ```
    make build
    ```

- Create agent docker image using the version from `version.txt`

    ```
    docker build -t policy-agent:1.2.1 .
    ```

- Load docker image to the cluster

    ```
    kind load docker-image policy-agent:1.2.1 --name test
    ```

- Install policy agent helm in policy-system namespace

    ```
    kubectl create ns policy-system
    helm install agent helm -n policy-system -f helm/values.yaml
    ```

- Install the policy/policies you'd like to use from [policy-library](https://github.com/weaveworks/policy-library)

- Install policy config crd

    ```
    kubectl apply -f policyconfig.yaml
    ```

- Apply your deployment/app then your deployment will be either accepted or rejected according to the policy/policy config you applied

## Debugging

- To know which config is applied refer to the kubernetes event, It should have the config name and the applied policiy parameters. Example

    ```
    kubectl get events -A -O yaml | grep parameters
    ```