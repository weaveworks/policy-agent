[![codecov](https://codecov.io/gh/weaveworks/policy-agent/branch/dev/graph/badge.svg?token=5HALYBWEIQ)](https://codecov.io/gh/weaveworks/policy-agent) ![build](https://github.com/weaveworks/policy-agent/actions/workflows/build.yml/badge.svg?branch=dev) [![Contributors](https://img.shields.io/github/contributors/weaveworks/policy-agent)](https://github.com/weaveworks/policy-agent/graphs/contributors)
[![Release](https://img.shields.io/github/v/release/weaveworks/policy-agent?include_prereleases)](https://github.com/weaveworks/policy-agent/releases/latest)

# Weave Policy Agent

The Weave Policy Agent helps users have security and compliance checks on their Kubernetes clusters by enforcing rego policies at deploy time, and periodically auditing runtime resources in the cluster.

## Features

### Admission

Enforce policies at deploy time for Kubernetes and [tf-controller](https://github.com/weaveworks/tf-controller) resources.

### Audit

Report runtime violations and compliance for Kubernetes resources.

### Sinks

Support for configuring multiple sinks for the audit and admission modes. We currently support the following sinks:
- Kubernetes events
- Filesystem
- Elasticsearch
- [notification-controller](https://github.com/fluxcd/notification-controller)


### Policies

Users can use our free policies or define their own policies using our Policy CRD.

## Getting started

To get started, check out this [guide](link to getting started) on how to install the policy agent to your kubernetes cluster and explore violations.

## Documentation

Policy agent guides for runnning the agent in Weave GitOps Enterprise, and leverging all its capabilities, is available at [docs.gitops.weave.works](https://docs.gitops.weave.works/docs/policy/intro/).

Refer to this [doc](docs/README.md) for documentation on the high-level architecture and the different components that make-up the agent.

## Contribution

Need help or want to contribute? Please see the links below.
- Need help?
    - Talk to us in
      the [#weave-policy-agent channel](@todo add channel url)
      on Weaveworks Community Slack. [Invite yourself if you haven't joined yet.](https://slack.weave.works/)
- Have feature proposals or want to contribute?
    - Please create a [Github issue](https://github.com/weaveworks/weave-policy-agent/issues)
    - Learn more about contributing [here](./CONTRIBUTING.md).
