package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/google/go-github/v68/github"
)

type params struct {
	repo   string
	host   string
	scheme string
	client *http.Client
	stdout io.Writer
}

func main() {
	if err := run(params{
		scheme: "https",
		host:   "api.github.com",
		repo:   "bazelbuild/rules_swift",
		client: http.DefaultClient,
		stdout: os.Stdout,
	}); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(p params) error {
	// Fetch the pull requests from /repos/{repo}/pulls/ endpoint
	u := url.URL{
		Scheme: p.scheme,
		Host:   p.host,
		Path:   fmt.Sprintf("/repos/%s/pulls", p.repo),
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return err
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("request to %q recieved bad response code %d: %q", u.String(), resp.StatusCode, string(body))
	}

	var pulls []*github.PullRequest
	if err := json.NewDecoder(resp.Body).Decode(&pulls); err != nil {
		return err
	}

	if len(pulls) == 0 {
		return errors.New("got 0 pulls")
	}

	diffURL := pulls[0].DiffURL
	if diffURL == nil {
		return errors.New("no diff_url found")
	}

	ctx, cancel = context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	diffReq, err := http.NewRequestWithContext(ctx, http.MethodGet, *diffURL, nil)
	if err != nil {
		return err
	}

	diffResp, err := p.client.Do(diffReq)
	if err != nil {
		return err
	}
	defer diffResp.Body.Close()

	if diffResp.StatusCode != http.StatusOK && diffResp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(diffResp.Body)
		return fmt.Errorf("request to %q recieved bad response code %d: %q", *diffURL, diffResp.StatusCode, string(body))
	}

	io.Copy(p.stdout, diffResp.Body)
	return nil
}
