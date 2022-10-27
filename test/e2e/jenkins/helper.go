package jenkins

import (
	"fmt"
	"io"
	"net/http"
)

// Client struct encapsulating ability to interact with Jenkins instance
type Client struct {
	URL      string
	Username string
	APIToken string
}

// NewJenkinsClient creates a new Jenkins Client instance.
func NewJenkinsClient(url string, username string, token string) Client {
	jenkins := Client{
		URL:      url,
		Username: username,
		APIToken: token,
	}
	return jenkins
}

// JobExists returns true if the specified Jenkins Job exists, false otherwise/
// An error is returned in case an error occurs.
func (jc *Client) JobExists(job string) (bool, error) {
	fullURL := fmt.Sprintf("%s/job/%s", jc.URL, job)

	status, _, err := jc.doRequest("GET", fullURL, nil, nil)
	if err != nil {
		return false, err
	}

	if *status == 200 {
		return true, nil
	}
	return false, nil
}

func (jc *Client) doRequest(method string, url string, data io.Reader, headers map[string]string) (*int, []byte, error) {
	var body []byte

	request, err := http.NewRequest(method, url, data)
	if err != nil {
		return nil, body, err
	}

	request.SetBasicAuth(jc.Username, jc.APIToken)
	for key, value := range headers {
		request.Header.Add(key, value)
	}

	client := &http.Client{}
	resp, err := client.Do(request)
	if err != nil {
		return nil, body, err
	}

	defer func() {
		_ = resp.Body.Close()
	}()
	body, err = io.ReadAll(resp.Body)
	if err != nil {
		return nil, body, err
	}

	return &resp.StatusCode, body, nil
}

// CreateJob creates a new Jenkins Job using the config.xml passed via data.
func (jc *Client) CreateJob(jobName string, data io.Reader) error {
	fullURL := fmt.Sprintf("%s/createItem?name=%s", jc.URL, jobName)

	headers := map[string]string{"Content-Type": "text/xml"}
	status, _, err := jc.doRequest("POST", fullURL, data, headers)
	if err != nil {
		return err
	}

	if *status != 200 {
		return fmt.Errorf("error creating job: '%s'", jobName)
	}

	return nil
}

// DeleteJob deletes the Job specified by jobName.
func (jc *Client) DeleteJob(jobName string) error {
	fullURL := fmt.Sprintf("%s/job/%s/doDelete", jc.URL, jobName)

	status, _, err := jc.doRequest("POST", fullURL, nil, nil)
	if err != nil {
		return err
	}

	if *status != 200 {
		return fmt.Errorf("error deleting job: '%s'", jobName)
	}

	return nil
}
