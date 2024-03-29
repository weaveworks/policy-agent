# Agent and policies versioning

## Agent versioning
- Increase major version in cases of `Policy` CRD api schema change or any breaking changes.
- Minor version for new features
- Patch version for bug fixes

The pipeline pushes an image with the same tag to dockerhub and no image is pushed with the latest version anymore.

## Policy CRD versioning

Schema definition is its own go submodule. The versioning should follow the agent major version so the schema API version is consistent.

> Updating and releasing new version of the [Policy library](https://github.com/weaveworks/policy-library) should be considered if Policy CRDs has a new changes.
