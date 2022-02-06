# Code structure

## apiextensions

This folder contains the defintion of new CRDs needed by the agent, mainly the policies CRD. It defines all the required objects to make the Kubernetes API client aware of the new policy object and allows operations that are possible to the other built-in Kubernets objects.

## clients

It contains the necessary clients for the agent. It defines a client to fetch the policies CRD from the Kubernets API server and defines an informer that can watch those CRDs and build a cache that is updated with changes to the policies.

## pkg

Includes the libraries used by the agent. Namely the follownig:
- `domain`: Includes all the objects used that are needed for validation and defines the interfaces used for that opeartion
- `validation`: Includes the `Validator` interface and the validator used to parse the entities and report back the violations and compliance and report them to the configured sinks

## policies

Contains the implementations of the `PoliciesSource` interface that is responsible for returning the policies to the validation operation. It contains the `crd` implementation which fetches them from the Kubernets API.

## server

The server package houses both the probes server, used for readiness and liveness probes and the admission server, which serves as a webhook that listens to admission requests and triggers the validation process and refuses the operation if it encounters any violation.


## sink

Contains the implementations of `ValidationResultSink`, responsible of writing the validation results to a specified source.
