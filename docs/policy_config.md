# Policy Configuration

## Goal

Users sometimes need to enforce the same policy(s) with different configurations (parameters) for different targets (applications, resources, or namespaces).

## Schema

A new `PolicyConfig` CRD allows using policies with multiple configurations by configuring policy parameters based on a certain match on applications or resources with Schema and match with one of the following

- Match by namespace

  ```yaml
  apiVersion: pac.weave.works/v2beta2 
  kind: PolicyConfig      # policy config resource kind 
  metadata:       
    name: my-config       # policy config name
  spec:
    match:                # matches (targets of the policy config)
      namespaces:         # add one or more name spaces
      - dev
      - prod
    config:               # config for policies [one or more]
      weave.policies.containers-minimum-replica-count:   
        parameters:
          replica_count: 3
  ```

- Match by apps

  ```yaml
  apiVersion: pac.weave.works/v2beta2 
  kind: PolicyConfig   # policy config resource kind 
  metadata:       
    name: my-config    # policy config name
  spec:
    match:             # matches (targets of the policy config)
      apps:            # add one or more apps [HelmRelease, Kustomization]
      - kind: HelmRelease
        name: my-app            # app name
        namespace: flux-system  # app namespace [if empty will match in any namespace]
    config:            # config for policies [one or more]
      weave.policies.containers-minimum-replica-count:   
        parameters:
          replica_count: 3
  ```

- Match by resources

  ```yaml
  apiVersion: pac.weave.works/v2beta2 
  kind: PolicyConfig   # policy config resource kind 
  metadata:       
    name: my-config    # policy config name
  spec:
    match:             # matches (targets of the policy config)
      resources:       # add one or more resources [Deployment, ReplicaSet, ..]
      - kind: Deployment          
        name: my-deployment     # resource name
        namespace: default      # resource namespace [if empty will match in any namespace]
    config:            # config for policies [one or more]
      weave.policies.containers-minimum-replica-count:   
        parameters:
          replica_count: 3
  ```

## Priority of enforcing multiple configs for the same target [from low to high]

- Policy configs which targets the namespace.
- Policy config which targets an application in all namespaces.
- Policy config which targets an application in a certain namespace.
- Policy config which targets a kubernetes resource in all namespaces.
- Policy config which targets a kubernetes resource in a specific namespace.

**Note**: 
- All configs are applied from low priority to high priority as well as common parameters between configs.
- Each config only affectes the parameters defined in it.

### Example

```yaml
apiVersion: pac.weave.works/v2beta2
kind: PolicyConfig
metadata:
  name: my-config-1
spec:
  match:
    namespaces:
    - flux-system
  config:
    weave.policies.containers-minimum-replica-count:
      parameters:
        replica_count: 2
        owner: owner-1
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
---
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
        owner: owner-4
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
      namespace: flux-system
  config:
    weave.policies.containers-minimum-replica-count:
      parameters:
        replica_count: 6
```

**In the previous example when you apply the 5 configurations**

- `app-a` will be affected by `my-config-5`. It will be applied on the policies defined in it, which will affect deployment `deployment-1` in namespace `flux-system` as it matches the kind, name and namespace. 

  Final config values will be as the following:

    ```yaml
      config:
        weave.policies.containers-minimum-replica-count:
          parameters:
            replica_count: 6
            owner: owner-4
    ```
  - <em>Deployment `deployment-1` in namespace `flux-system` replica_count must be `>= 6`</em>
  - <em>Also it will be affected by `my-config-4` for `owner` configuration parameter `owner: owner-4`</em>


**In the previous example when you apply the `my-config-1`, `my-config-2`, `my-config-3` and `my-config-4`**

- `my-config-4` will be applied on the policies defined in it. which will affect deployment `deployment-1` in all namespaces as it matches the kind and name only.

  Final config values will be as the following:

    ```yaml
      config:
        weave.policies.containers-minimum-replica-count:
          parameters:
            replica_count: 5
            owner: owner-4
    ```

  - <em>Deployment `deployment-1` in all namespaces replica_count must be `>= 5`</em>
  - <em>Also it will be affected by `my-config-4` for `owner` configuration parameter `owner: owner-4`</em>

**In the previous example when you apply the `my-config-1`, `my-config-2` and `my-config-3`**

- `my-config-3` will be applied on the policies defined in it. which will affect application `app-a` and all the resources in it in namespace `flux-system` as it matches the kind, name and namespace.

  Final config values will be as the following:

    ```yaml
      config:
        weave.policies.containers-minimum-replica-count:
          parameters:
            replica_count: 4
            owner: owner-1
    ```

  - <em>Application `app-a` and all the resources in it in namespaces `flux-system` replica_count must be `>= 4`</em>
  - <em>Also it will be affected by `my-config-1` for `owner` configuration parameter `owner: owner-1`</em>

**In the previous example when you apply the `my-config-1` and `my-config-2`**

- `my-config-2` will be applied on the policies defined in it. which will affect application `app-a` and all the resources in it in all namespaces as it matches the kind and name only.

  Final config values will be as the following:

    ```yaml
      config:
        weave.policies.containers-minimum-replica-count:
          parameters:
            replica_count: 3
            owner: owner-1
    ```

  - <em>Application `app-a` and all the resources in it in all namespaces replica_count must be `>= 3`</em>
  - <em>Also it will be affected by `my-config-1` for `owner` configuration parameter `owner: owner-1`</em>

**In the previous example when you apply the `my-config-1`**

- `my-config-1` will be applied on the policies defined in it. which will affect the namespace `flux-system` with all applications and resources in it as it matches by namespace only.

  Final config values will be as the following:

    ```yaml
      config:
        weave.policies.containers-minimum-replica-count:
          parameters:
            replica_count: 2
            owner: owner-1
    ```

  - <em>Any application or resource in namespace `flux-system` replica_count must be `>= 2`</em>
  - <em>Also it will be affected by `my-config-1` for `owner` configuration parameter `owner: owner-1`</em>


**Note** 

- You can use one or more policies as the following example

  ```yaml
  apiVersion: pac.weave.works/v2beta2
  kind: PolicyConfig
  metadata:
    name: my-app-config
  spec:
    match:
      resources:
        name: my-deployment
        kind: Deployment
    config:
      weave.policies.policy-1:
        params:
          replica_count: 3
      weave.policies.policy-2:
        params:
          run_as_root: true
  ```

## Possible senarios for using PolicyConfig

Refer to design document [here](https://www.notion.so/weaveworks/Policy-Configuration-4864364188664656ba41f62ecb31945c#52cab387b78448c7a723f6fe449050a8)
