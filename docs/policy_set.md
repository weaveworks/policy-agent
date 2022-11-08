# PolicySet

This is an optional resource. It is used to select group of policies to work in specific modes.

In each mode, The agent will list all the PolicySets of this mode and check which policies match any of those policy sets, Then validate the resources against them.

If there is no policy set found all policies will work on all modes.

> Note: [Tenant Policies](./policy.md#tenant-policy) is always active in the [Admission](./README.md#admission) mode, event if it is not selected in the `admission` policy sets

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

You can check the available modes [here](./README.md).

## Grouping Policies

Policies can be grouped by their ids, categories, severities, standards and tags

The policy will be matched if any of the filters are matched.


## Migration from v2beta1 to v2beta2

### New fields
- New required field `spec.mode` is added. PolicySets should be updated to set the policy set mode

Previously the agent was configured with which policy sets to use in each mode. Now we removed this argument from the agent's configuration and
add the mode to the Policyset itself. 

#### Example of the old agent configuration

```yaml
# config.yaml
admission:
   enabled: true
   policySet: admission-policy-set
   sinks:
      filesystemSink:
         fileName: admission.txt
```

#### Example of current PolicySet with mode field

```yaml
apiVersion: pac.weave.works/v2beta2
kind: PolicySet
metadata:
  name: admission-policy-set
spec:
  mode: admission
  filters:
    ids:
      - weave.policies.containers-minimum-replica-count
```


### Updated fields
- Field `spec.name` became optional.

### Deprecate fields
- Field `spec.id` is deprecated.
