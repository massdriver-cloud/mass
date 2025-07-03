package jsonschema

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	gourl "net/url"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"gopkg.in/yaml.v3"
)

const filePrefix = "file://"

var loaderPrefixPattern = regexp.MustCompile(`^(file|http|https)://`)

// LoadSchemaFromFile loads a JSON schema from a local file path.
// The path can be absolute, relative, or prefixed with "file://".
func LoadSchemaFromFile(path string) (*jsonschema.Schema, error) {
	var ref string
	if loaderPrefixPattern.MatchString(path) {
		ref = path
	} else {
		// Ensure relative paths work correctly with the file loader
		if filepath.IsLocal(path) && !strings.HasPrefix(path, "./") {
			path = "./" + path
		}
		ref = filePrefix + path
	}

	compiler := newCompiler()
	schema, err := compiler.Compile(ref)
	if err != nil {
		return nil, fmt.Errorf("failed to compile schema from file %q: %w", path, err)
	}
	return schema, nil
}

// LoadSchemaFromURL loads a JSON schema from an HTTP or HTTPS URL.
func LoadSchemaFromURL(url string) (*jsonschema.Schema, error) {
	compiler := newCompiler()
	schema, err := compiler.Compile(url)
	if err != nil {
		return nil, fmt.Errorf("failed to compile schema from URL %q: %w", url, err)
	}
	return schema, nil
}

// LoadSchemaFromGo loads a JSON schema from a Go object by marshaling it to JSON first.
func LoadSchemaFromGo(obj any) (*jsonschema.Schema, error) {
	jsonBytes, err := json.Marshal(obj)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal Go object to JSON: %w", err)
	}
	return LoadSchemaFromReader(bytes.NewReader(jsonBytes))
}

// LoadSchemaFromReader loads a JSON schema from an io.Reader.
func LoadSchemaFromReader(reader io.Reader) (*jsonschema.Schema, error) {
	compiler := newCompiler()
	schema, err := jsonschema.UnmarshalJSON(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON schema: %w", err)
	}
	if err := compiler.AddResource("schema.json", schema); err != nil {
		return nil, fmt.Errorf("failed to add resource to compiler: %w", err)
	}

	compiledSchema, err := compiler.Compile("schema.json")
	if err != nil {
		return nil, fmt.Errorf("failed to compile schema: %w", err)
	}
	return compiledSchema, nil
}

// newCompiler creates a new JSON schema compiler with default settings.
func newCompiler() *jsonschema.Compiler {
	compiler := jsonschema.NewCompiler()
	compiler.DefaultDraft(jsonschema.Draft7)

	loader := newLoader(map[string]string{})
	compiler.UseLoader(loader)

	return compiler
}

// newLoader creates a URL loader with HTTP client timeout and file system support.
func newLoader(mappings map[string]string) jsonschema.URLLoader {
	httpLoader := HTTPLoader(http.Client{
		Timeout: 15 * time.Second,
	})
	return &Loader{
		mappings: mappings,
		fallback: jsonschema.SchemeURLLoader{
			"file":  FileLoader{},
			"http":  &httpLoader,
			"https": &httpLoader,
		},
	}
}

// Loader is a custom URL loader that supports file path mappings and falls back to standard loaders.
type Loader struct {
	mappings map[string]string
	fallback jsonschema.URLLoader
}

func (l *Loader) Load(url string) (any, error) {
	for prefix, dir := range l.mappings {
		if suffix, ok := strings.CutPrefix(url, prefix); ok {
			return loadFile(filepath.Join(dir, suffix))
		}
	}
	return l.fallback.Load(url)
}

func loadFile(path string) (any, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	if ext := filepath.Ext(path); ext == ".yaml" || ext == ".yml" {
		var v any
		err := yaml.NewDecoder(f).Decode(&v)
		return v, err
	}
	return jsonschema.UnmarshalJSON(f)
}

// FileLoader handles loading schema files from the local filesystem.
type FileLoader struct{}

func (l FileLoader) Load(url string) (any, error) {
	path, err := l.ToFile(url)
	if err != nil {
		return nil, err
	}
	return loadFile(path)
}

func (l FileLoader) ToFile(url string) (string, error) {
	u, err := gourl.Parse(url)
	if err != nil {
		return "", err
	}
	if u.Scheme != "file" {
		return "", fmt.Errorf("invalid file url: %s", u)
	}
	path := u.Path
	if runtime.GOOS == "windows" {
		path = strings.TrimPrefix(path, "/")
		path = filepath.FromSlash(path)
	}
	if u.Host != "" {
		return gourl.JoinPath(u.Host, path)
	}
	return path, nil
}

// HTTPLoader handles loading schemas from HTTP/HTTPS URLs with YAML support.
type HTTPLoader http.Client

func (l *HTTPLoader) Load(url string) (any, error) {
	client := (*http.Client)(l)
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		_ = resp.Body.Close()
		return nil, fmt.Errorf("%s returned status code %d", url, resp.StatusCode)
	}
	defer resp.Body.Close()

	isYAML := strings.HasSuffix(url, ".yaml") || strings.HasSuffix(url, ".yml")
	if !isYAML {
		ctype := resp.Header.Get("Content-Type")
		isYAML = strings.HasSuffix(ctype, "/yaml") || strings.HasSuffix(ctype, "-yaml")
	}
	if isYAML {
		var v any
		err := yaml.NewDecoder(resp.Body).Decode(&v)
		return v, err
	}
	return jsonschema.UnmarshalJSON(resp.Body)
}
