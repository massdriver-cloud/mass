package container

import (
	"bufio"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/moby/moby/client"
	"nhooyr.io/websocket"
)

// List lists containers
//
//	@Summary		List containers
//	@Description	List containers searches using the name param, defaults to 'mass' if none provided.
//	@ID				list-containers
//	@Produce		json
//	@Param			all		query	bool	false	"all containers, even stopped"				default(false)
//	@Param			limit	query	int		false	"number of containers to return, 0 is all"	default(0)
//	@Param			name	query	string	false	"name of container to search with"			default(mass)
//	@Success		200		{array}	types.Container
//	@Router			/containers/list [get]
func List(w http.ResponseWriter, r *http.Request) {
	queries := r.URL.Query()
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithVersion("1.41"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var listOpts types.ContainerListOptions
	if queries.Get("all") == "true" {
		listOpts.All = true
	}
	if queries.Get("limit") != "" {
		l, atoiErr := strconv.Atoi(queries.Get("limit"))
		if atoiErr != nil {
			http.Error(w, atoiErr.Error(), http.StatusBadRequest)
			return
		}
		listOpts.Limit = l
	}

	// Our default name search so the UI isn't required to provide it
	nameValue := "mass"
	if queries.Get("name") != "" {
		nameValue = queries.Get("name")
	}

	listOpts.Filters = filters.NewArgs(filters.KeyValuePair{Key: "name", Value: nameValue})

	containers, err := cli.ContainerList(ctx, listOpts)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	c, err := json.Marshal(containers)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(c)
	if err != nil {
		slog.Error(err.Error())
	}
}

// StreamLogs opens a websocket to stream the logs of a container
//
//	@Summary		Stream logs
//	@Description	Stream the logs from a container using a websocket
//	@ID				stream-logs
//	@Produce		plain
//	@Param			id	query	string	true	"id of the container"
//	@Success		101
//	@Router			/containers/logs [get]
func StreamLogs(w http.ResponseWriter, r *http.Request) {
	containerID := r.URL.Query().Get("id")
	if containerID == "" {
		http.Error(w, "query param 'id' is required", http.StatusBadRequest)
		return
	}

	// Just in case....
	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithVersion("1.41"))
	if err != nil {
		slog.Error(err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	reader, err := cli.ContainerLogs(ctx, containerID, types.ContainerLogsOptions{
		ShowStderr: true,
		ShowStdout: true,
		Timestamps: true,
		Follow:     true,
		Tail:       "50",
	})
	if err != nil {
		slog.Error(err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	defer func() {
		slog.Debug("closing reader")
		if readerErr := reader.Close(); readerErr != nil {
			slog.Error("readerErr", "error", readerErr)
		}
	}()

	wc, err := websocket.Accept(w, r, &websocket.AcceptOptions{})
	if err != nil {
		// Don't write an error to the writer, websocket handles that
		// internally
		slog.Error(err.Error())
		return
	}

	closectx := wc.CloseRead(ctx)

	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		if closectx.Err() != nil {
			slog.Info("Websocket connection closed")
			return
		}
		err = wc.Write(ctx, websocket.MessageText, []byte(scanner.Text()))
		if err != nil {
			slog.Error("Websocket write error", "error", err.Error())
			break
		}
	}

	if ctx.Err() != nil {
		slog.Debug("context error", "error", ctx.Err().Error())
	}

	wc.Close(websocket.StatusNormalClosure, "")
}
