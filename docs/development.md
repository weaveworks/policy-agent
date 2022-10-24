# Development

## Running locally

While the agent can be run like any go binary this won't be ideal in certain situations. Mainly to try the audit functionality the agent instance needs to be reachable by the cluster so it can send the admission request to the webhook server. This means that the agent needs to be run as a workload inside a cluster with a service so it can be reached from the API server. It might also be possible that you want to test the way user handle permissions and you would need to define a `ClusterRole` for the agent.

The easiest way to achieve this is by using the provided helm chart.

You will need (cert-manager)[https://cert-manager.io/docs/installation/] on your cluster before using the agent locally.

The first step would be to build the agent binary this is by done by running the following in the repo root:

```bash
make build
```

This will generate a binary at `bin/agent`  but we won't be using directly. We will build a local image:

```bash
docker build . -t agent-test:{chart-version}
```

Notice that the tag version needs to be the same as the helm chart version. You can give any name to the image.

You know have an image that you can add to your local cluster. This vary depending on which provider you are using.

If you are using `kind`:

```bash
kind load docker-image agent-test:1.0.0 --name {clustername}
```

With `minikube` you will need to run this in a shell before building an image and that will build the image inside the `minikube` cluster:


```bash
eval $(minikube docker-env)
```

Next you will have to create your values file, you can configure the agent inside as necessary overriding the default values. The following needs to be configured:

```yaml
image: agent-test
config:
  accountId: "agent-dev"
  clusterId: "wge-dev"
```

Then you can finally run the agent:

```bash
helm3 install agent -f {values-file-path} helm -npolicy-system
```

When agent pod is ready, it should be good to go.
