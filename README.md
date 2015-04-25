# GitHub Releases Resource

Fetches and creates versioned GitHub resources.

## Source Configuration

* `user`: *Required.* The GitHub username or organization name for the
  repository that the releases are in.

* `repository`: *Required.* The repository name that contains the releases.

* `access_token`: *Optional.* The GitHub access token that should be used to
  access the API. Only required for publishing releases.

* `github_api_url`: *Optional.* If you use a non-public GitHub deployment then
  you can set your API URL here.

## Behavior

### `check`: Check for released versions.

Releases are listed and sorted by their tag, using
[semver](http://semver.org) semantics if possible.

### `in`: Fetch assets from a release.

Fetches artifacts from the given release version. If the version is not
specified, the latest version is chosen using [semver](http://semver.org)
semantics.

Creates a file `tag` containing the tag name/version of the release being fetched.

#### Parameters

* `globs`: *Optional.* A list of globs for files that will be downloaded from
  the release. If not specified, all assets will be fetched.

### `out`: Publish a release.

Given a name specified in `name`, a body specified in `body`, and the tag to use
specified in `tag`, this creates a release on GitHub then uploads the files
matching the patterns in `globs` to the release.

#### Parameters

* `name`: *Required.* A path to a file containing the name of the release.

* `tag`: *Required.* A path to a file containing the name of the Git tag to use
  for the release.

* `body`: *Optional.* A path to a file containing the body text of the release.

* `globs`: *Optional.* A list of globs for files that will be uploaded alongside
  the created release.
