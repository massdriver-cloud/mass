// Package gqlmock provides utilities for mocking GraphQL HTTP clients in tests.
package gqlmock

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"

	"github.com/Khan/genqlient/graphql"
)

// MockEndpoint is the default GraphQL endpoint path used by mock clients.
const MockEndpoint string = "/graphql"

// NewClient creates a GraphQL client backed by the given mux without a real HTTP connection.
func NewClient(mux *http.ServeMux) graphql.Client {
	return graphql.NewClient(MockEndpoint, &http.Client{Transport: localRoundTripper{handler: mux}})
}

// localRoundTripper is an http.RoundTripper that executes HTTP transactions
// by using handler directly, instead of going over an HTTP connection.
type localRoundTripper struct {
	handler http.Handler
}

func (l localRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	w := httptest.NewRecorder()
	l.handler.ServeHTTP(w, req)
	return w.Result(), nil
}

// MustMarshalJSON marshals v to JSON, panicking on error.
func MustMarshalJSON(v map[string]any) []byte {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return data
}

// MustUnmarshalJSON unmarshals JSON data into v, panicking on error.
func MustUnmarshalJSON(data []byte, v any) {
	err := json.Unmarshal(data, &v)
	if err != nil {
		panic(err)
	}
}

// MustWrite writes a string to w, panicking on error.
func MustWrite(w io.Writer, s string) {
	_, err := io.WriteString(w, s)
	if err != nil {
		panic(err)
	}
}

// MuxWithJSONResponseMap creates a mux that routes responses by GraphQL operation name.
func MuxWithJSONResponseMap(responses map[string]any) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc(MockEndpoint, func(w http.ResponseWriter, req *http.Request) {
		var parsedReq GraphQLRequest
		err := json.NewDecoder(req.Body).Decode(&parsedReq)
		_ = err

		response := responses[parsedReq.OperationName]
		data, _ := json.Marshal(response)
		MustWrite(w, string(data))
	})

	return mux
}

// NewClientWithJSONResponseMap creates a GraphQL client that returns responses keyed by operation name.
func NewClientWithJSONResponseMap(responses map[string]any) graphql.Client {
	mux := MuxWithJSONResponseMap(responses)
	client := NewClient(mux)
	return client
}

// MuxWithJSONResponseArray creates a mux that returns responses from the array in order.
func MuxWithJSONResponseArray(responses []any) *http.ServeMux {
	mux := http.NewServeMux()
	counter := 0
	mux.HandleFunc(MockEndpoint, func(w http.ResponseWriter, _ *http.Request) {
		response := responses[counter]
		counter++
		data, _ := json.Marshal(response)
		MustWrite(w, string(data))
	})

	return mux
}

// MuxWithJSONResponse creates a mux that always responds with the given JSON map.
func MuxWithJSONResponse(response map[string]any) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc(MockEndpoint, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		data, _ := json.Marshal(response)
		MustWrite(w, string(data))
	})

	return mux
}

// ParseInputVariables parses graphql input variables from request and returns a JSON-ish map
func ParseInputVariables(req *http.Request) map[string]any {
	var parsedReq GraphQLRequest
	if err := json.NewDecoder(req.Body).Decode(&parsedReq); err != nil {
		panic(err)
	}

	return parsedReq.Variables
}

// ResponseFunc is a function type that produces a response from an HTTP request.
type ResponseFunc func(req *http.Request) any

// MuxWithFuncResponseArray creates a mux that invokes each ResponseFunc in order per request.
func MuxWithFuncResponseArray(responses []ResponseFunc) *http.ServeMux {
	mux := http.NewServeMux()
	counter := 0
	mux.HandleFunc(MockEndpoint, func(w http.ResponseWriter, req *http.Request) {
		handler := responses[counter]
		response := handler(req)
		counter++
		data, _ := json.Marshal(response)
		MustWrite(w, string(data))
	})

	return mux
}

// NewClientWithFuncResponseArray creates a GQL client that will respond with a series of results of handler functions
func NewClientWithFuncResponseArray(responses []ResponseFunc) graphql.Client {
	mux := MuxWithFuncResponseArray(responses)
	client := NewClient(mux)
	return client
}

// NewClientWithSingleJSONResponse creates a GraphQL client that always returns the given JSON map.
func NewClientWithSingleJSONResponse(response map[string]any) graphql.Client {
	mux := MuxWithJSONResponse(response)
	client := NewClient(mux)
	return client
}

// NewClientWithJSONResponseArray creates a GraphQL client that returns responses from the array in order.
func NewClientWithJSONResponseArray(responses []any) graphql.Client {
	mux := MuxWithJSONResponseArray(responses)
	client := NewClient(mux)
	return client
}

// GraphQLRequest represents the JSON body of an incoming GraphQL HTTP request.
type GraphQLRequest struct {
	OperationName string         `json:"operationName"`
	Query         string         `json:"query"`
	Variables     map[string]any `json:"variables"`
}

// MockQueryResponse builds a QueryResponse with the given data keyed by operation name.
func MockQueryResponse(operationName string, responseData any) QueryResponse {
	r := QueryResponse{
		Data: map[string]any{},
	}

	r.Data[operationName] = responseData

	return r
}

// MockMutationResponse creates a successful mutation response from a mutation name and payload
func MockMutationResponse(operationName string, result any) MutationResponse {
	r := MutationResponse{
		Data: map[string]MutationResponseData{},
	}
	r.Data[operationName] = MutationResponseData{
		Successful: true,
		Result:     result,
	}
	return r
}

// QueryResponse represents a GraphQL query response envelope.
type QueryResponse struct {
	Data map[string]any `json:"data"`
}

// MutationResponseMessage holds a single message from a mutation response.
type MutationResponseMessage struct {
	Message string `json:"message"`
}

// MutationResponseData holds the result and messages for a single mutation operation.
type MutationResponseData struct {
	Successful bool                      `json:"successful"`
	Result     any                       `json:"result"`
	Messages   []MutationResponseMessage `json:"messages"`
}

// MutationResponse represents a GraphQL mutation response envelope.
type MutationResponse struct {
	Data map[string]MutationResponseData `json:"data"`
}
