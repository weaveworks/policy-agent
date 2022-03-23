[![codecov](https://codecov.io/gh/weaveworks/magalix-policy-agent/branch/dev/graph/badge.svg?token=5HALYBWEIQ)](https://codecov.io/gh/weaveworks/magalix-policy-agent) [![CircleCI](https://circleci.com/gh/weaveworks/magalix-policy-agent.svg?style=shield&circle-token=1d1e7616349e46a7338b44d58c950b0eff08efa7)](https://app.circleci.com/pipelines/github/weaveworks/magalix-policy-agent?branch=dev)


# Magalix Policy Agent

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

- `kube-config-file` | `AGENT_KUBE_CONFIG_FILE`: path to the kubernetes config file to access the cluster
- `account-id` | `AGENT_ACCOUNT_ID`: unique identifier that signifies the owner of that agent
- `cluster-id` | `AGENT_CLUSTER_ID`: unique identifier for the cluster that the agent will run against

There are additional arguments that can be specified, refer to the help for more info.

```bash
agent -h
NAME:
   Magalix agent - Enforces compliance on your kubernetes cluster

USAGE:
   agent [global options] command [command options] [arguments...]

VERSION:
   0.0.1

COMMANDS:
   help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
    --kube-config-file value  path to kubernetes client config file [$AGENT_KUBE_CONFIG_FILE]
   --account-id value        Account id, unique per organization [$AGENT_ACCOUNT_ID]
   --cluster-id value        Cluster id, cluster identifier [$AGENT_CLUSTER_ID]
   --webhook-listen value    port for the admission webhook server to listen on (default: 8443) [$AGENT_WEBHOOK_LISTEN]
   --webhook-cert-dir value  cert directory path for webhook server (default: "/certs") [$AGENT_WEBHOOK_CERT_DIR]
   --probes-listen value     address for the probes server to run on (default: ":9000") [$AGENT_PROBES_LISTEN]
   --write-compliance        enables writing compliance results (default: false) [$AGENT_WRITE_COMPLIANCE]
   --disable-admission       disables admission control (default: false) [$AGENT_DISABLE_ADMISSION]
   --disable-audit           disables cluster periodical audit (default: false) [$AGENT_DISABLE_AUDIT]
   --log-level value         app log level (default: "info") [$AGENT_LOG_LEVEL]
   --sink-file-path value    file path to write validation result to (default: "/tmp/results.json") [$AGENT_SINK_FILE_PATH]
   --metrics-addr value      address the metric endpoint binds to (default: ":8080") [$AGENT_METRICS_ADDR]
   --help, -h                show help (default: false)
   --version, -v             print the version (default: false)
```
