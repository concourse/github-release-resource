package resource

import "github.com/shurcooL/githubv4"

// ReleaseObject represent the graphql release object
// https://developer.github.com/v4/object/release
type ReleaseObject struct {
	CreatedAt    githubv4.DateTime `graphql:"createdAt"`
	PublishedAt  githubv4.DateTime `graphql:"publishedAt"`
	ID           string            `graphql:"id"`
	DatabaseId   githubv4.Int      `graphql:"databaseId"`
	IsDraft      bool              `graphql:"isDraft"`
	IsPrerelease bool              `graphql:"isPrerelease"`
	Name         string            `graphql:"name"`
	TagName      string            `graphql:"tagName"`
	URL          string            `graphql:"url"`
}
