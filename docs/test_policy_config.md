## How to use / test against local kind cluster

- Create kind cluster and apply cert-manager

    ```
    kind create cluster --name name

    kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.9.1/cert-manager.yaml
    ```

- Build agent binary

    ```
    make build
    ```

- Create agent docker image using the version from `version.txt`

    ```
    docker build -t policy-agent:1.2.1 .
    ```

- Load docker image to the cluster

    ```
    kind load docker-image policy-agent:1.2.1 --name test
    ```

- Install policy agent helm in policy-system namespace

    ```
    kubectl create ns policy-system
    helm install agent helm -n policy-system -f helm/values.yaml
    ```

- Install the policy/policies you'd like to use from [policy-library](https://github.com/weaveworks/policy-library)

- Install policy config crd

    ```
    kubectl apply -f policyconfig.yaml
    ```

- Apply your deployment/app then your deployment will be either accepted or rejected according to the policy/policy config you applied

## Debugging

- To know which config is applied refer the validation event that's stored as a kubernetes event, It should have the config name and the applied policiy parameters. Example

    ```
    kubectl get events -A -O yaml | grep parameters
    ```