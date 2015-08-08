package resource

import "github.com/zachgersh/go-github/github"

func metadataFromRelease(release *github.RepositoryRelease) []MetadataPair {
	metadata := []MetadataPair{}

	if release.Name != nil {
		nameMeta := MetadataPair{
			Name:  "name",
			Value: *release.Name,
		}

		if release.HTMLURL != nil {
			nameMeta.URL = *release.HTMLURL
		}

		metadata = append(metadata, nameMeta)
	}

	if release.HTMLURL != nil {
		metadata = append(metadata, MetadataPair{
			Name:  "url",
			Value: *release.HTMLURL,
		})
	}

	if release.Body != nil {
		metadata = append(metadata, MetadataPair{
			Name:     "body",
			Value:    *release.Body,
			Markdown: true,
		})
	}

	return metadata
}
