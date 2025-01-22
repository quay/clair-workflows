package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"path"
	"runtime"
	"slices"
	"strings"
	"sync"

	"github.com/quay/clair-workflows/common/internal/dagger"
)

func cacheDir(ctx context.Context, name string, env string) dagger.WithContainerFunc {
	ns, _ := dag.CurrentModule().Name(ctx)
	opts := dagger.CacheVolumeOpts{
		Namespace: ns,
	}
	return func(c *dagger.Container) *dagger.Container {
		dir, err := c.EnvVariable(ctx, "XDG_CACHE_HOME")
		if dir == "" || err != nil {
			dir = "/root/.cache"
		}
		cache := dag.CacheVolume(name, opts)
		path := path.Join(dir, name)

		c = c.WithMountedCache(path, cache)
		if env != "" {
			c = c.WithEnvVariable(env, path)
		}
		return c
	}
}

func (m *Common) GoDistribution(
	//+optional
	version string,
	//+optional
	arch string,
) *dagger.Directory {
	const wd = `/run/untar`
	var ( // Using `path` on purpose.
		arFile = path.Join(wd, "archive")
		outDir = path.Join(wd, "go")
	)

	if version == "" {
		version = goVersionOnce()
	}
	if !strings.HasPrefix(version, "go") {
		version = "go" + version
	}
	if arch == "" {
		arch = runtime.GOARCH
	}

	dist := fmt.Sprintf(`https://go.dev/dl/%s.linux-%s.tar.gz`, version, arch)
	tarball := dag.HTTP(dist)
	cmd := []string{`tar`, `-xzf`, arFile}

	return m.UBI("").
		WithWorkdir(wd).
		WithFile(arFile, tarball).
		WithExec(cmd).
		Directory(outDir)
}

var goVersionOnce = sync.OnceValue(func() string {
	res, err := http.Get("https://go.dev/dl/?mode=json")
	if err != nil {
		panic(err)
	}

	type File struct {
		Filename string `json:"filename"`
		OS       string `json:"os"`
		Arch     string `json:"arch"`
		Version  string `json:"version"`
		SHA256   string `json:"sha256"`
		Size     int64  `json:"size"`
		Kind     string `json:"kind"`
	}
	type Version struct {
		Version string `json:"version"`
		Stable  bool   `json:"stable"`
		Files   []File `json:"files"`
	}

	var dl []Version
	if err := json.NewDecoder(res.Body).Decode(&dl); err != nil {
		panic(err)
	}

	v := slices.MaxFunc(dl, func(a, b Version) int {
		return strings.Compare(a.Version, b.Version)
	})

	return v.Version
})

func (*Common) UBI(
	//+optional
	tag string,
) *dagger.Container {
	if tag == "" {
		tag = Versions["ubi"]
	}
	ref := fmt.Sprintf(`registry.access.redhat.com/ubi9:%s`, tag)
	return dag.Container().
		From(ref)
}
