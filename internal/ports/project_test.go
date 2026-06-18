package ports

import (
	"os"
	"path/filepath"
	"testing"
)

// fakeFS is an in-memory fileSystem for testing project detection.
type fakeFS struct {
	files map[string]string // path -> contents (also marks existence)
	dirs  map[string]bool   // directory paths that exist (e.g. ".git")
}

func newFakeFS() fakeFS {
	return fakeFS{files: map[string]string{}, dirs: map[string]bool{}}
}

func (f fakeFS) exists(p string) bool {
	if _, ok := f.files[p]; ok {
		return true
	}
	return f.dirs[p]
}

func (f fakeFS) readFile(p string) ([]byte, error) {
	if c, ok := f.files[p]; ok {
		return []byte(c), nil
	}
	return nil, os.ErrNotExist
}

func TestDetectProjectGoModule(t *testing.T) {
	fs := newFakeFS()
	fs.files[filepath.Join("/a/b", "go.mod")] = "module github.com/x/coolproj\n\ngo 1.22\n"

	got := DetectProject(fs, "/a/b/cmd/server")
	if got.Type != "go" || got.Name != "coolproj" || got.Root != "/a/b" {
		t.Errorf("go.mod detection = %+v", got)
	}
}

func TestDetectProjectPackageJSON(t *testing.T) {
	fs := newFakeFS()
	fs.files[filepath.Join("/srv/web", "package.json")] = `{"name":"@scope/web","version":"1.0.0"}`

	got := DetectProject(fs, "/srv/web")
	if got.Type != "node" || got.Name != "@scope/web" {
		t.Errorf("package.json detection = %+v", got)
	}
}

func TestDetectProjectCargo(t *testing.T) {
	fs := newFakeFS()
	fs.files[filepath.Join("/code/rs", "Cargo.toml")] = "[package]\nname = \"rusty\"\nversion = \"0.1.0\"\n"

	got := DetectProject(fs, "/code/rs/src")
	if got.Type != "rust" || got.Name != "rusty" {
		t.Errorf("Cargo.toml detection = %+v", got)
	}
}

func TestDetectProjectPyprojectPoetry(t *testing.T) {
	fs := newFakeFS()
	fs.files[filepath.Join("/py/app", "pyproject.toml")] = "[tool.poetry]\nname = \"pyproj\"\n"

	got := DetectProject(fs, "/py/app")
	if got.Type != "python" || got.Name != "pyproj" {
		t.Errorf("pyproject detection = %+v", got)
	}
}

func TestDetectProjectGitFallback(t *testing.T) {
	fs := newFakeFS()
	fs.dirs[filepath.Join("/repo", ".git")] = true

	got := DetectProject(fs, "/repo/internal/pkg")
	if got.Name != "repo" || got.Root != "/repo" || got.Type != "" {
		t.Errorf("git fallback = %+v", got)
	}
}

func TestDetectProjectMonorepoPrefersNearest(t *testing.T) {
	fs := newFakeFS()
	fs.dirs[filepath.Join("/repo", ".git")] = true
	fs.files[filepath.Join("/repo", "package.json")] = `{"name":"root"}`
	fs.files[filepath.Join("/repo/packages/api", "package.json")] = `{"name":"api"}`

	got := DetectProject(fs, "/repo/packages/api")
	if got.Name != "api" {
		t.Errorf("monorepo should prefer nearest package, got %+v", got)
	}
}

func TestDetectProjectNone(t *testing.T) {
	fs := newFakeFS()
	if got := DetectProject(fs, "/nowhere/special"); !got.Empty() {
		t.Errorf("expected empty project, got %+v", got)
	}
	if got := DetectProject(fs, ""); !got.Empty() {
		t.Errorf("empty start should yield empty project, got %+v", got)
	}
}

func TestProjectNameFallsBackToDir(t *testing.T) {
	fs := newFakeFS()
	// A marker file with no parseable name should fall back to the dir base.
	fs.files[filepath.Join("/svc/payments", "go.mod")] = "// no module line here\n"

	got := DetectProject(fs, "/svc/payments")
	if got.Name != "payments" {
		t.Errorf("expected dir-base fallback name, got %+v", got)
	}
}
