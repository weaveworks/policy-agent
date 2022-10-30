# PolicySet

This is an optional resource. It is used to select group of policies to work in specific modes.

**Example**
```yaml
apiVersion: pac.weave.works/v2beta2
kind: PolicySet
metadata:
  name: my-policy-set
spec:
  mode: admission
  filters:
    ids:
      - weave.policies.containers-minimum-replica-count
    categories:
      - security
    severities:
      - high
      - medium
    standards:
      - pci-dss
    tags:
      - tag-1  
```

## Modes

### `audit`

This mode performs the audit functionality. It triggers per the specified interval (by default every 24 hour) and then lists all the resources in the cluster which the agent has access to read it and performs the validation.

> Works with policies of provider `kubernetes`


### `admission`

This contains the admission module. It uses the `controller-runtime` Kubernetes package to register a callback that will be called when the agent recieves an admission request.

> Works with policies of provider `kubernetes`


### `tf-admission`

This is a webhook used to validate terraform plans. It mainly used by the [TF-Controller](https://github.com/weaveworks/tf-controller) to enforce policies on terraform plans

> Works with policies of provider `terraform`


## Grouping Policies

Policies can be grouped by their ids, categories, severities, standards and tags

The policy will be matched if any of the filters are matched.


## Migration from v2beta1 to v2beta2

### New fields
- New required field `spec.mode` is added. PolicySets should be updated to set the policy set mode

### Updated fields
- Field `spec.name` became optional.

### Deprecate fields
- Field `spec.id` is deprecated.
