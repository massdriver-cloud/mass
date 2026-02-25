package studio

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/cli/browser"
	"github.com/massdriver-cloud/mass/pkg/api"
	"github.com/massdriver-cloud/mass/pkg/proxy"
	"github.com/massdriver-cloud/mass/pkg/studio/artifactdef"
	sb "github.com/massdriver-cloud/mass/pkg/studio/bundle"

	mdclient "github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
	dockerclient "github.com/moby/moby/client"
	httpSwagger "github.com/swaggo/http-swagger"
)

// Studio represents the local development studio server
type Studio struct {
	BaseDir          string
	Items            []StudioItem
	DockerCli        *dockerclient.Client
	MassdriverClient *mdclient.Client
	SSE              *SSENotifier

	httpServer *http.Server
	itemsMu    sync.RWMutex
}

// New creates a new Studio instance
func New(dir string) (*Studio, error) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	cli, err := dockerclient.NewClientWithOpts(dockerclient.FromEnv, dockerclient.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("error creating docker client: %w", err)
	}

	mdClient, err := mdclient.New()
	if err != nil {
		return nil, fmt.Errorf("error creating massdriver client: %w", err)
	}

	// Initial scan of the directory
	items, err := ScanDirectory(absDir)
	if err != nil {
		return nil, fmt.Errorf("error scanning directory: %w", err)
	}

	slog.Info("Discovered items", "bundles", len(FilterByType(items, ItemTypeBundle)), "artifact-definitions", len(FilterByType(items, ItemTypeArtifactDefinition)))

	server := &http.Server{ReadHeaderTimeout: 60 * time.Second}

	return &Studio{
		BaseDir:          absDir,
		Items:            items,
		DockerCli:        cli,
		MassdriverClient: mdClient,
		SSE:              NewSSENotifier(),
		httpServer:       server,
	}, nil
}

// Start starts the studio server
func (s *Studio) Start(port string, launchBrowser bool) error {
	ln, err := net.Listen("tcp", "127.0.0.1:"+port)
	if err != nil {
		slog.Error(err.Error())
		return err
	}

	serverURL := "http://" + ln.Addr().String()

	slog.Info(fmt.Sprintf("Visit %s in your browser", serverURL))

	if launchBrowser {
		go s.openUIinBrowser(serverURL)
	}

	return s.httpServer.Serve(ln)
}

// Stop gracefully stops the studio server
func (s *Studio) Stop(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

// Rescan rescans the directory for items
func (s *Studio) Rescan() error {
	items, err := ScanDirectory(s.BaseDir)
	if err != nil {
		return err
	}

	s.itemsMu.Lock()
	s.Items = items
	s.itemsMu.Unlock()

	s.SSE.BroadcastRescanComplete(len(items))
	return nil
}

// RegisterHandlers registers all HTTP handlers for the studio
// If localUIDir is provided, it serves the UI from that directory instead of downloading
func (s *Studio) RegisterHandlers(ctx context.Context, localUIDir string) {
	var studioUIDir string
	var err error

	if localUIDir != "" {
		// Use local UI directory for development
		studioUIDir = localUIDir
		slog.Info("Serving UI from local directory", "path", studioUIDir)
	} else {
		// Download UI from GitHub releases
		studioUIDir, err = SetupUIDir()
		if err != nil {
			slog.Error(err.Error())
			os.Exit(1)
		}

		if err = DownloadUI(ctx, studioUIDir); err != nil {
			slog.Error(err.Error())
			os.Exit(1)
		}
	}

	// UI routes - handle both built UI structure (public/) and dev structure (direct)
	publicDir := filepath.Join(studioUIDir, "public")
	if _, statErr := os.Stat(publicDir); os.IsNotExist(statErr) {
		// No public subdirectory, serve from root (dev mode with npm start)
		publicDir = studioUIDir
	}

	http.Handle("/", s.corsMiddleware(http.FileServer(http.Dir(publicDir))))
	http.Handle("/dist/", s.corsMiddleware(http.FileServer(http.Dir(studioUIDir))))
	http.Handle("/public/", s.corsMiddleware(http.FileServer(http.Dir(studioUIDir))))

	// Client-side routing fallback
	http.HandleFunc("/deploy", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		indexPath := filepath.Join(publicDir, "index.html")
		http.ServeFile(w, r, indexPath)
	})

	// Health check
	http.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		_, _ = w.Write([]byte("ok"))
	})

	// Swagger docs
	http.HandleFunc("/swagger/", httpSwagger.Handler(
		httpSwagger.URL("http://127.0.0.1:8080/swagger/doc.json"),
	))

	// API: Items discovery
	http.Handle("/api/items", s.corsMiddleware(http.HandlerFunc(s.handleItems)))
	http.Handle("/api/items/rescan", s.corsMiddleware(http.HandlerFunc(s.handleRescan)))

	// API: Artifact definitions from Massdriver (for dropdowns)
	http.Handle("/api/artifact-definitions", s.corsMiddleware(http.HandlerFunc(s.handleArtifactDefinitions)))

	// API: Create new artifact definition
	http.Handle("/api/artifact-definition/create", s.corsMiddleware(http.HandlerFunc(s.handleCreateArtifactDefinition)))

	// API: Item-specific routes (bundles and artifact definitions)
	http.Handle("/api/bundle/", s.corsMiddleware(http.HandlerFunc(s.handleBundleRoutes)))
	http.Handle("/api/artifact-definition/", s.corsMiddleware(http.HandlerFunc(s.handleArtifactDefRoutes)))

	// Legacy bundle routes for backward compatibility with existing UI
	http.Handle("/bundle/", s.corsMiddleware(http.HandlerFunc(s.handleLegacyBundleRoutes)))
	http.Handle("/bundle-server/", http.StripPrefix("/bundle-server/", s.corsMiddleware(http.FileServer(http.Dir(s.BaseDir)))))

	// SSE events endpoint
	http.Handle("/events", s.SSE)

	// Proxy to Massdriver API
	proxyHandler, err := proxy.New(s.MassdriverClient.Config.URL)
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
	http.Handle("/proxy/", s.corsMiddleware(proxyHandler))

	// Config endpoint
	http.Handle("/config", s.corsMiddleware(http.HandlerFunc(s.handleConfig)))

	// Version endpoint
	http.Handle("/version", s.corsMiddleware(http.HandlerFunc(s.handleVersion)))
}

// handleItems returns the list of discovered items
func (s *Studio) handleItems(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	s.itemsMu.RLock()
	items := s.Items
	s.itemsMu.RUnlock()

	// Convert absolute paths to relative paths for the API
	bundles := FilterByType(items, ItemTypeBundle)
	artifactDefs := FilterByType(items, ItemTypeArtifactDefinition)

	// Make paths relative to BaseDir for the frontend
	for i := range bundles {
		if rel, err := filepath.Rel(s.BaseDir, bundles[i].Path); err == nil {
			bundles[i].Path = rel
		}
	}
	for i := range artifactDefs {
		if rel, err := filepath.Rel(s.BaseDir, artifactDefs[i].Path); err == nil {
			artifactDefs[i].Path = rel
		}
	}

	// Group items by type for the UI
	response := struct {
		Bundles             []StudioItem `json:"bundles"`
		ArtifactDefinitions []StudioItem `json:"artifactDefinitions"`
	}{
		Bundles:             bundles,
		ArtifactDefinitions: artifactDefs,
	}

	out, err := json.Marshal(response)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(out)
}

// handleRescan triggers a rescan of the directory
func (s *Studio) handleRescan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	if err := s.Rescan(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.itemsMu.RLock()
	count := len(s.Items)
	s.itemsMu.RUnlock()

	response := map[string]any{"status": "ok", "count": count}
	out, _ := json.Marshal(response)
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(out)
}

// handleArtifactDefinitions fetches artifact definitions from Massdriver API
func (s *Studio) handleArtifactDefinitions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()
	defs, err := api.ListArtifactDefinitions(ctx, s.MassdriverClient)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to fetch artifact definitions: %v", err), http.StatusInternalServerError)
		return
	}

	out, err := json.Marshal(defs)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(out)
}

// handleCreateArtifactDefinition handles creating a new artifact definition
func (s *Studio) handleCreateArtifactDefinition(w http.ResponseWriter, r *http.Request) {
	artifactdef.CreateDefinition(w, r, s.BaseDir)

	// Trigger rescan after creation
	if r.Method == http.MethodPost {
		go func() {
			if err := s.Rescan(); err != nil {
				slog.Error("Failed to rescan after artifact creation", "error", err)
			}
		}()
	}
}

// handleBundleRoutes handles routes for specific bundles
// Route format: /api/bundle/{path...}/{action}
func (s *Studio) handleBundleRoutes(w http.ResponseWriter, r *http.Request) {
	// Parse path: /api/bundle/path/to/bundle/action
	path := strings.TrimPrefix(r.URL.Path, "/api/bundle/")
	parts := strings.Split(path, "/")

	if len(parts) < 1 {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}

	// Last part is the action (build, params, connections, secrets, envs)
	action := parts[len(parts)-1]
	bundlePath := strings.Join(parts[:len(parts)-1], "/")

	// Resolve to absolute path and validate it's within BaseDir
	absPath := filepath.Join(s.BaseDir, bundlePath)
	if !s.isSubPath(absPath) {
		http.Error(w, "invalid path: path traversal not allowed", http.StatusBadRequest)
		return
	}

	// Create handler for this bundle
	handler, err := sb.NewHandler(absPath, s.MassdriverClient)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to create handler: %v", err), http.StatusInternalServerError)
		return
	}

	switch action {
	case "build":
		handler.Build(w, r)
	case "params":
		handler.Params(w, r)
	case "connections":
		handler.Connections(w, r)
	case "secrets":
		handler.GetSecrets(w, r)
	case "envs":
		handler.GetEnvironmentVariables(w, r)
	case "bundle":
		handler.GetBundle(w, r)
	default:
		http.Error(w, "unknown action", http.StatusNotFound)
	}
}

// handleArtifactDefRoutes handles routes for specific artifact definitions
// Route format: /api/artifact-definition/{path...}/{action}
func (s *Studio) handleArtifactDefRoutes(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/artifact-definition/")

	// Handle create separately (no path)
	if path == "create" {
		s.handleCreateArtifactDefinition(w, r)
		return
	}

	parts := strings.Split(path, "/")

	if len(parts) < 1 {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}

	// Last part is the action (definition, schema, save)
	action := parts[len(parts)-1]
	defPath := strings.Join(parts[:len(parts)-1], "/")

	// Resolve to absolute path and validate it's within BaseDir
	absPath := filepath.Join(s.BaseDir, defPath)
	if !s.isSubPath(absPath) {
		http.Error(w, "invalid path: path traversal not allowed", http.StatusBadRequest)
		return
	}

	// Create handler for this artifact definition
	handler, err := artifactdef.NewHandler(absPath, s.MassdriverClient)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to create handler: %v", err), http.StatusInternalServerError)
		return
	}

	switch action {
	case "definition":
		handler.GetDefinition(w, r)
	case "schema":
		handler.GetSchema(w, r)
	case "save":
		handler.SaveDefinition(w, r)
		// Notify file change
		go func() {
			s.SSE.BroadcastFileChanged(absPath, ItemTypeArtifactDefinition)
		}()
	default:
		http.Error(w, "unknown action", http.StatusNotFound)
	}
}

// handleLegacyBundleRoutes handles legacy /bundle/* routes for backward compatibility
func (s *Studio) handleLegacyBundleRoutes(w http.ResponseWriter, r *http.Request) {
	// For legacy routes, use the first bundle found or BaseDir
	bundleDir := s.BaseDir

	s.itemsMu.RLock()
	bundles := FilterByType(s.Items, ItemTypeBundle)
	s.itemsMu.RUnlock()

	if len(bundles) > 0 {
		bundleDir = bundles[0].Path
	}

	handler, err := sb.NewHandler(bundleDir, s.MassdriverClient)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to create handler: %v", err), http.StatusInternalServerError)
		return
	}

	action := strings.TrimPrefix(r.URL.Path, "/bundle/")
	switch action {
	case "build":
		handler.Build(w, r)
	case "params":
		handler.Params(w, r)
	case "connections":
		handler.Connections(w, r)
	case "secrets":
		handler.GetSecrets(w, r)
	case "envs":
		handler.GetEnvironmentVariables(w, r)
	default:
		http.Error(w, "unknown action", http.StatusNotFound)
	}
}

// handleConfig returns the Massdriver client configuration
func (s *Studio) handleConfig(w http.ResponseWriter, _ *http.Request) {
	response := struct {
		OrgID  string `json:"orgID"`
		APIKey string `json:"apiKey"`
		URL    string `json:"url"`
	}{
		OrgID:  s.MassdriverClient.Config.OrganizationID,
		APIKey: s.MassdriverClient.Config.Credentials.Secret,
		URL:    s.MassdriverClient.Config.URL,
	}

	out, err := json.Marshal(response)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(out)
}

// handleVersion returns version information
func (s *Studio) handleVersion(w http.ResponseWriter, _ *http.Request) {
	// TODO: Implement proper version checking
	response := struct {
		IsLatest       bool   `json:"isLatest"`
		LatestVersion  string `json:"latestVersion"`
		CurrentVersion string `json:"currentVersion"`
	}{
		IsLatest:       true,
		LatestVersion:  "dev",
		CurrentVersion: "dev",
	}

	out, _ := json.Marshal(response)
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(out)
}

// corsMiddleware adds CORS headers to responses
func (s *Studio) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")

		if r.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", r.Header.Get("Access-Control-Request-Headers"))
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// isSubPath validates that targetPath is within BaseDir (prevents path traversal)
func (s *Studio) isSubPath(targetPath string) bool {
	absTarget, err := filepath.Abs(targetPath)
	if err != nil {
		return false
	}
	absBase, err := filepath.Abs(s.BaseDir)
	if err != nil {
		return false
	}
	// Ensure both paths end with separator for proper prefix matching
	if !strings.HasSuffix(absBase, string(filepath.Separator)) {
		absBase += string(filepath.Separator)
	}
	return strings.HasPrefix(absTarget+string(filepath.Separator), absBase)
}

// openUIinBrowser opens the studio UI in the default browser
func (s *Studio) openUIinBrowser(serverURL string) {
	var iter int
	for {
		if iter > 3 {
			slog.Warn("Studio never responded healthy")
			return
		}

		healthURL, err := url.JoinPath(serverURL, "health")
		if err != nil {
			slog.Error("Error creating healthcheck", "error", err.Error())
			return
		}

		ctx, cancelFunc := context.WithTimeout(context.Background(), 1*time.Second)

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, healthURL, nil)
		if err != nil {
			cancelFunc()
			slog.Error("Error creating healthcheck", "error", err.Error())
			return
		}

		res, err := http.DefaultClient.Do(req)
		cancelFunc() // Cancel context immediately after request completes

		if err == nil {
			if res.StatusCode == http.StatusOK {
				res.Body.Close()
				if err = browser.OpenURL(serverURL); err != nil {
					slog.Error("Error trying to open browser", "error", err.Error())
				}
				return
			}
			res.Body.Close()
		}

		iter++
		time.Sleep(200 * time.Millisecond)
	}
}
