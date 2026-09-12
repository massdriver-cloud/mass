// Package configfile edits the Massdriver config file.
//
// The SDK's massdriver/config package owns the schema, the file location, and
// how the file resolves into credentials: this package reads through
// [sdkconfig.ReadFile] and writes the types the SDK declares, so there is one
// definition of a profile and one notion of which profile is active.
//
// Writing is the CLI's job — the SDK only reads. Edits are applied to the
// parsed yaml.Node tree rather than by re-marshaling a struct, so comments,
// key order, and keys this CLI doesn't model survive a write.
package configfile

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"

	sdkconfig "github.com/massdriver-cloud/massdriver-sdk-go/massdriver/config"
	"gopkg.in/yaml.v3"
)

// EnvProfile selects the active profile. It mirrors the envconfig tag the SDK
// resolves ahead of current_profile; the SDK exports no constant for it. Read
// only to label output — the SDK is what acts on it.
const EnvProfile = "MASSDRIVER_PROFILE"

const (
	profilesKey       = "profiles"
	versionKey        = "version"
	currentProfileKey = "current_profile"

	// fileMode keeps the file owner-only: it holds API keys.
	fileMode os.FileMode = 0600
)

// NamedProfile is a profile together with the key it is stored under.
type NamedProfile struct {
	Name string `json:"name"`
	sdkconfig.Profile
}

// File is an editable view of the config file.
type File struct {
	path string
	// parsed is the SDK's view, used for every read. Nil when the file does
	// not exist.
	parsed *sdkconfig.File
	// doc is the same file as a node tree, used for every write.
	doc *yaml.Node
}

// Path returns the config file location the SDK will read.
func Path() (string, error) {
	return sdkconfig.FilePath()
}

// Load reads the config file. A missing file is not an error — it yields an
// empty File that Save will create. A file the SDK cannot parse is, so the
// CLI never edits a file the SDK would reject.
func Load() (*File, error) {
	path, err := Path()
	if err != nil {
		return nil, err
	}

	parsed, readErr := sdkconfig.ReadFile()
	if readErr != nil {
		return nil, readErr
	}

	f := &File{path: path, parsed: parsed}
	if parsed == nil {
		return f, nil
	}

	contents, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("could not read config file %s: %w", path, err)
	}
	var doc yaml.Node
	if err = yaml.Unmarshal(contents, &doc); err != nil {
		return nil, fmt.Errorf("could not parse config file %s: %w", path, err)
	}
	if doc.Kind != 0 {
		f.doc = &doc
	}

	return f, nil
}

// Path returns the file's location on disk.
func (f *File) Path() string { return f.path }

// Exists reports whether the file was present when loaded.
func (f *File) Exists() bool { return f.parsed != nil }

// Profiles returns every configured profile, sorted by name.
func (f *File) Profiles() []NamedProfile {
	names := f.parsed.ProfileNames()
	profiles := make([]NamedProfile, 0, len(names))
	for _, name := range names {
		profiles = append(profiles, NamedProfile{Name: name, Profile: f.parsed.Profiles[name]})
	}
	return profiles
}

// ProfileNames returns the configured profile names, sorted.
func (f *File) ProfileNames() []string { return f.parsed.ProfileNames() }

// Profile returns the named profile and whether it exists.
func (f *File) Profile(name string) (sdkconfig.Profile, bool) {
	if f.parsed == nil {
		return sdkconfig.Profile{}, false
	}
	p, exists := f.parsed.Profiles[name]
	return p, exists
}

// CurrentProfile returns the file's current_profile, or "" when unset.
func (f *File) CurrentProfile() string {
	if f.parsed == nil {
		return ""
	}
	return f.parsed.CurrentProfile
}

// ActiveProfileName reports which profile the SDK will select, so command
// output can label it. override is the --profile flag, which the CLI passes to
// the SDK as [massdriver.WithProfile].
//
// This mirrors the precedence documented on [sdkconfig.Load], which is
// authoritative. TestActiveProfileNameMatchesSDK holds the two in agreement.
func (f *File) ActiveProfileName(override string) string {
	if override != "" {
		return override
	}
	if env := os.Getenv(EnvProfile); env != "" {
		return env
	}
	if current := f.CurrentProfile(); current != "" {
		return current
	}
	return sdkconfig.DefaultProfileName
}

// SetProfile creates or updates a profile. Empty fields are removed from the
// file rather than written as blanks, so clearing a value and never setting
// one look the same on disk.
func (f *File) SetProfile(name string, p sdkconfig.Profile) error {
	if name == "" {
		return errors.New("profile name is required")
	}
	root, err := f.rootMap()
	if err != nil {
		return err
	}
	mapSet(root, versionKey, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!int", Value: strconv.Itoa(sdkconfig.Version)})

	profiles := mapGet(root, profilesKey)
	if profiles == nil || profiles.Kind != yaml.MappingNode {
		profiles = &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		mapSet(root, profilesKey, profiles)
	}

	node := mapGet(profiles, name)
	if node == nil || node.Kind != yaml.MappingNode {
		node = &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		mapSet(profiles, name, node)
	}

	setOrDelete(node, "organization_id", p.OrganizationID)
	setOrDelete(node, "api_key", p.APIKey)
	setOrDelete(node, "url", p.URL)
	setOrDelete(node, "templates_path", p.TemplatesPath)

	f.syncProfile(name, p)
	return nil
}

// RemoveProfile deletes a profile, reporting whether it was present. The
// current_profile pointer is cleared when it named the removed profile, since
// the SDK now fails rather than falling back when it points at nothing.
func (f *File) RemoveProfile(name string) bool {
	root, err := f.rootMap()
	if err != nil {
		return false
	}
	if !mapDelete(mapGet(root, profilesKey), name) {
		return false
	}
	if f.CurrentProfile() == name {
		mapDelete(root, currentProfileKey)
		f.parsed.CurrentProfile = ""
	}
	delete(f.parsed.Profiles, name)
	return true
}

// SetCurrentProfile points current_profile at name. The SDK reads this key, so
// the selection applies to every tool built on it, not just this CLI.
func (f *File) SetCurrentProfile(name string) error {
	root, err := f.rootMap()
	if err != nil {
		return err
	}
	mapSet(root, currentProfileKey, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: name})
	f.parsed.CurrentProfile = name
	return nil
}

// Save writes the file, creating parent directories as needed. The write is
// atomic and the result is owner-readable only.
func (f *File) Save() error {
	if f.doc == nil {
		return errors.New("nothing to write")
	}
	if err := os.MkdirAll(filepath.Dir(f.path), 0750); err != nil {
		return fmt.Errorf("could not create config directory: %w", err)
	}

	var buf bytes.Buffer
	encoder := yaml.NewEncoder(&buf)
	encoder.SetIndent(2)
	if err := encoder.Encode(f.doc); err != nil {
		return fmt.Errorf("could not encode config file: %w", err)
	}
	if err := encoder.Close(); err != nil {
		return fmt.Errorf("could not encode config file: %w", err)
	}

	tmp, err := os.CreateTemp(filepath.Dir(f.path), ".config.yaml.*")
	if err != nil {
		return fmt.Errorf("could not create temporary config file: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	if err = tmp.Chmod(fileMode); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("could not set permissions on config file: %w", err)
	}
	if _, err = tmp.Write(buf.Bytes()); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("could not write config file: %w", err)
	}
	if err = tmp.Close(); err != nil {
		return fmt.Errorf("could not write config file: %w", err)
	}
	if err = os.Rename(tmpName, f.path); err != nil {
		return fmt.Errorf("could not save config file %s: %w", f.path, err)
	}

	return nil
}

// syncProfile keeps the read view consistent with an edit, so a command can
// write a profile and then read it back without reloading.
func (f *File) syncProfile(name string, p sdkconfig.Profile) {
	if f.parsed == nil {
		f.parsed = &sdkconfig.File{Version: sdkconfig.Version}
	}
	if f.parsed.Profiles == nil {
		f.parsed.Profiles = map[string]sdkconfig.Profile{}
	}
	f.parsed.Profiles[name] = p
}

// rootMap returns the top-level mapping, creating the document skeleton for a
// file that does not exist yet.
func (f *File) rootMap() (*yaml.Node, error) {
	if f.doc == nil {
		f.doc = &yaml.Node{
			Kind:    yaml.DocumentNode,
			Content: []*yaml.Node{{Kind: yaml.MappingNode, Tag: "!!map"}},
		}
	}
	if len(f.doc.Content) == 0 {
		f.doc.Content = append(f.doc.Content, &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"})
	}
	root := f.doc.Content[0]
	if root.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("config file %s must contain a YAML mapping at the top level", f.path)
	}
	return root, nil
}

func mapGet(m *yaml.Node, key string) *yaml.Node {
	if m == nil || m.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value == key {
			return m.Content[i+1]
		}
	}
	return nil
}

// mapSet replaces a key's value in place, carrying over any comments attached
// to the value it displaces, or appends the pair when the key is new.
func mapSet(m *yaml.Node, key string, val *yaml.Node) {
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value == key {
			old := m.Content[i+1]
			val.HeadComment = old.HeadComment
			val.LineComment = old.LineComment
			val.FootComment = old.FootComment
			m.Content[i+1] = val
			return
		}
	}
	m.Content = append(m.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key},
		val,
	)
}

func mapDelete(m *yaml.Node, key string) bool {
	if m == nil || m.Kind != yaml.MappingNode {
		return false
	}
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value == key {
			m.Content = slices.Delete(m.Content, i, i+2)
			return true
		}
	}
	return false
}

func setOrDelete(m *yaml.Node, key, value string) {
	if value == "" {
		mapDelete(m, key)
		return
	}
	mapSet(m, key, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: value})
}
