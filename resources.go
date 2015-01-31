package resource

type OutRequest struct {
	Source OutSource `json:"source"`
	Params OutParams `json:"params"`
}

type OutSource struct {
	AccessToken string `json:"access_token"`

	User       string `json:"user"`
	Repository string `json:"repository"`
}

type OutParams struct {
	NamePath string `json:"name"`
	BodyPath string `json:"body"`
	TagPath  string `json:"tag"`

	Globs []string `json:"globs"`
}

type OutResponse struct {
	Version  Version        `json:"version"`
	Metadata []MetadataPair `json:"metadata"`
}

type Version struct {
	Tag string `json:"tag"`
}

type MetadataPair struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}
