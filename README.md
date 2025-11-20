# GitHub Releases Resource

Fetches and creates versioned GitHub resources.

<a href="https://ci.concourse-ci.org/teams/main/pipelines/resource/jobs/build?vars.type=%22github-release%22">
  <img src="https://ci.concourse-ci.org/api/v1/teams/main/pipelines/resource/jobs/build/badge?vars.type=%22github-release%22" alt="Build Status">
</a>


> If you're seeing rate limits affecting you then please add a token to the source
> configuration. This will increase your number of allowed requests.

## Source Configuration

<table>
  <thead>
    <tr>
      <th>Field Name</th>
      <th>Description</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td><code>owner</code> (Required)</td>
      <td> The GitHub user or organization name for the repository that the releases are in.</td>
    </tr>
    <tr>
      <td><code>repository</code> (Required)</td>
      <td>The repository name that contains the releases.</td>
    </tr>
    <tr>
      <td><code>access_token</code> (Optional)</td>
      <td>
        Used for accessing a release in a private-repo during an <code>in</code> and pushing a
        release to a repo during an <code>out</code>. The
        <a href="https://github.com/settings/personal-access-tokens">fine-grained access token</a> you create is only
        required to have the <code>content</code> permission. For classic access tokens, you need the
        <code>repo</code> or <code>public_repo</code> permission.
      </td>
    </tr>
    <tr>
      <td><code>github_api_url</code> (Optional)</td>
      <td>If you use a non-public GitHub deployment then you can set your API URL here.</td>
    </tr>
    <tr>
      <td><code>github_v4_api_url</code> (Optional)</td>
      <td>
        If you use a non-public GitHub deployment then you can set your API URL for graphql calls
        here.
      </td>
    </tr>
    <tr>
      <td><code>github_uploads_url</code> (Optional)</td>
      <td>
        Some GitHub instances have a separate URL for uploading. If <code>github_api_url</code> is
        set, this value defaults to the same value, but if you have your own endpoint, this field will override it.
      </td>
    </tr>
    <tr>
      <td><code>insecure</code> (Optional)</td>
      <td>
        Defaults to <code>false</code>. When set to <code>true</code>, concourse will allow insecure
        connection to your github API.
      </td>
    </tr>
    <tr>
      <td><code>release</code> (Optional)</td>
      <td>
        Defaults to <code>true</code>. When set to <code>true</code>, <code>check</code> detects final
        releases and <code>put</code> publishes final releases (as opposed to pre-releases). If <code>false</code>,
        <code>check</code> will ignore final releases, and <code>put</code> will publish pre-releases if
        <code>pre_release</code> is set to <code>true</code>
      </td>
    </tr>
    <tr>
      <td><code>pre_release</code> (Optional)</td>
      <td>
        Defaults to <code>false</code>. When set to <code>true</code>, <code>check</code> detects
        pre-releases, and <code>put</code> will produce pre-releases (if <code>release</code> is also set to
        <code>false</code>). If <code>false</code>, only non-prerelease releases will be detected and published.
        <br/><br/>
        <strong>NOTE:</strong>
        If both <code>release</code> and <code>pre_release</code> are set to <code>true</code>,
        <code>put</code> produces final releases and <code>check</code> detects both pre-releases and releases. In order
        to produce pre-releases, you must set <code>pre_release</code> to <code>true</code> and <code>release</code> to
        <code>false</code>.<br /><strong>note:</strong> if both <code>release</code> and <code>pre_release</code> are
        set to <code>false</code>, <code>put</code> will still produce final releases.<br /><strong>note:</strong>
        releases must have
        <a href="https://semver.org/#backusnaur-form-grammar-for-valid-semver-versions">semver compliant</a>
        tags to be detected.
      </td>
    </tr>
    <tr>
      <td><code>drafts</code> (Optional)</td>
      <td>
        Defaults to <code>false</code>. When set to <code>true</code>, <code>put</code> produces drafts
        and <code>check</code> only detects drafts. If <code>false</code>, only non-draft releases will be detected and
        published. Note that releases must have
        <a href="https://semver.org/#backusnaur-form-grammar-for-valid-semver-versions">semver compliant</a>
        tags to be detected, even if they're drafts.
      </td>
    </tr>
    <tr>
      <td><code>semver_constraint</code> (Optional)</td>
      <td>
        If set, constrain the returned semver tags according to a semver constraint, e.g.
        <code>"~1.2.x"</code>, <code>">= 1.2 < 3.0.0 || >= 4.2.3"</code>. Follows the rules outlined in
        <a href="https://github.com/Masterminds/semver#checking-version-constraints"
          >https://github.com/Masterminds/semver#checking-version-constraints</a
        >.
      </td>
    </tr>
    <tr>
      <td><code>tag_filter</code> (Optional)</td>
      <td>
        If set, override default tag filter regular expression of <code>v?([^v].*)</code>. If the
        filter includes a capture group, the capture group is used as the release version; otherwise, the entire
        matching substring is used as the version. You can test your regex in the <a href="https://go.dev/play/p/shzMfC-rfI-">Go Playground</a>.
      </td>
    </tr>
    <tr>
      <td><code>order_by</code> (Optional)</td>
      <td>
        One of [<code>version</code>, <code>time</code>]. Defaults to <code>version</code>.
        Selects whether to order releases by version (as extracted by <code>tag_filter</code>) or by time. See
        <code>check</code> behavior described below for details.
      </td>
    </tr>
    <tr>
      <td><code>asset_dir</code> (Optional)</td>
      <td>
        Default <code>false</code>. When set to <code>true</code>, downloaded assets will be created
        in a separate directory called <code>assets</code>. Otherwise, they will be created in the same directory as the
        other files.
      </td>
    </tr>
  </tbody>
</table>

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

```json
{
    "id": "12345",
    "tag": "v1.2.3",
    "timestamp": "2006-01-02T15:04:05.999999999Z"
}
```

When `check` is given such an object as the `version` parameter, it returns releases from the specified version or time on.
Otherwise it returns the release with the latest version or time.

### `get`: Fetch assets from a release.

Fetches artifacts from the requested release.  If `asset_dir` source param is set to `true`,
artifacts will be created in a subdirectory called `assets`.

Also creates the following files:

* `tag` containing the git tag name of the release being fetched.
* `version` containing the version determined by the git tag of the release being fetched. If a capture group is used in `tag_filter` then this will be the value of the capture group.
* `body` containing the body text of the release.
* `timestamp` containing the publish or creation timestamp for the release in RFC 3339 format.
* `commit_sha` containing the commit SHA the tag is pointing to.
* `url` containing the HTMLURL for the release being fetched.

#### Parameters

<table>
  <thead>
    <tr>
      <th>Field Name</th>
      <th>Description</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td><code>globs</code> (Optional)</td>
      <td>A list of globs for files that will be downloaded from the release. If
      not specified, all assets will be fetched.</td>
    </tr>
    <tr>
      <td><code>include_source_tarball</code> (Optional)</td>
      <td>Enables downloading of the source artifact tarball for the release as
      <code>source.tar.gz</code>. Defaults to <code>false</code>.</td>
    </tr>
    <tr>
      <td><code>include_source_zip</code> (Optional)</td>
      <td>Enables downloading of the source artifact zip for the release as
      <code>source.zip</code>. Defaults to <code>false</code>.</td>
    </tr>
  </tbody>
</table>

### `put`: Publish a release.

Given a name specified in `name`, a body specified in `body`, and the tag to use
specified in `tag`, this creates a release on GitHub then uploads the files
matching the patterns in `globs` to the release.

#### Parameters

<table>
  <thead>
    <tr>
      <th>Field Name</th>
      <th>Description</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td><code>name</code> (Required)</td>
      <td>A path to a file containing the name of the release.</td>
    </tr>
    <tr>
      <td><code>tag</code> (Required)</td>
      <td>A path to a file containing the name of the Git tag to use for the release.</td>
    </tr>
    <tr>
      <td><code>tag_prefix</code> (Optional)</td>
      <td>If specified, the tag read from the file will be prepended with this
      string. This is useful for adding v in front of version numbers.</td>
    </tr>
    <tr>
      <td><code>commitish</code> (Optional)</td>
      <td>A path to a file containing the commitish (SHA, tag, branch name) that
      the release should be associated with.</td>
    </tr>
    <tr>
      <td><code>body</code> (Optional)</td>
      <td>A path to a file containing the body text of the release.</td>
    </tr>
    <tr>
      <td><code>globs</code> (Optional)</td>
      <td>A list of globs for files that will be uploaded alongside the created release.</td>
    </tr>
    <tr>
      <td><code>generate_release_notes</code> (Optional)</td>
      <td>Causes GitHub to autogenerate the release notes when creating a new
      release, based on the commits since the last release. If <code>body</code>
      is specified, the body will be pre-pended to the automatically generated
      notes. Has no effect when updating an existing release. Defaults to
      <code>false</code>.</td>
    </tr>
  </tbody>
</table>

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
