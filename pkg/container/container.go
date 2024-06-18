package container

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/strslice"
	"github.com/massdriver-cloud/mass/pkg/bundle"
	sb "github.com/massdriver-cloud/mass/pkg/server/bundle"
	"github.com/moby/moby/client"
	"github.com/moby/moby/pkg/jsonmessage"
	"github.com/moby/moby/pkg/namesgenerator"
	"github.com/moby/term"
	"nhooyr.io/websocket"
)

const allowedMethods = "OPTIONS, GET, POST"

type Handler struct {
	baseDir      string
	dockerCLI    *client.Client
	parsedBundle bundle.Bundle
}

type DeployPayload struct {
	Action  string            `json:"action"`
	Secrets map[string]string `json:"secrets,omitempty"`
	Params  map[string]any    `json:"params,omitempty"`
	Image   string            `json:"image,omitempty"`
}

type deployReply struct {
	ContainerID string `json:"containerID"`
}

func NewHandler(baseDir string, dockerCLI *client.Client) (*Handler, error) {
	b, err := bundle.Unmarshal(baseDir)

	if err != nil {
		return nil, err
	}

	return &Handler{
		baseDir:      baseDir,
		dockerCLI:    dockerCLI,
		parsedBundle: *b,
	}, nil
}

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
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	queries := r.URL.Query()
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

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
	nameValue := "mass_"
	if queries.Get("name") != "" {
		nameValue = queries.Get("name")
	}

	listOpts.Filters = filters.NewArgs(filters.KeyValuePair{Key: "name", Value: nameValue})

	containers, err := h.dockerCLI.ContainerList(ctx, listOpts)
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
func (h *Handler) StreamLogs(w http.ResponseWriter, r *http.Request) {
	containerID := r.URL.Query().Get("id")
	if containerID == "" {
		http.Error(w, "query param 'id' is required", http.StatusBadRequest)
		return
	}

	ctx := r.Context()

	reader, err := h.dockerCLI.ContainerLogs(ctx, containerID, types.ContainerLogsOptions{
		ShowStderr: true,
		ShowStdout: true,
		Timestamps: false,
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

	// InsecureSkipVerify is fine here since we are running locally.
	wc, err := websocket.Accept(w, r, &websocket.AcceptOptions{InsecureSkipVerify: true})
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
		b := scanner.Bytes()
		if json.Valid(b) {
			err = wc.Write(ctx, websocket.MessageText, b)
			if err != nil {
				slog.Error("Websocket write error", "error", err.Error())
				break
			}
		} else if !utf8.ValidString(string(b)) {
			// If a container is ran without TTY then the logs coming back from docker
			// can potentially have invalid utf-8 characters which won't pass the valid json
			// check above. At least warn so there is indication why nothing is coming through
			slog.Warn("Container log has invalid utf-8", "log", string([]rune(scanner.Text())))
		}
	}

	if scanner.Err() != nil {
		slog.Error("Error reading container logs", "error", scanner.Err().Error())
		wc.Close(websocket.StatusNormalClosure, "failed reading logs")
		return
	}

	if ctx.Err() != nil {
		slog.Debug("Context error", "error", ctx.Err().Error())
	}

	wc.Close(websocket.StatusNormalClosure, "")
}

// Deploy runs the provisioner container locally
//
//	@Summary		Deploy the bundle
//	@Description	Deploy runs the local provisioner to deploy the bundle
//	@ID				deploy-container
//	@Accept			json
//	@Success		200				{object}	container.deployReply
//	@Param			deployPayload	body		container.DeployPayload	true	"DeployPayload"
//	@Router			/bundle/deploy [post]
func (h *Handler) Deploy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodOptions {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	if r.Method == http.MethodOptions {
		h.options(w, r)
		return
	}

	conns, err := io.ReadAll(r.Body)
	if err != nil {
		slog.Debug("Error reading payload", "error", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	payload := DeployPayload{}

	err = json.Unmarshal(conns, &payload)
	if err != nil {
		slog.Debug("Error unmarshalling payload", "error", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	payload.Action = strings.ToLower(payload.Action)

	if payload.Action != "provision" && payload.Action != "decommission" {
		http.Error(w, "action must be either 'provision' or 'decommission'", http.StatusBadRequest)
		return
	}

	for _, step := range h.parsedBundle.Steps {
		inputHandler, _ := sb.NewInputHandler(step.Provisioner)
		basePath := path.Join(h.baseDir, step.Path)

		if len(payload.Secrets) != 0 {
			err = inputHandler.WriteSecrets(basePath, payload.Secrets)
			if err != nil {
				slog.Debug("Error writing secrets contents", "error", err)
				http.Error(w, "unable to write secrets file", http.StatusBadRequest)
				return
			}
		}

		if len(payload.Params) != 0 {
			if err = inputHandler.WriteParams(basePath, payload.Params); err != nil {
				slog.Debug("Error writing params contents", "error", err)
				http.Error(w, "unable to write params file", http.StatusBadRequest)
				return
			}
		}
	}

	image := "massdrivercloud/local-terraform-provisioner:latest"
	if payload.Image != "" {
		image = payload.Image
	}

	// Allow running of a local image without bombing out trying to pull it
	// The image string in the payload would be "local/my-dev-image:latest"
	if strings.HasPrefix(image, "local/") {
		image = strings.TrimPrefix(image, "local/")
	} else {
		err = h.pullImage(r.Context(), image)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}

	containerID, err := h.runContainer(r.Context(), payload.Action, image)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	out, err := json.Marshal(deployReply{ContainerID: containerID})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	_, err = w.Write(out)
	if err != nil {
		slog.Warn("failed to write response", "error", err)
	}
}

func (h *Handler) pullImage(ctx context.Context, image string) error {
	reader, err := h.dockerCLI.ImagePull(ctx, image, types.ImagePullOptions{})
	if err != nil {
		return err
	}
	defer reader.Close()

	termFd, isTerm := term.GetFdInfo(os.Stderr)
	return jsonmessage.DisplayJSONMessagesStream(reader, os.Stderr, termFd, isTerm, nil)
}

func (h *Handler) runContainer(ctx context.Context, action, image string) (string, error) {
	// TODO: This is aws only at this point
	envs, err := h.getAWSCreds(ctx)
	if err != nil {
		slog.Debug("Error getting aws creds", "error", err)
		return "", err
	}

	// This makes the container logs huge but debug doesn't come through events
	envs = append(envs, "TF_LOG=DEBUG")

	c := &container.Config{
		Cmd:   strslice.StrSlice{"run.sh", action},
		Image: image,
		Env:   envs,
		Tty:   true,
	}

	abs, err := filepath.Abs(h.baseDir)
	if err != nil {
		return "", err
	}

	host := &container.HostConfig{
		Binds: []string{abs + ":/bundle"},
	}

	// Prefix with mass_ for easy searching/deleting later
	name := "mass_" + namesgenerator.GetRandomName(0)

	response, err := h.dockerCLI.ContainerCreate(ctx, c, host, nil, nil, name)
	if err != nil {
		return "", err
	}

	err = h.dockerCLI.ContainerStart(ctx, response.ID, types.ContainerStartOptions{})

	return response.ID, err
}

// getAWSCreds returns a formatted slice of aws cred envvars suitable for a containers env.
func (h *Handler) getAWSCreds(ctx context.Context) ([]string, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, err
	}
	creds, err := cfg.Credentials.Retrieve(ctx)
	if err != nil {
		return nil, err
	}

	slog.Debug("Found aws credentials", "source", creds.Source)

	var envVar []string

	envVar = append(envVar, fmt.Sprintf("AWS_ACCESS_KEY_ID=%s", creds.AccessKeyID))
	envVar = append(envVar, fmt.Sprintf("AWS_SECRET_ACCESS_KEY=%s", creds.SecretAccessKey))
	if creds.SessionToken != "" {
		envVar = append(envVar, fmt.Sprintf("AWS_SESSION_TOKEN=%s", creds.SessionToken))
	}
	return envVar, nil
}

func (h *Handler) options(w http.ResponseWriter, r *http.Request) {
	headers := w.Header()

	headers["Access-Control-Allow-Headers"] = r.Header["Access-Control-Request-Headers"]
	headers["Access-Control-Allow-Methods"] = []string{allowedMethods}
	w.WriteHeader(http.StatusOK)
}
