package resource

import "github.com/google/go-github/v66/github"

func metadataFromRelease(release *github.RepositoryRelease, commitSHA string) []MetadataPair {
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

	if release.TagName != nil {
		metadata = append(metadata, MetadataPair{
			Name:  "tag",
			Value: *release.TagName,
		})
	}

	if commitSHA != "" {
		metadata = append(metadata, MetadataPair{
			Name:  "commit_sha",
			Value: commitSHA,
		})
	}

	if *release.Draft {
		metadata = append(metadata, MetadataPair{
			Name:  "draft",
			Value: "true",
		})
	}

	if *release.Prerelease {
		metadata = append(metadata, MetadataPair{
			Name:  "pre-release",
			Value: "true",
		})
	}
	return metadata
}
