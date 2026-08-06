package resourcetype

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// placeholderVersion is written into the converted massdriver.yaml since a raw
// JSON schema carries no version. The author must set a real version before
// publishing.
const placeholderVersion = "0.0.0"

// ConvertResult describes the files a Convert call produced.
type ConvertResult struct {
	MassdriverYAML string   // path to the written massdriver.yaml
	ExtraFiles     []string // paths to extracted instruction/export files
}

// Convert reads a raw JSON (or YAML) resource type schema at schemaPath and
// writes an equivalent massdriver.yaml. Inlined instruction/export content is
// extracted back out to referenced files. outputPath is the massdriver.yaml to
// write; when empty it defaults to a massdriver.yaml alongside schemaPath.
// Existing files are not overwritten unless force is set.
func Convert(schemaPath, outputPath string, force bool) (*ConvertResult, error) {
	raw, readErr := readRawSchema(schemaPath)
	if readErr != nil {
		return nil, readErr
	}

	if outputPath == "" {
		outputPath = filepath.Join(filepath.Dir(schemaPath), "massdriver.yaml")
	}
	outputDir := filepath.Dir(outputPath)

	config, extraFiles := reverseBuild(raw)

	out, marshalErr := yaml.Marshal(config)
	if marshalErr != nil {
		return nil, fmt.Errorf("failed to marshal massdriver.yaml: %w", marshalErr)
	}

	// Refuse to clobber anything unless forced.
	targets := []string{outputPath}
	for rel := range extraFiles {
		targets = append(targets, filepath.Join(outputDir, rel))
	}
	if !force {
		for _, t := range targets {
			if _, statErr := os.Stat(t); statErr == nil {
				return nil, fmt.Errorf("%s already exists; use --force to overwrite", t)
			}
		}
	}

	if mkErr := os.MkdirAll(outputDir, 0750); mkErr != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", mkErr)
	}

	result := &ConvertResult{MassdriverYAML: outputPath}
	for rel, content := range extraFiles {
		dst := filepath.Join(outputDir, rel)
		if mkErr := os.MkdirAll(filepath.Dir(dst), 0750); mkErr != nil {
			return nil, fmt.Errorf("failed to create directory for %s: %w", rel, mkErr)
		}
		if writeErr := os.WriteFile(dst, content, 0600); writeErr != nil {
			return nil, fmt.Errorf("failed to write %s: %w", rel, writeErr)
		}
		result.ExtraFiles = append(result.ExtraFiles, dst)
	}

	if writeErr := os.WriteFile(outputPath, out, 0600); writeErr != nil {
		return nil, fmt.Errorf("failed to write %s: %w", outputPath, writeErr)
	}

	return result, nil
}

func readRawSchema(path string) (map[string]any, error) {
	data, readErr := os.ReadFile(path)
	if readErr != nil {
		return nil, fmt.Errorf("failed to read schema: %w", readErr)
	}

	var raw map[string]any
	switch strings.ToLower(filepath.Ext(path)) {
	case ".json":
		if err := json.Unmarshal(data, &raw); err != nil {
			return nil, fmt.Errorf("failed to parse JSON schema: %w", err)
		}
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, &raw); err != nil {
			return nil, fmt.Errorf("failed to parse YAML schema: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported schema file extension: %s (expected .json, .yaml, or .yml)", filepath.Ext(path))
	}
	return raw, nil
}

// reverseBuild is the inverse of [Build]: it lifts the `$md` block back into the
// massdriver.yaml fields, extracts inlined instruction/export content into files
// keyed by their relative path, and moves the remaining keys under `schema`.
func reverseBuild(raw map[string]any) (*MassdriverYAML, map[string][]byte) {
	config := &MassdriverYAML{Version: placeholderVersion}
	extraFiles := map[string][]byte{}

	if md, ok := raw["$md"].(map[string]any); ok {
		config.Name = asString(md["name"])
		config.Label = asString(md["label"])
		config.Icon = asString(md["icon"])

		if uiRaw, ok := md["ui"].(map[string]any); ok {
			config.UI = reverseUI(uiRaw, extraFiles)
		}

		if exportsRaw, ok := md["export"].([]any); ok {
			config.Exports = reverseExports(exportsRaw, extraFiles)
		}
	}

	// Everything that isn't the $md block is the JSON schema itself.
	schema := map[string]any{}
	for key, value := range raw {
		if key == "$md" {
			continue
		}
		schema[key] = value
	}
	config.Schema = schema

	return config, extraFiles
}

func reverseUI(uiRaw map[string]any, extraFiles map[string][]byte) *UIConfig {
	ui := &UIConfig{
		ConnectionOrientation:   asString(uiRaw["connectionOrientation"]),
		EnvironmentDefaultGroup: asString(uiRaw["environmentDefaultGroup"]),
	}

	instructions, ok := uiRaw["instructions"].([]any)
	if !ok {
		return ui
	}
	for i, instRaw := range instructions {
		inst, ok := instRaw.(map[string]any)
		if !ok {
			continue
		}
		label := asString(inst["label"])
		rel := uniqueRel(extraFiles, "instructions", slugify(label, i), "md", i)
		extraFiles[rel] = []byte(asString(inst["content"]))
		ui.Instructions = append(ui.Instructions, InstructionConfig{
			Label: label,
			Path:  "./" + rel,
		})
	}
	return ui
}

func reverseExports(exportsRaw []any, extraFiles map[string][]byte) []ExportConfig {
	var exports []ExportConfig
	for i, expRaw := range exportsRaw {
		exp, ok := expRaw.(map[string]any)
		if !ok {
			continue
		}
		lang := asString(exp["templateLang"])
		ext := lang
		if ext == "" {
			ext = "tmpl"
		}
		rel := uniqueRel(extraFiles, "exports", slugify(asString(exp["downloadButtonText"]), i), ext, i)
		extraFiles[rel] = []byte(asString(exp["template"]))
		exports = append(exports, ExportConfig{
			DownloadButtonText: asString(exp["downloadButtonText"]),
			FileFormat:         asString(exp["fileFormat"]),
			TemplatePath:       "./" + rel,
			TemplateLang:       lang,
		})
	}
	return exports
}

// uniqueRel builds "<dir>/<slug>.<ext>", appending the item index if that path
// was already taken so two items with the same label don't clobber each other.
func uniqueRel(extraFiles map[string][]byte, dir, slug, ext string, index int) string {
	rel := fmt.Sprintf("%s/%s.%s", dir, slug, ext)
	if _, taken := extraFiles[rel]; !taken {
		return rel
	}
	return fmt.Sprintf("%s/%s-%d.%s", dir, slug, index+1, ext)
}

func asString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

var nonSlugChars = regexp.MustCompile(`[^a-z0-9]+`)

// slugify turns a human label into a filesystem-friendly slug, falling back to
// an index-based name when the label has no usable characters.
func slugify(label string, index int) string {
	slug := nonSlugChars.ReplaceAllString(strings.ToLower(label), "-")
	slug = strings.Trim(slug, "-")
	if slug == "" {
		return strconv.Itoa(index + 1)
	}
	return slug
}
