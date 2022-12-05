package commands_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"

	"github.com/Khan/genqlient/graphql"
)

// TODO: backout this main_test and api/main_test into a api_test helper
const mockEndpoint string = "/graphql"

func mockClient(mux *http.ServeMux) graphql.Client {
	return graphql.NewClient(mockEndpoint, &http.Client{Transport: localRoundTripper{handler: mux}})
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

func mustMarshalJSON(v map[string]interface{}) []byte {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return data
}

func mustWrite(w io.Writer, s string) {
	_, err := io.WriteString(w, s)
	if err != nil {
		panic(err)
	}
}

func muxWithJSONResponse(response map[string]interface{}) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc(mockEndpoint, func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		data, _ := json.Marshal(response)
		mustWrite(w, string(data))
	})

	return mux
}

func muxWithJSONResponseMap(responses map[string]interface{}) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc(mockEndpoint, func(w http.ResponseWriter, req *http.Request) {
		var parsedReq graphQLRequest
		json.NewDecoder(req.Body).Decode(&parsedReq)

		response := responses[parsedReq.OperationName]
		data, _ := json.Marshal(response)
		mustWrite(w, string(data))
	})

	return mux
}

// Takes a map of graphql operation names to JSON responses and creates a GraphQL client that returns based on operation name
func mockClientWithJSONResponseMap(responses map[string]interface{}) graphql.Client {
	mux := muxWithJSONResponseMap(responses)
	client := mockClient(mux)
	return client
}

func muxWithJSONResponseArray(responses []interface{}) *http.ServeMux {
	mux := http.NewServeMux()
	counter := 0
	mux.HandleFunc(mockEndpoint, func(w http.ResponseWriter, req *http.Request) {
		response := responses[counter]
		counter++
		data, _ := json.Marshal(response)
		mustWrite(w, string(data))
	})

	return mux
}

// Takes an array of responses and creates a graphql client that returns them in order
func mockClientWithJSONResponseArray(responses []interface{}) graphql.Client {
	mux := muxWithJSONResponseArray(responses)
	client := mockClient(mux)
	return client
}

type graphQLRequest struct {
	OperationName string                 `json:"operationName"`
	Query         string                 `json:"query"`
	Variables     map[string]interface{} `json:"variables"`
}

type queryResponse struct {
	Data map[string]interface{} `json:"data"`
}

func mockQueryResponse(operationName string, responseData interface{}) queryResponse {
	r := queryResponse{
		Data: map[string]interface{}{},
	}

	r.Data[operationName] = responseData

	return r
}

func mockMutationResponse(operationName string, result interface{}) mutationResponse {
	r := mutationResponse{
		Data: map[string]mutationResponseData{},
	}
	r.Data[operationName] = mutationResponseData{
		Successful: true,
		Result:     result,
	}
	return r
}

type mutationResponseMessage struct {
	Message string `json:"message"`
}

type mutationResponseData struct {
	Successful bool                      `json:"successful"`
	Result     interface{}               `json:"result"`
	Messages   []mutationResponseMessage `json:"messages"`
}

type mutationResponse struct {
	Data map[string]mutationResponseData `json:"data"`
}
