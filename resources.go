package resource

import (
	"time"
)

type Source struct {
	Owner      string `json:"owner"`
	Repository string `json:"repository"`

	// Deprecated; use Owner instead
	User string `json:"user"`

	GitHubAPIURL     string `json:"github_api_url"`
	GitHubV4APIURL   string `json:"github_v4_api_url"`
	GitHubUploadsURL string `json:"github_uploads_url"`
	AccessToken      string `json:"access_token"`
	Drafts           bool   `json:"drafts"`
	PreRelease       bool   `json:"pre_release"`
	Release          bool   `json:"release"`
	Insecure         bool   `json:"insecure"`
	AssetDir         bool   `json:"asset_dir"`

	TagFilter        string `json:"tag_filter"`
	OrderBy          string `json:"order_by"`
	SemverConstraint string `json:"semver_constraint"`
}

type CheckRequest struct {
	Source  Source  `json:"source"`
	Version Version `json:"version"`
}

func NewCheckRequest() CheckRequest {
	res := CheckRequest{}
	res.Source.Release = true
	return res
}

func NewOutRequest() OutRequest {
	res := OutRequest{}
	res.Source.Release = true
	return res
}

func NewInRequest() InRequest {
	res := InRequest{}
	res.Source.Release = true
	return res
}

type InRequest struct {
	Source  Source   `json:"source"`
	Version *Version `json:"version"`
	Params  InParams `json:"params"`
}

type InParams struct {
	Globs                []string `json:"globs"`
	IncludeSourceTarball bool     `json:"include_source_tarball"`
	IncludeSourceZip     bool     `json:"include_source_zip"`
}

type InResponse struct {
	Version  Version        `json:"version"`
	Metadata []MetadataPair `json:"metadata"`
}

type OutRequest struct {
	Source Source    `json:"source"`
	Params OutParams `json:"params"`
}

type OutParams struct {
	NamePath             string `json:"name"`
	BodyPath             string `json:"body"`
	TagPath              string `json:"tag"`
	CommitishPath        string `json:"commitish"`
	TagPrefix            string `json:"tag_prefix"`
	GenerateReleaseNotes bool   `json:"generate_release_notes"`

	Globs []string `json:"globs"`
}

type OutResponse struct {
	Version  Version        `json:"version"`
	Metadata []MetadataPair `json:"metadata"`
}

type Version struct {
	Tag       string    `json:"tag,omitempty"`
	ID        string    `json:"id"`
	Timestamp time.Time `json:"timestamp"`
}

type MetadataPair struct {
	Name     string `json:"name"`
	Value    string `json:"value"`
	URL      string `json:"url"`
	Markdown bool   `json:"markdown"`
}
