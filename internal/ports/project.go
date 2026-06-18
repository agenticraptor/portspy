package ports

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// maxMarkerFileSize caps how much of a project marker file we read. Marker
// files (go.mod, package.json, Cargo.toml, …) are tiny, so capping the read is
// cheap defense against a pathological or hostile file somewhere in a scanned
// directory tree.
const maxMarkerFileSize = 1 << 20 // 1 MiB

// fileSystem abstracts the handful of filesystem reads project detection needs,
// so the walker can be unit-tested without touching disk.
type fileSystem interface {
	exists(path string) bool
	readFile(path string) ([]byte, error)
}

type osFS struct{}

func (osFS) exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func (osFS) readFile(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	return io.ReadAll(io.LimitReader(f, maxMarkerFileSize))
}

// marker maps a sentinel filename to the ecosystem it signals.
type marker struct {
	file string
	typ  string
}

// projectMarkers are checked in priority order at each directory level. The
// nearest directory (walking up from the process cwd) that contains any marker
// wins, and within a level the first match in this list wins.
var projectMarkers = []marker{
	{"go.mod", "go"},
	{"package.json", "node"},
	{"deno.json", "deno"},
	{"deno.jsonc", "deno"},
	{"Cargo.toml", "rust"},
	{"pyproject.toml", "python"},
	{"Pipfile", "python"},
	{"requirements.txt", "python"},
	{"setup.py", "python"},
	{"Gemfile", "ruby"},
	{"composer.json", "php"},
	{"mix.exs", "elixir"},
	{"pubspec.yaml", "dart"},
	{"pom.xml", "java"},
	{"build.gradle", "java"},
	{"build.gradle.kts", "java"},
	{"CMakeLists.txt", "c/c++"},
}

// DetectProject walks up from start looking for a project root. A language
// marker (go.mod, package.json, …) is preferred; a lone .git directory is used
// as a fallback root. It returns the zero Project if nothing is found.
func DetectProject(fs fileSystem, start string) Project {
	if start == "" {
		return Project{}
	}
	dir := filepath.Clean(start)
	for {
		for _, m := range projectMarkers {
			path := filepath.Join(dir, m.file)
			if fs.exists(path) {
				return Project{
					Name: projectName(fs, dir, m),
					Type: m.typ,
					Root: dir,
				}
			}
		}
		if fs.exists(filepath.Join(dir, ".git")) {
			return Project{Name: filepath.Base(dir), Root: dir}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return Project{}
		}
		dir = parent
	}
}

// projectName extracts a friendly name from a marker file, falling back to the
// directory's base name when the file can't be parsed.
func projectName(fs fileSystem, dir string, m marker) string {
	base := filepath.Base(dir)
	data, err := fs.readFile(filepath.Join(dir, m.file))
	if err != nil {
		return base
	}
	var name string
	switch m.file {
	case "go.mod":
		name = lastPathSegment(moduleName(data))
	case "package.json", "composer.json", "deno.json", "deno.jsonc":
		name = jsonName(data)
	case "Cargo.toml":
		name = tomlName(data, "package")
	case "pyproject.toml":
		name = firstNonEmpty(tomlName(data, "project"), tomlName(data, "tool.poetry"))
	}
	if name = strings.TrimSpace(name); name != "" {
		return name
	}
	return base
}

func moduleName(data []byte) string {
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if rest, ok := strings.CutPrefix(line, "module "); ok {
			return strings.TrimSpace(rest)
		}
	}
	return ""
}

func lastPathSegment(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if i := strings.LastIndex(s, "/"); i >= 0 {
		return s[i+1:]
	}
	return s
}

func jsonName(data []byte) string {
	var v struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(data, &v); err != nil {
		return ""
	}
	return v.Name
}

// tomlName scans a tiny subset of TOML for `name = "..."` within [section].
// It is deliberately minimal — just enough to read a project name without a
// TOML dependency.
func tomlName(data []byte, section string) string {
	inSection := false
	for _, raw := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			header := strings.Trim(line, "[]")
			inSection = strings.EqualFold(strings.TrimSpace(header), section)
			continue
		}
		if !inSection {
			continue
		}
		if key, val, ok := strings.Cut(line, "="); ok && strings.EqualFold(strings.TrimSpace(key), "name") {
			return strings.Trim(strings.TrimSpace(val), `"'`)
		}
	}
	return ""
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

// fsProjectFinder is the production projectFinder backed by the real filesystem.
type fsProjectFinder struct{}

func (fsProjectFinder) Find(dir string) Project { return DetectProject(osFS{}, dir) }
