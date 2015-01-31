# GitHub Releases Resource

Fetches and creates versioned GitHub resources.

## Source Configuration

* `access_token`: *Required.* The GitHub access token that should be used to
  access the API.

* `user`: *Required.* The GitHub username or organization name for the
  repository that the releases are in.

* `repository`: *Required.* The repository name that contains the releases.

## Behavior

### `check`: Extract versions from the bucket.

*TODO*

### `in`: Fetch an object from the bucket.

*TODO*

### `out`: Upload an object to the bucket.

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
