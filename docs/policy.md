# Policy CRD

This is the main resource and it is used to define policies which will be evaluated by the policy agent.

It uses [OPA Rego Language](https://www.openpolicyagent.org/docs/latest/policy-language) to evaluate the entities.

## Schema

You can find the cutom resource schema [here](../config/crd/bases/pac.weave.works_policies.yaml)


## Policy Library

Here is the Weaveworks [Policy Library](https://github.com/weaveworks/policy-library)

## Tenant Policy

It is used in [Multi Tenancy](https://docs.gitops.weave.works/docs/enterprise/multi-tenancy/) feature in [Weave GitOps Enterprise](https://docs.gitops.weave.works/docs/enterprise/intro/)

Tenant policies has a special tag `tenancy`. 