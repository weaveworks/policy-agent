# Policy Agent Helm Release

## Installation
```bash
helm repo add policy-agent https://weaveworks.github.io/policy-agent/
```

## Configuration

List of available variables:


| Key                   | Type          | Default                   | Description                                                                                               |
|-----------------------|---------------|---------------------------|-----------------------------------------------------------------------------------------------------------|
| `image`               | `string`      | `weaveworks/policy-agent` | docker image.                                                                                             |
| `useCertManager`      | `boolean`     | `true`                    | use [cert-manager](https://cert-manager.io/) to manage agent's TLS certificate.                           |
| `certificate`         | `string`      |                           | TLS certificate. Not needed if `useCertManager` is set to `true`.                                         |
| `key`                 | `string`      |                           | TLS key. Not needed if `useCertManager` is set to `true`.                                                 |
| `caCertificate`       | `string`      |                           | TLS CA Certificate . Not needed if `useCertManager` is set to `true`.                                     |
| `failurePolicy`       | `string`      | `Fail`                    |  Whether to fail or ignore when the admission controller request fails. Available values `Fail`, `Ignore` |
| `excludeNamespaces`   | `[]string`    |                           | List of namespaces to ignore by the admission controller.                                                 |
| `config`              | `object`      |                           | Agent configuration. See agent's configuration [guide](../docs/README.md#configuration).                  |
