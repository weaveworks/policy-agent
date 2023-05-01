[![codecov](https://codecov.io/gh/weaveworks/policy-agent/branch/dev/graph/badge.svg?token=5HALYBWEIQ)](https://codecov.io/gh/weaveworks/policy-agent) ![build](https://github.com/weaveworks/policy-agent/actions/workflows/build.yml/badge.svg?branch=dev) [![Contributors](https://img.shields.io/github/contributors/weaveworks/policy-agent)](https://github.com/weaveworks/policy-agent/graphs/contributors)
[![Release](https://img.shields.io/github/v/release/weaveworks/policy-agent?include_prereleases)](https://github.com/weaveworks/policy-agent/releases/latest)

# Policy Agent

Policy agent that enforces rego policies by controlling admission of violating resources.

## Documentation

Policy agent guides for runnning the agent in Weave GitOps Enterprse, and leverging all its capabilities, is available at [docs.gitops.weave.works](https://docs.gitops.weave.works/docs/policy/intro/).

Refer to this [doc](docs/README.md) for documentation on the high-level architecture and the different components that make up the agent.

## Features

- Enforce policies at deploy time
- Report runtime violations and compliance
- Support for multiple sinks for validation results
- Extend policies by defining your own policy using CustomResourceDefinitions

## Running the Agent

### Kubernetes Workload

Refer to this [doc](docs/running_agent.md) for the steps needed to run the agent with all its necessary componenets.

### Local

The agent needs the following arguments to start, they can be specified as command line arguments or as environment variables:

- `config-file` | `AGENT_CONFIG_FILE`: path to the policy agent config file

There are additional arguments that can be specified, refer to the help for more info.

```bash
agent -h
NAME:
   Policy agent - Enforces compliance on your kubernetes cluster

USAGE:
   main [global options] command [command options] [arguments...]

VERSION:
   0.0.1

COMMANDS:
   help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --config-file value  path to policy agent configuration file [$AGENT_CONFIG_FILE]
   --help, -h           show help (default: false)
   --version, -v        print the version (default: false)
```

### Policy Agent config File

The config file is the single entry point for configuring the agent.

The agent needs the following parameters to be provided in the configuration yaml file:
- kubeConfigFile: path to the kubernetes config file to access the cluster
- accountId: unique identifier that signifies the owner of that agent
- clusterId: unique identifier for the cluster that the agent will run against


There are additional parameters could be provided:
- logLevel: app log level (default: "info")
- probesListen: address for the probes server to run on (default: ":9000")
- metricsAddress: address the metric endpoint binds to (default: ":8080")
- admission: defines admission control configuration including the supported sinks and webhooks (disabled by default)
- audit: defines defines cluster periodical audit configuration including configuration including the supported sinks (disabled by default)

Example:

This example provides the expected format for the config file, you can define different sinks configuration for admission control mode and cluster periodical audit mode, such as (File system Sink, Flux notification Controller Sink and K8S events Sink)
```
accountId: "76xdx488-a02x-78xc-32xx-8f5574bexxx"
clusterId: "76xdx488-a02x-78xc-32xx-8f5574bexxx"
kubeConfigFile: "/.kube/config"
logLevel: "Info"
admission:
   enabled: true
   sinks:
      filesystemSink:
         fileName: ""
      fluxNotificationSink:
         address: ""
      k8sEventsSink:
         enabled: true
audit:
   enabled: true
   writeCompliance: true
   sinks:
      filesystemSink:
         fileName: ""
      fluxNotificationSink:
         address: ""
      k8sEventsSink:
         enabled: true
```
## Install included policies

The policy agent is shipped with 5 sample [policies](./policies/). To include those policies while installing the agent in Weave GitOps Enterprise using the policy profile, add `weave-policy-agent` repo as a source for policies in the profile values with the path `./policies` as shown in the example below. Refer to this [document](https://docs.gitops.weave.works/docs/policy/weave-policy-profile/#policy-sources) for more information.

```yaml
policySource:
  enabled: true
  url: https://github.com/weaveworks/policy-agent
  tag: <add-latest-tag-here>
  path: ./policies/
```

**Note**  Policies can be applied directly on the cluster

```bash
kubectl apply -f <path-to-policy>
```

## Contribution

Need help or want to contribute? Please see the links below.
- Need help?
    - Talk to us in
      the [#weave-policy-agent channel](@todo add channel url)
      on Weaveworks Community Slack. [Invite yourself if you haven't joined yet.](https://slack.weave.works/)
- Have feature proposals or want to contribute?
    - Please create a [Github issue](https://github.com/weaveworks/weave-policy-agent/issues)
    - Learn more about contributing [here](./CONTRIBUTING.md).
