# Workflows

## Pull request label check workflow

Currently, we have `Pull request label check` workflow to Check that PR has a label for use in release notes. 
Pull requests require exactly one label from the allowed labels:

 1. 🚀 **Enhancements** `enhancement`: New feature or request
 2. 🐛 Bugs `bug`: Something isn't working
 3. 🧪 Tests `test`: Mark a PR as being about tests
 4. Uncategorized `exclude from release notes`: Use this label to exclude a PR from the release notes

## Build Changelog and Github Release workflow
`Build Changelog and Github Release` workflow is triggered by creating a versioned tag.
This workflow creates a release and generates release notes from Pull requests labels based on changelog configuration.