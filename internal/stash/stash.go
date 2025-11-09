package stash

import (
	"context"
	"fmt"
	"net/http"

	"github.com/shurcooL/graphql"
)

type StashQlClient struct {
	Endpoint string
	APIKey   string
	cl       graphql.Client
}

func (st *StashQlClient) FindSceneByHash(ctx context.Context, hash string) (*Scene, error) {
	var Query struct {
		// The struct tag maps the Go struct field 'Scene' to the GraphQL field
		// 'findSceneByHash' and includes the argument 'input: { oshash: $hash }'.
		// $hash is the placeholder for the dynamic variable.
		Scene Scene `graphql:"findSceneByHash(input: { oshash: $hash })"`
	}
	variables := map[string]interface{}{
		// The key must match the variable name used in the struct tag (e.g., $hash)
		"hash": graphql.String(hash),
	}
	if err := st.cl.Query(ctx, &Query, variables); err != nil {
		return nil, fmt.Errorf("can not find scene by hash: %w", err)
	}
	if Query.Scene.ID == "" {
		return nil, fmt.Errorf("scene not found")
	}
	return &Query.Scene, nil
}
func (st *StashQlClient) FindSceneById(ctx context.Context, id string) (*Scene, error) {
	var Query struct {
		// The struct tag maps the Go struct field 'Scene' to the GraphQL field
		// 'findSceneByHash' and includes the argument 'input: { oshash: $hash }'.
		// $hash is the placeholder for the dynamic variable.
		Scene Scene `graphql:"findScene(id: $id)"`
	}
	variables := map[string]interface{}{
		// The key must match the variable name used in the struct tag (e.g., $hash)
		"id": graphql.ID(id),
	}
	if err := st.cl.Query(ctx, &Query, variables); err != nil {
		return nil, fmt.Errorf("can not find scene by hash: %w", err)
	}
	if Query.Scene.ID == "" {
		return nil, fmt.Errorf("scene not found")
	}
	return &Query.Scene, nil
}
func NewStashQlClient(endpoint string, apiKey string) *StashQlClient {
	httpCl := http.DefaultClient
	if apiKey != "" {
		httpCl = &http.Client{
			Transport: &apiKeyTransport{
				apiKey: apiKey,
				base:   http.DefaultTransport,
			},
		}
	}
	graphqlEndpoint := fmt.Sprintf("%s/graphql", endpoint)
	return &StashQlClient{
		Endpoint: graphqlEndpoint,
		APIKey:   apiKey,
		cl:       *graphql.NewClient(graphqlEndpoint, httpCl),
	}
}

type apiKeyTransport struct {
	apiKey string
	base   http.RoundTripper
}

func (t *apiKeyTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	req.Header.Set("ApiKey", t.apiKey)
	return t.base.RoundTrip(req)
}
