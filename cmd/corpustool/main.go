package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"hash/maphash"
	"io"
	"iter"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"runtime"
	"runtime/pprof"
	"slices"
	"strconv"
	"strings"

	"github.com/quay/clair-workflows/cmd/corpustool/sql"
	"golang.org/x/sync/errgroup"
	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"
)

func main() {
	var code int
	defer func() {
		if code != 0 {
			os.Exit(code)
		}
	}()

	var loglevel slog.LevelVar
	slog.SetDefault(
		slog.New(
			slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
				Level: &loglevel,
			})))

	ctx := context.Background()
	ctx, done := signal.NotifyContext(ctx, os.Interrupt)
	defer done()

	flag.BoolFunc("D", "debug logging", func(v string) error {
		ok, err := strconv.ParseBool(v)
		if err != nil {
			return err
		}
		if ok {
			loglevel.Set(slog.LevelDebug)
		}
		return nil
	})
	cpuprofile := flag.String("cpuprofile", "", "write cpu profile to `file`")
	memprofile := flag.String("memprofile", "", "write memory profile to `file`")
	fetchCount := flag.Int("count", 500, "number of repository objects to fetch")
	dbURI := flag.String("db", "corpus.db", "database to write to")
	flag.Parse()

	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			slog.Error("could not create CPU profile", "reason", err)
		}
		defer f.Close() // error handling omitted
		if err := pprof.StartCPUProfile(f); err != nil {
			slog.Error("could not start CPU profile", "reason", err)
		}
		defer pprof.StopCPUProfile()
	}
	defer func() {
		if *memprofile != "" {
			f, err := os.Create(*memprofile)
			if err != nil {
				slog.Error("could not create memory profile", "reason", err)
			}
			defer f.Close() // error handling omitted
			runtime.GC()    // get up-to-date statistics
			// Lookup("allocs") creates a profile similar to go test -memprofile.
			// Alternatively, use Lookup("heap") for a profile
			// that has inuse_space as the default index.
			if err := pprof.Lookup("allocs").WriteTo(f, 0); err != nil {
				slog.Error("could not write memory profile", "reason", err)
			}
		}
	}()

	if err := Main(ctx, *fetchCount, *dbURI); err != nil {
		slog.Error("exiting", "reason", err)
		code = 1
	}
}

func Main(ctx context.Context, count int, uri string) error {
	n := runtime.GOMAXPROCS(0)
	pool, err := sqlitex.NewPool(uri, sqlitex.PoolOptions{
		PoolSize: n,
	})
	if err != nil {
		return err
	}
	defer pool.Close()

	err = func() error {
		conn, err := pool.Take(ctx)
		if err != nil {
			return err
		}
		defer pool.Put(conn)
		return sqlitex.ExecuteScriptFS(conn, sql.FS, "init.sql", nil)
	}()
	if err != nil {
		return err
	}

	c, err := NewClient(new(http.Client), `https://quay.io/api/v1/`)
	if err != nil {
		return err
	}

	eg, ctx := errgroup.WithContext(ctx)
	repos := make(chan Repo, n)

	// Tags fetcher goroutines
	for range n {
		eg.Go(func() error {
			conn, err := pool.Take(ctx)
			if err != nil {
				return err
			}
			defer pool.Put(conn)
			for {
				var r Repo
				var ok bool
				select {
				case r, ok = <-repos:
					if !ok {
						return nil
					}
				case <-ctx.Done():
					return context.Cause(ctx)
				}
				l := slog.With(
					"namespace", r.Namespace,
					"repository", r.Name,
				)

				seq, check := c.Tags(ctx, r)
				tags := slices.Collect(seq)
				if err := check(); err != nil {
					return err
				}
				if len(tags) == 0 {
					l.DebugContext(ctx, "no tags found")
					continue
				}
				l.DebugContext(ctx, "got tags", "count", len(tags))

				var err error
				err = sqlitex.ExecuteFS(conn, sql.FS, "insert_namespace.sql", &sqlitex.ExecOptions{
					Args: []any{r.Namespace},
				})
				if err != nil {
					return err
				}
				err = sqlitex.ExecuteFS(conn, sql.FS, "insert_repository.sql", &sqlitex.ExecOptions{
					Args: []any{r.Name},
				})
				if err != nil {
					return err
				}

				for _, tag := range tags {
					err = sqlitex.ExecuteFS(conn, sql.FS, "insert_tag.sql", &sqlitex.ExecOptions{
						Args: []any{tag},
					})
					if err != nil {
						return err
					}
				}

				// Okay, now build all the rows to insert:
				todo := make([][]any, 0, len(tags))
				var nsID, rID int64 = -1, -1
				err = sqlitex.ExecuteFS(conn, sql.FS, "get_namespace_id.sql", &sqlitex.ExecOptions{
					Args: []any{r.Namespace},
					ResultFunc: func(stmt *sqlite.Stmt) (err error) {
						nsID = stmt.ColumnInt64(0)
						return err
					},
				})
				if err != nil {
					slog.ErrorContext(ctx, "query failed", "namespace", r.Namespace)
					return err
				}
				err = sqlitex.ExecuteFS(conn, sql.FS, "get_repository_id.sql", &sqlitex.ExecOptions{
					Args: []any{r.Name},
					ResultFunc: func(stmt *sqlite.Stmt) (err error) {
						rID = stmt.ColumnInt64(0)
						return err
					},
				})
				if err != nil {
					slog.ErrorContext(ctx, "query failed", "repository", r.Name)
					return err
				}
				for _, t := range tags {
					err = sqlitex.ExecuteFS(conn, sql.FS, "get_tag_id.sql", &sqlitex.ExecOptions{
						Args: []any{t},
						ResultFunc: func(stmt *sqlite.Stmt) error {
							tID := stmt.ColumnInt64(0)
							todo = append(todo, []any{nsID, rID, tID})
							return nil
						},
					})
					if err != nil {
						return err
					}
				}

				for _, args := range todo {
					err = sqlitex.ExecuteFS(conn, sql.FS, "insert_repo.sql", &sqlitex.ExecOptions{
						Args: args,
					})
					if err != nil {
						return err
					}
				}
				l.DebugContext(ctx, "inserted repos", "count", len(todo))
			}
		})
	}

	// Repo fetcher goroutine
	eg.Go(func() error {
		defer close(repos)
		n := 0

		defer func() {
			slog.InfoContext(ctx, "done paging repositories", "count", n, "limit", count)
		}()
		slog.InfoContext(ctx, "start paging repositories", "count", n, "limit", count)

		var err error
		seq, check := c.Repositories(ctx)
	Seq:
		for r := range seq {
			select {
			case repos <- r:
			case <-ctx.Done():
				err = context.Cause(ctx)
				break Seq
			}
			n++
			switch {
			case n%10 == 0:
				slog.DebugContext(ctx, "fetched repos", "count", n, "limit", count)
			case n >= count:
				break Seq
			}
		}

		if err := errors.Join(err, check()); err != nil {
			return err
		}

		return nil
	})

	// curl -H 'Accept: application/json' -H 'Content-Type: application/json' -H "Authorization: Bearer ${quay_token}" 'https://quay.io/api/v1/find/repositories?includeUsage=false&page_size=15&query=*&page=1' | jq '.results |= map_values(.href)'
	return eg.Wait()
}

type Repo struct {
	Namespace string
	Name      string
}

type client struct {
	c     *http.Client
	root  *url.URL
	Token string
}

func NewClient(c *http.Client, root string) (*client, error) {
	if !strings.HasSuffix("root", "/") {
		root += "/"
	}
	u, err := url.Parse(root)
	if err != nil {
		return nil, err
	}
	return &client{
		c:    c,
		root: u,
	}, nil
}

func (c *client) Repositories(ctx context.Context) (iter.Seq[Repo], func() error) {
	var errReturn error
	errFunc := func() error { return errReturn }
	seq := func(yield func(Repo) bool) {
		const maxPage = 100
		page := 1
		var buf bytes.Buffer
		buf.Grow(1 << 20)
		dup := make(map[uint64]struct{})
		seed := maphash.MakeSeed()

		endpt := c.root.JoinPath("find", "repositories")
		for {
			u := *endpt
			v := u.Query()
			v.Set("includeUsage", "false")
			v.Set("query", "*")
			v.Set("page", strconv.Itoa(page))
			u.RawQuery = v.Encode()

			req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
			if err != nil {
				errReturn = err
				return
			}
			req.Header.Set(`Accept`, `application/json`)
			if c.Token != "" {
				req.Header.Set(`Authorization`, `Bearer `+c.Token)
			}

			slog.DebugContext(ctx, "making request", "page", page, "url", u.String())
			res, err := c.c.Do(req)
			if err != nil {
				errReturn = err
				return
			}
			if res.StatusCode != http.StatusOK {
				res.Body.Close()
				errReturn = fmt.Errorf("unexpected response: %s", res.Status)
				return
			}
			buf.Reset()
			_, err = io.Copy(&buf, res.Body)
			if err := errors.Join(err, res.Body.Close()); err != nil {
				errReturn = err
				return
			}

			var findres FindRepositoriesResult
			if err := json.Unmarshal(buf.Bytes(), &findres); err != nil {
				errReturn = err
				return
			}

			// The Quay API is really odd here and will just return page 10
			// forever, so stop walking if this isn't the page requested.
			if page != findres.Page {
				break
			}

			for _, r := range findres.Results {
				id := maphash.String(seed, r.Href)
				_, ok := dup[id]
				if !ok {
					dup[id] = struct{}{}
					out := Repo{
						Namespace: r.Namespace.Name,
						Name:      r.Name,
					}
					if !yield(out) {
						return
					}
				} else {
					slog.DebugContext(ctx, "skip repo", "page", findres.Page, "additional", findres.Additional, "href", r.Href)
				}
			}

			page++
			// In case the Quay API starts being normal:
			if page >= maxPage || !findres.Additional {
				break
			}
		}
	}

	return seq, errFunc
}

type FindRepositoriesResult struct {
	Results []struct {
		Name      string `json:"name"`
		Namespace struct {
			Name string `json:"name"`
		} `json:"namespace"`
		Href string `json:"href"`
	} `json:"results"`
	Additional bool `json:"has_additional"`
	Page       int  `json:"page"`
}

func (c *client) Tags(ctx context.Context, repo Repo) (iter.Seq[string], func() error) {
	var errReturn error
	errFunc := func() error { return errReturn }
	seq := func(yield func(string) bool) {
		page := 1
		additional := true
		var buf bytes.Buffer
		buf.Grow(1 << 20)

		endpt := c.root.JoinPath("repository", repo.Namespace, repo.Name, "tag", "")
		for additional {
			u := *endpt
			v := u.Query()
			v.Set("page", strconv.Itoa(page))
			v.Set("limit", "100")
			v.Set("onlyActiveTags", "true")
			u.RawQuery = v.Encode()

			req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
			if err != nil {
				errReturn = err
				return
			}
			req.Header.Set(`Accept`, `application/json`)
			if c.Token != "" {
				req.Header.Set(`Authorization`, `Bearer `+c.Token)
			}

			res, err := c.c.Do(req)
			if err != nil {
				errReturn = err
				return
			}
			if res.StatusCode != http.StatusOK {
				res.Body.Close()
				errReturn = fmt.Errorf("unexpected response: %s", res.Status)
				return
			}
			buf.Reset()
			_, err = io.Copy(&buf, res.Body)
			if err := errors.Join(err, res.Body.Close()); err != nil {
				errReturn = err
				return
			}

			var tagsres ListTagsResult
			if err := json.Unmarshal(buf.Bytes(), &tagsres); err != nil {
				errReturn = err
				return
			}

			for _, t := range tagsres.Tags {
				if !t.IsList {
					continue
				}
				if !yield(t.Name) {
					return
				}
			}

			additional = tagsres.Additional
			page++
		}
	}

	return seq, errFunc
}

type ListTagsResult struct {
	Tags []struct {
		Name   string `json:"name"`
		Digest string `json:"manifest_digest"`
		IsList bool   `json:"is_manifest_list"`
	} `json:"tags"`
	Additional bool `json:"has_additional"`
	Page       int  `json:"page"`
}
