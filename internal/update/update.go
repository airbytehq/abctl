package update

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"golang.org/x/mod/semver"
)

var ErrDevVersion = errors.New("dev version not supported")

type doer interface {
	Do(req *http.Request) (*http.Response, error)
}

// Check looks to see if there is a newer version of abctl.
// This is accomplished by fetching the latest github tag and comparing it to the version provided.
// Returns the latest version, or an empty string if we're already running the latest version.
// Will return ErrDevVersion if the build.Version is currently set to "dev".
func Check(ctx context.Context, doer doer, version string) (string, error) {
	if version == "dev" {
		return "", ErrDevVersion
	}

	latest, err := latest(ctx, doer)
	if err != nil {
		return "", err
	}

	if semver.Compare(version, latest) < 0 {
		return latest, nil
	}

	// if we're here then our version is the latest
	return "", nil
}

const url = "https://api.github.com/repos/airbytehq/abctl/releases/latest"

func latest(ctx context.Context, doer doer) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("unable to create request: %w", err)
	}

	res, err := doer.Do(req)
	if err != nil {
		return "", fmt.Errorf("unable to do request: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unable to do request, status code: %d", res.StatusCode)
	}

	var latest struct {
		TagName string `json:"tag_name"`
	}

	decoder := json.NewDecoder(res.Body)
	if err := decoder.Decode(&latest); err != nil {
		return "", fmt.Errorf("unable to decode response: %w", err)
	}

	if !semver.IsValid(latest.TagName) {
		return "", fmt.Errorf("invalid semver tag: %s", latest.TagName)
	}

	return latest.TagName, nil
}
