// Package api is a temporary holding pen for GraphQL operations that the
// massdriver-sdk-go doesn't expose yet. Today this is just the resource-type
// surface (Get / List / Publish / Delete). When the SDK grows native support
// the corresponding files here disappear; once the package is empty, delete it.
package api

import (
	"fmt"

	"github.com/Khan/genqlient/graphql"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/gql"
)

// transportOverride is set by tests to short-circuit transport construction
// (we can't reach inside *massdriver.Client to get its graphql client, so
// tests need their own injection point). Production code leaves it nil and
// gqlClient builds a real transport from the resolved config.
var transportOverride graphql.Client

// SetTransportForTest installs a graphql.Client that every api operation will
// use instead of the configured Massdriver transport. Tests pair this with
// gqltest.NewClient and t.Cleanup to scrub on teardown.
func SetTransportForTest(c graphql.Client) func() {
	transportOverride = c
	return func() { transportOverride = nil }
}

// gqlClient builds a v2-shape GraphQL client from a *massdriver.Client's
// resolved config. Each call reconstructs the transport — cheap, and avoids
// stashing state in this package.
func gqlClient(mdClient *massdriver.Client) graphql.Client {
	if transportOverride != nil {
		return transportOverride
	}
	return gql.NewV2Client(mdClient.Config())
}

// mutationMessage is the per-field message bag returned by GraphQL mutations.
type mutationMessage struct {
	Code    string `json:"code"`
	Field   string `json:"field"`
	Message string `json:"message"`
}

// mutationError formats one or more mutation messages into a single error
// matching the legacy CLI's user-facing output.
func mutationError(label string, messages []mutationMessage) error {
	if len(messages) == 0 {
		return fmt.Errorf("%s: server reported failure with no detail", label)
	}
	out := label + ":"
	for _, m := range messages {
		out += "\n  - "
		if m.Field != "" {
			out += m.Field + ": "
		}
		out += m.Message
		if m.Code != "" {
			out += " (" + m.Code + ")"
		}
	}
	return fmt.Errorf("%s", out)
}
