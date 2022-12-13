package gqlmock

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"

	"github.com/Khan/genqlient/graphql"
)

const MockEndpoint string = "/graphql"

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

func MustMarshalJSON(v map[string]interface{}) []byte {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return data
}

func MustUnmarshalJSON(data []byte, v any) {
	err := json.Unmarshal(data, &v)
	if err != nil {
		panic(err)
	}
}

func MustWrite(w io.Writer, s string) {
	_, err := io.WriteString(w, s)
	if err != nil {
		panic(err)
	}
}

func MuxWithJSONResponseMap(responses map[string]interface{}) *http.ServeMux {
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

// Takes a map of graphql operation names to JSON responses and creates a GraphQL client that returns based on operation name
func NewClientWithJSONResponseMap(responses map[string]interface{}) graphql.Client {
	mux := MuxWithJSONResponseMap(responses)
	client := NewClient(mux)
	return client
}

func MuxWithJSONResponseArray(responses []interface{}) *http.ServeMux {
	mux := http.NewServeMux()
	counter := 0
	mux.HandleFunc(MockEndpoint, func(w http.ResponseWriter, req *http.Request) {
		response := responses[counter]
		counter++
		data, _ := json.Marshal(response)
		MustWrite(w, string(data))
	})

	return mux
}

func MuxWithJSONResponse(response map[string]interface{}) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc(MockEndpoint, func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		data, _ := json.Marshal(response)
		MustWrite(w, string(data))
	})

	return mux
}

// Takes a JSON map and creates a GraphQL client that always returns it
func NewClientWithSingleJSONResponse(response map[string]interface{}) graphql.Client {
	mux := MuxWithJSONResponse(response)
	client := NewClient(mux)
	return client
}

// Takes an array of responses and creates a graphql client that returns them in order
func NewClientWithJSONResponseArray(responses []interface{}) graphql.Client {
	mux := MuxWithJSONResponseArray(responses)
	client := NewClient(mux)
	return client
}

type GraphQLRequest struct {
	OperationName string                 `json:"operationName"`
	Query         string                 `json:"query"`
	Variables     map[string]interface{} `json:"variables"`
}

func MockQueryResponse(operationName string, responseData interface{}) QueryResponse {
	r := QueryResponse{
		Data: map[string]interface{}{},
	}

	r.Data[operationName] = responseData

	return r
}

func MockMutationResponse(operationName string, result interface{}) MutationResponse {
	r := MutationResponse{
		Data: map[string]MutationResponseData{},
	}
	r.Data[operationName] = MutationResponseData{
		Successful: true,
		Result:     result,
	}
	return r
}

type QueryResponse struct {
	Data map[string]interface{} `json:"data"`
}

type MutationResponseMessage struct {
	Message string `json:"message"`
}

type MutationResponseData struct {
	Successful bool                      `json:"successful"`
	Result     interface{}               `json:"result"`
	Messages   []MutationResponseMessage `json:"messages"`
}

type MutationResponse struct {
	Data map[string]MutationResponseData `json:"data"`
}
