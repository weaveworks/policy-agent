[![codecov](https://codecov.io/gh/weaveworks/policy-agent/branch/dev/graph/badge.svg?token=5HALYBWEIQ)](https://codecov.io/gh/weaveworks/policy-agent) ![build](https://github.com/weaveworks/policy-agent/actions/workflows/build.yml/badge.svg?branch=dev) [![Contributors](https://img.shields.io/github/contributors/weaveworks/policy-agent)](https://github.com/weaveworks/policy-agent/graphs/contributors)
[![Release](https://img.shields.io/github/v/release/weaveworks/policy-agent?include_prereleases)](https://github.com/weaveworks/policy-agent/releases/latest)

# Weave Policy Agent

Weave Policy Agent is a policy-as-code engine built on Open Policy Agent (OPA) that ensures security, compliance, and best practices for Kubernetes applications. Designed for GitOps workflows, especially Flux, it enables fine-grained policies for Flux applications and tenants, ensuring isolation and compliance across Kubernetes deployments.

## Features

#### Prevent violating K8s resources via admission controller
Weave Policy Agent uses the Kubernetes admission controller to monitor any Kubernetes Resource changes and prevent the ones violating the policies from getting deployed.

#### Prevent violating terraform plans via `tf-controller`
If you are using flux's terraform controller ([tf-controller](https://github.com/weaveworks/tf-controller)) to apply and sync your terraform plans, you can use Weave Policy Agent to prevent violating plans from being applied to your cluster.

#### Audit runtime compliance
The agent scans Kubernetes resources on the cluster and reports runtime violations at a configurable frequency.

#### Advanced features for flux
While the agent works natively with Kubernetes resources, Weave Policy Agent has specific features allowing fine-grained policy configurations to flux applications and tenants, as well as alerting integration with flux's `notification-controller`

#### Observability via WeaveGitOps UI
Policies and violations can be displayed on WeaveGitOps Dashboards allowing better observability of the cluster's compliance.

#### Example Policies
Example policies that target K8s and Flux best practices are available [here](policies). Users can as well write their policies in Rego using the agent policy CRD.

## Getting started

To get started, check out this [guide](docs/getting-started.md) on how to install the policy agent to your Kubernetes cluster and explore violations.

## Documentation

Policy agent guides for running the agent in Weave GitOps Enterprise, and leveraging all its capabilities, are available at [docs.gitops.weave.works](https://docs.gitops.weave.works/docs/policy/intro/).

Refer to this [doc](docs/README.md) for documentation on the high-level architecture and the different components that make up the agent.

## Contribution

Need help or want to contribute? Please see the links below.
<!-- - Need help?
    - Talk to us in
      the [#weave-policy-agent channel](@todo add channel url)
      on Weaveworks Community Slack. [Invite yourself if you haven't joined yet.](https://slack.weave.works/) -->
- Have feature proposals or want to contribute?
    - Please create a [Github issue](https://github.com/weaveworks/weave-policy-agent/issues).
    - Learn more about contributing [here](./CONTRIBUTING.md).
