# Release Steps

## Policy Agent

- Update policy agent version with the new version under [version.txt](../version.txt)
- Create a pull request from `dev` to `master`
- The `release` workflow will create a new tag for policy agent and will push the new docker image

## Releasing packages under `pkg`

- Create and push a new tag with the updated version for each packages if it has new changes
