# Release Steps

## Policy Agent

- Create a pull request from `dev` to `master`
- Create a new tag and push it to origin as the following example

    ```bash
    git tag vx.x.x # replace x with your new tag
    git push origin vx.x.x # replace x with your new tag
    ```

- The `release` workflow will create a new tag for policy agent and will push the new docker image

## Releasing packages under `pkg`

- Create and push a new tag with the updated version for each packages if it has new changes
