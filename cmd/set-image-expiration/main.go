// Set-image-expiration is a small command to set the tag expiration for a
// container image in a Quay registry.
//
// This is needed to have an expiring and floating tag, one where the latest
// version is always present, but previous ones are allowed to expire.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"runtime/debug"
	"time"
)

var userAgent = `github.com/quay/clair-workflows/cmd/set-image-expiration`

func init() {
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return
	}
	v := "???"
	if bi.Main.Version != "" {
		v = bi.Main.Version
	}
	userAgent += " " + v
}

func main() {
	code := 1
	defer func() {
		if code != 0 {
			os.Exit(code)
		}
	}()
	ctx := context.Background()
	ctx, undoSig := signal.NotifyContext(ctx, os.Interrupt, os.Kill)
	defer undoSig()
	ctx, undoOS := osHandleSignals(ctx)
	defer undoOS()

	var opt Options
	flag.DurationVar(&opt.Duration, "duration", 2*7*24*time.Hour, "set expiration, as a duration from now")
	flag.StringVar(&opt.Token, "token", "", "Quay API token (also taken from QUAY_TOKEN environment variable)")
	flag.StringVar(&opt.Host, "host", "quay.io", "Quay host")
	flag.StringVar(&opt.Repo, "repo", "", "container image namespace+repository")
	flag.StringVar(&opt.Tag, "tag", "latest", "tag to operate on")
	flag.Parse()

	if opt.Token == "" {
		var ok bool
		opt.Token, ok = os.LookupEnv("QUAY_TOKEN")
		if !ok {
			fmt.Fprintln(os.Stderr, "no API token provided")
			return
		}
	}
	if opt.Repo == "" {
		fmt.Fprintln(os.Stderr, "no repository provided")
		return
	}

	if err := Main(ctx, opt); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return
	}
	code = 0
}

type Options struct {
	Duration time.Duration
	Token    string
	Host     string
	Repo     string
	Tag      string
}

func Main(ctx context.Context, opt Options) error {
	u := fmt.Sprintf("https://%s/api/v1/repository/%s/tag/%s", opt.Host, opt.Repo, opt.Tag)
	e := time.Now().Unix() + int64(opt.Duration.Truncate(time.Second).Seconds())
	b, err := json.Marshal(map[string]int64{"expiration": e})
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, u, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Authorization", "Bearer "+opt.Token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected response: %s", res.Status)
	}

	return nil
}
