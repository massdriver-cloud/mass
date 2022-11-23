package main

import "github.com/massdriver-cloud/mass/pkg/api"

func main() {
	api.NewClient(api.Endpoint, "foo")
}
