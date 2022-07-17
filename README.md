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
