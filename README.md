# GitHub Releases Resource

Fetches and creates versioned GitHub resources.

> If you're seeing rate limits affecting you then please add a token to the source
> configuration. This will increase your number of allowed requests.

## Source Configuration

* `user`: *Required.* The GitHub username or organization name for the
  repository that the releases are in.

* `repository`: *Required.* The repository name that contains the releases.

* `access_token`: *Optional.* Used for accessing a release in a private-repo
   during an `in` and pushing a release to a repo during an `out`. The access
   token you create is only required to have the `repo` or `public_repo` scope.

* `github_api_url`: *Optional.* If you use a non-public GitHub deployment then
  you can set your API URL here.

* `github_uploads_url`: *Optional.* Some GitHub instances have a separate URL
  for uploading. If `github_api_url` is set, this value defaults to the same
  value, but if you have your own endpoint, this field will override it.

* `release`: *Optional. Default `true`.* When set to `true`, `put` produces
  release and `check` will detects releases.  If `false`, `put` and `check` will ignore releases.
  Note that releases must have semver compliant tags to be detected.

* `pre_release`: *Optional. Default `false`.* When set to `true`, `put` produces
  pre-release and `check` detects prereleases. If `false`, only non-prerelease releases
  will be detected and published. Note that releases must have semver compliant
  tags to be detected.
  If `release` and `pre_release` are set to `true` `put` produces
  release and `check` will detects prereleases and releases.

* `drafts`: *Optional. Default `false`.* When set to `true`, `put` produces
  drafts and `check` only detects drafts. If `false`, only non-draft releases
  will be detected and published. Note that releases must have semver compliant
  tags to be detected, even if they're drafts.

* `version`: *Optional* If specified, will check all releases from the specified version to latest. Otherwise, will check only the latest release.

### Example

``` yaml
- name: gh-release
  type: github-release
  source:
    user: concourse
    repository: concourse
    access_token: abcdef1234567890
```

``` yaml
- name: gh-release
  type: github-release
  source:
    user: concourse
    repository: concourse
    access_token: abcdef1234567890
    version: { tag: 'v0.0.1' }
```

``` yaml
- get: gh-release
```

``` yaml
- put: gh-release
  params:
    name: path/to/name/file
    tag: path/to/tag/file
    body: path/to/body/file
    globs:
    - paths/to/files/to/upload-*.tgz
```

To get a specific version of a release:

``` yaml
- get: gh-release
  version: { tag: 'v0.0.1' }
```

## Behavior

### `check`: Check for released versions.

Releases are listed and sorted by their tag, using
[semver](http://semver.org) semantics if possible. If `version` is specified, `check` returns releases from the specified version on. Otherwise, `check` returns the latest release.

### `in`: Fetch assets from a release.

Fetches artifacts from the given release version. If the version is not
specified, the latest version is chosen using [semver](http://semver.org)
semantics.

Also creates the following files:

* `tag` containing the git tag name of the release being fetched.
* `version` containing the version determined by the git tag of the release being fetched.
* `body` containing the body text of the release.

#### Parameters

* `globs`: *Optional.* A list of globs for files that will be downloaded from
  the release. If not specified, all assets will be fetched.

* `include_source_tarball`: *Optional.* Enables downloading of the source
  artifact tarball for the release as `source.tar.gz`. Defaults to `false`.

* `include_source_zip`: *Optional.* Enables downloading of the source
  artifact zip for the release as `source.zip`. Defaults to `false`.

### `out`: Publish a release.

Given a name specified in `name`, a body specified in `body`, and the tag to use
specified in `tag`, this creates a release on GitHub then uploads the files
matching the patterns in `globs` to the release.

#### Parameters

* `name`: *Required.* A path to a file containing the name of the release.

* `tag`: *Required.* A path to a file containing the name of the Git tag to use
  for the release.

* `tag_prefix`: *Optional.*  If specified, the tag read from the file will be
prepended with this string. This is useful for adding v in front of version numbers.

* `commitish`: *Optional.* A path to a file containing the commitish (SHA, tag,
  branch name) that the release should be associated with.

* `body`: *Optional.* A path to a file containing the body text of the release.

* `globs`: *Optional.* A list of globs for files that will be uploaded alongside
  the created release.
