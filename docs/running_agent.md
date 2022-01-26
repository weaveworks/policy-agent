# Running the agent with admission control

In order to get the agent up and running there are some extra steps that needs to be performed by the user to have a usable deployment file. Those additional configurations are user sepcific and for now they are left up to the user to fill them out. In the future those details should be pre filled to the user though either an interface or through generating those values to the user.

## policies

The agent needs access to policies to validate the user's requests against. These are provided to him as CRDs running in the cluster. Thus having those CRDs in the cluster is necessary for the agent to function. The policies are hosted in the policies library [repo](https://github.com/MagalixTechnologies/policy-library).

To install the policies you first need to register the policy CRD. At the root of the repo run:

```bash
kubectl apply -f crd.yaml
```

You can then check the `policies` directory and apply the required policies.

## Agent componenets

The following are the `Kubernetes` entities needed to run the agent:

- `Namespace`: hosts the agents componenets
- `ServiceAccount`, `ClusterRole` and `ClusterRoleBinding`: to give necessary permission to policies CRD
- `PersistentVolumeClaim`: to persist the validation result to storage(needs user input)
- `ConfigMap`: for agent configuration
- `Secret`: contains the self signed certificates to be mounted to the agent container(needs user input)
- `Deployment`: agent deployment(needs user input to mount the persistent volume)
- `Service`: exposes the admission webhook server
- `ValidatingWebhookConfiguration`: configures the admission control(needs user input for the CA public certificate)

## Filling in the data

Before you could run the yaml [file](../agent.yaml) the following values need to be manually filled first.

### Persistent volume

The agent writes the result of its validation requests to a local file. This file naturally won't persist after a container restart and thus the history will be removed and the written results won't be able to be exported or used afterwards. Hence it is pretty important to mount the results file to an existing storage solution if the results of the validation are of importance to the user.

The user will have to define a [persistent volume](https://kubernetes.io/docs/concepts/storage/persistent-volumes/) and then define a claim that binds to that volume and mount that on the agent deployment.

An example of a `PersistentVolumeClaim`:

```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: magalix-policy-agent
  namespace: magalix-system
spec:
  storageClassName: manual
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 1Gi
```

The `agent.yaml` file assumes that the user would want to mount his results file and that it uses a claim with the name `magalix-policy-agent` in the `magalix-system` namespace. If this isn't a desired behavior then these sections needs to be removed from the file:

From the `containers` section in the yaml:

```yaml
volumeMounts:
    ...
    - name: validation-results
      mountPath: /var
```

```yaml
volumes:
    ...
    - name: validation-results
      persistentVolumeClaim:
        claimName: magalix-policy-agent
```

### TLS certificates

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
We need to add the `tls.crt` and `tls.key` to the secret `magalix-policy-agent`. First they should be base64 encoded.

```bash
crt_base64="$(cat tls.crt | base64)"
key_base64="$(cat tls.key | base64)"
```

Then in the yaml:

```yaml
data:
  tls.crt: |
      $crt_base64
  tls.key: |
      $key_base64
```

Encode the CA certificate:

```bash
ca_b64="$(openssl base64 -A <"ca.crt")"
```

Lastly add it to the `ValidatingWebhookConfiguration`:

```yaml
webhooks:
  - name: admission.agent.magalix
    clientConfig:
      service:
        namespace: magalix-system
        name: magalix-policy-agent
        path: /admission
      caBundle: $ca_b64
```

