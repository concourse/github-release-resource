# GitHub Releases Resource

Fetches and creates versioned GitHub resources.

> If you're seeing rate limits affecting you then please add a token to the source
> configuration. This will increase your number of allowed requests.

## Source Configuration

* `owner`: *Required.* The GitHub user or organization name for the repository
  that the releases are in.

* `repository`: *Required.* The repository name that contains the releases.

* `access_token`: *Optional.* Used for accessing a release in a private-repo
   during an `in` and pushing a release to a repo during an `out`. The access
   token you create is only required to have the `repo` or `public_repo` scope.

* `github_api_url`: *Optional.* If you use a non-public GitHub deployment then
  you can set your API URL here.
  
* `github_v4_api_url`: *Optional.* If you use a non-public GitHub deployment then
  you can set your API URL for graphql calls here.

* `github_uploads_url`: *Optional.* Some GitHub instances have a separate URL
  for uploading. If `github_api_url` is set, this value defaults to the same
  value, but if you have your own endpoint, this field will override it.

* `insecure`: *Optional. Default `false`.* When set to `true`, concourse will allow
  insecure connection to your github API.

* `release`: *Optional. Default `true`.* When set to `true`, `check` detects
  final releases and `put` publishes final releases (as opposed to
  pre-releases). If `false`, `check` will ignore final releases, and `put` will
  publish pre-releases if `pre_release` is set to `true`

* `pre_release`: *Optional. Default `false`.* When set to `true`, `check`
  detects pre-releases, and `put` will produce pre-releases (if `release` is
  also set to `false`). If `false`, only non-prerelease releases will be detected
  and published.

  **note:** if both `release` and `pre_release` are set to `true`, `put`
  produces final releases and `check` detects both pre-releases and releases. In
  order to produce pre-releases, you must set `pre_release` to `true` and
  `release` to `false`.  
  **note:** if both `release` and `pre_release` are set to `false`, `put` will
  still produce final releases.  
  **note:** releases must have [semver compliant](https://semver.org/#backusnaur-form-grammar-for-valid-semver-versions) tags to be detected.

* `drafts`: *Optional. Default `false`.* When set to `true`, `put` produces
  drafts and `check` only detects drafts. If `false`, only non-draft releases
  will be detected and published. Note that releases must have [semver compliant](https://semver.org/#backusnaur-form-grammar-for-valid-semver-versions)
  tags to be detected, even if they're drafts.

* `semver_constraint`: *Optional.* If set, constrain the returned semver tags according
  to a semver constraint, e.g. `"~1.2.x"`, `">= 1.2 < 3.0.0 || >= 4.2.3"`.
  Follows the rules outlined in https://github.com/Masterminds/semver#checking-version-constraints.

* `tag_filter`: *Optional.* If set, override default tag filter regular
  expression of `v?([^v].*)`. If the filter includes a capture group, the capture
  group is used as the release version; otherwise, the entire matching substring
  is used as the version.

* `order_by`: *Optional. One of [`version`, `time`]. Default `version`.*
   Selects whether to order releases by version (as extracted by `tag_filter`)
   or by time. See `check` behavior described below for details.

* `asset_dir`:  *Optional. Default `false`.* When set to `true`, downloaded assets
  will be created in a separate directory called `assets`. Otherwise, they will be
  created in the same directory as the other files.

### Example

``` yaml
- name: gh-release
  type: github-release
  source:
    owner: concourse
    repository: concourse
    access_token: abcdef1234567890
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
    generate_release_notes: true
```

To get a specific version of a release:

``` yaml
- get: gh-release
  version: { tag: 'v0.0.1' }
```

To set a custom tag filter:

```yaml
- name: gh-release
  type: github-release
  source:
    owner: concourse
    repository: concourse
    tag_filter: "version-(.*)"
```

## Behavior

### `check`: Check for released versions.

Lists releases, sorted either by their version or time, depending on the `order_by` source option.

When sorting by version, the version is extracted from the git tag using the `tag_filter` source option.
Versions are compared using [semver](http://semver.org) semantics if possible.

When sorting by time and a release is published, it uses the publication time, otherwise it uses the creation time.

The returned list contains an object of the following format for each release (with timestamp in the RFC3339 format):

```
{
    "id": "12345",
    "tag": "v1.2.3",
    "timestamp": "2006-01-02T15:04:05.999999999Z"
}
```

When `check` is given such an object as the `version` parameter, it returns releases from the specified version or time on.
Otherwise it returns the release with the latest version or time.

### `in`: Fetch assets from a release.

Fetches artifacts from the requested release.  If `asset_dir` source param is set to `true`,
artifacts will be created in a subdirectory called `assets`.

Also creates the following files:

* `tag` containing the git tag name of the release being fetched.
* `version` containing the version determined by the git tag of the release being fetched.
* `body` containing the body text of the release.
* `timestamp` containing the publish or creation timestamp for the release in RFC 3339 format.
* `commit_sha` containing the commit SHA the tag is pointing to.
* `url` containing the HTMLURL for the release being fetched.

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

* `generate_release_notes`: *Optional.* Causes GitHub to autogenerate the release notes
  when creating a new release, based on the commits since the last release.
  If `body` is specified, the body will be pre-pended to the automatically generated
  notes. Has no effect when updating an existing release. Defaults to `false`.

## Development

### Prerequisites

* golang is *required* - version 1.15.x is tested; earlier versions may also
  work.
* docker is *required* - version 17.06.x is tested; earlier versions may also
  work.

### Running the tests

The tests have been embedded with the `Dockerfile`; ensuring that the testing
environment is consistent across any `docker` enabled platform. When the docker
image builds, the test are run inside the docker container, on failure they
will stop the build.

Run the tests with the following command:

```sh
docker build -t github-release-resource --target tests .
```

### Contributing

Please make all pull requests to the `master` branch and ensure tests pass
locally.
