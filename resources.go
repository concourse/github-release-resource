package resource

type Source struct {
	User       string `json:"user"`
	Repository string `json:"repository"`

	GitHubAPIURL     string `json:"github_api_url"`
	GitHubUploadsURL string `json:"github_uploads_url"`
	AccessToken      string `json:"access_token"`
	Drafts           bool   `json:"drafts"`
	PreRelease       bool   `json:"pre_release"`
}

type CheckRequest struct {
	Source  Source  `json:"source"`
	Version Version `json:"version"`
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
	NamePath      string `json:"name"`
	BodyPath      string `json:"body"`
	TagPath       string `json:"tag"`
	CommitishPath string `json:"commitish"`
	TagPrefix     string `json:"tag_prefix"`

	Globs []string `json:"globs"`
}

type OutResponse struct {
	Version  Version        `json:"version"`
	Metadata []MetadataPair `json:"metadata"`
}

type Version struct {
	Tag string `json:"tag,omitempty"`
	ID  string `json:"id,omitempty"`
}

type MetadataPair struct {
	Name     string `json:"name"`
	Value    string `json:"value"`
	URL      string `json:"url"`
	Markdown bool   `json:"markdown"`
}
