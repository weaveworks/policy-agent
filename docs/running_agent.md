# Running the Agent

In order to get the agent up and running there are some extra steps that needs to be performed by the user to have a usable deployment file. Those additional configurations are user sepcific and for now they are left up to the user to fill them out. In the future those details should be pre filled to the user through either an interface or through generating those values to the user.

## Policies

The agent needs access to policies to validate the user's requests against. These are provided to him as CRDs running in the cluster. Thus having those CRDs in the cluster is necessary for the agent to function. The policies are hosted in the policies library [repo](https://github.com/MagalixTechnologies/policy-library).

To install the policies you first need to register the policy CRD. At the root of the repo run:

```bash
kubectl apply -f crd.yaml
```

You can then check the `policies` directory and apply the required policies.

## Agent Componenets

The following are the `Kubernetes` entities needed to run the agent:

- `Namespace`: hosts the agents componenets
- `ServiceAccount`, `ClusterRole` and `ClusterRoleBinding`: to give necessary permission to policies CRD
- `PersistentVolumeClaim`: to persist the validation result to storage(needs user input)
- `ConfigMap`: for agent configuration
- `Secret`: contains the self signed certificates to be mounted to the agent container(needs user input)
- `Deployment`: agent deployment(needs user input to mount the persistent volume)
- `Service`: exposes the admission webhook server
- `ValidatingWebhookConfiguration`: configures the admission control(needs user input for the CA public certificate)

## Required Data

Before you can start running the agent, the following values need to be filled first. You need to have [helm](https://helm.sh/) installed to generate the agent yaml. The chart directory for the agent can be found [here](../helm/Chart.yaml).

### Persistent Volume

The agent writes the result of its validation requests to a local file. This file naturally won't persist after a container restart and thus the history will be removed and the written results won't be able to be exported or used afterwards. Hence it is pretty important to mount the results file to an existing storage solution if the results of the validation are of importance to the user.

The user will have to define a [persistent volume](https://kubernetes.io/docs/concepts/storage/persistent-volumes/) and then define a claim that binds to that volume and mount that on the agent deployment.

To use a persistent volume you need to set the following values in your helm values file:

```yaml
storageClassName: example # storage class for the persistent volume
usePersistence: true # flag to add claim configuration to yaml and mounts to the deployment
claimStorage: xGi # claim size, optional
sinkDir: /dir # location of mount, optional
```

This will generate a claim from a pre-existing volume and mount those volumes to the agent with the specified configuration.

### Configmap

The configmap needs to be configured to configure the agent with the available command line arguments. The following are the required arguements before the agent can start:

```yaml
accountId: account-id # unique identifier for the agent owner
clusterId: cluster-id # unique identifier for the cluster that the agent is running on
```
To configure the agent using argemnts, you need to define a config object in the values file that will be added to the configmap, for example:

```yaml
config:
  AGENT_WRITE_COMPLIANCE: 1 # enables writing compliance results
  AGENT_AUDIT: 0 # disables audit functionality
```

### TLS Certificates

For a server to be registered as an admission control it needs to be able to serve HTTPS traffic. Since the server should only be accessible as a webhook server, self signed certificates will be used.

The following steps shows how to generate the certificate from a self signed CA using `openssl`.

To generate the CA cert and private key:

```bash
openssl req -nodes -x509 -new -keyout ca.key -out ca.crt -sha256 -days 365
```

You will be prompted to fill out metadata for the CA.

To generate a private key for the server:

```bash
openssl genrsa -out tls.key 2048
```

Next you will need to generate a Certificate Signing Request to get a certificate from the self signed CA.

```bash
cat >admission.conf <<EOF
[req]
req_extensions = v3_req
distinguished_name = req_distinguished_name
prompt = no
[req_distinguished_name]
CN = magalix-policy-agent.magalix-system.svc
[ v3_req ]
basicConstraints = CA:FALSE
keyUsage = nonRepudiation, digitalSignature, keyEncipherment
extendedKeyUsage = clientAuth, serverAuth
subjectAltName = @alt_names
[alt_names]
DNS.1 = magalix-policy-agent.magalix-system.svc
DNS.2 = magalix-policy-agent.magalix-system
EOF

openssl req -new -key tls.key -subj "/CN=magalix-policy-agent.magalix-system.svc" -config admission.conf -out admission.csr
```

Generate the certificate:

```bash
openssl x509 -req -in admission.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out tls.crt -extensions v3_req -extfile admission.conf
```

Now that we generated the needed files we need to add them to the yaml.
We need to add the `tls.crt` and `tls.key` to the secret `magalix-policy-agent`. As well as the configuration for the validating webhook.
To do that we copy the content of the files to the helm values file:

```yaml
certificate: |
  <crt>
key: |
  <key>
CaCertificate:
  <cacert>
```

## Generating the File

After filling all the data you need to generate the valid deployment using the values file and `helm`, run this command at the root of thte repo:

```bash
helm3 template -f {values-file} ./helm > agent.yaml
```
