package stash

import "github.com/hasura/go-graphql-client"

// File represents the 'files' field inside the scene.
type File struct {
	Basename string
}

// Scene represents the main object returned by findSceneByHash.
type Scene struct {
	ID    graphql.ID
	Files []File
}
