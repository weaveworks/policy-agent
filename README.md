[![codecov](https://codecov.io/gh/weaveworks/policy-agent/branch/dev/graph/badge.svg?token=5HALYBWEIQ)](https://codecov.io/gh/weaveworks/policy-agent) [![CircleCI](https://circleci.com/gh/weaveworks/policy-agent.svg?style=shield&circle-token=1d1e7616349e46a7338b44d58c950b0eff08efa7)](https://app.circleci.com/pipelines/github/weaveworks/policy-agent?branch=dev)


# Policy Agent

Policy agent that enforces rego policies by controlling admission of violating resources.

## Features

- Enforce policies at deploy time
- Report runtime violations and compliance
- Support for multiple sinks for validation results
- Extend policies by defining your own policy using Custom Resource Definitions

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
- admissionMode: defines admission control configuration including the supported sinks and webhooks (disabled by default)
- auditMode: defines defines cluster periodical audit configuration including configuration including the supported sinks (disabled by default)

Example:

This example provides the expected format for the config file, you can define different sinks configuration for admission control mode and cluster periodical audit mode, such as (File system Sink, Flux notification Controller Sink, K8S events Sink and WeavePolicy SaaS Sink)
```
accountId: "76xdx488-a02x-78xc-32xx-8f5574bexxx"
clusterId: "76xdx488-a02x-78xc-32xx-8f5574bexxx"
kubeConfigFile: "/.kube/config"
logLevel: "Info"
admissionMode:
   enabled: true
   validationSinks:
      filesystemSink:
         fileName: ""
      fluxNotificationSink:
         address: ""
      k8sEventsSink:
         enabled: true
      saasGatewaySink:
         url: ""
auditMode:
   enabled: true
   writeCompliance: true
   validationSinks:
      filesystemSink:
         fileName: ""
      fluxNotificationSink:
         address: ""
      k8sEventsSink:
         enabled: true
      saasGatewaySink:
         url: ""
```
