package engines

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"time"
)

var (
	sleep = time.Sleep
)

// GetBuildID calls out to the build number URL in order to get a build number for the build.
func GetBuildID(key, buildNumURL string) (string, error) {
	if buildNumURL == "" {
		return "", nil
	}
	var err error
	parsedURL, err := url.Parse(buildNumURL)
	if err != nil {
		return "", fmt.Errorf("invalid build number url: %v", err)
	}
	parsedURL.Path = path.Join(parsedURL.Path, "vend", key)
	sleepDuration := 100 * time.Millisecond
	for retries := 0; retries < 10; retries++ {
		if retries > 0 {
			sleep(sleepDuration)
			sleepDuration = sleepDuration * 2
		}
		var resp *http.Response
		resp, err = http.Get(parsedURL.String())
		if err != nil {
			continue
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			err = fmt.Errorf("got unexpected response from build number service: %v", resp.Status)
			continue
		}
		var buf []byte
		buf, err = ioutil.ReadAll(resp.Body)
		if err == nil {
			return string(buf), nil
		}
		return "", err
	}
	return "", err
}
