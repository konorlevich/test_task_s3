package register

import (
	"fmt"
	"net/http"
	"net/url"
	"time"
)

const (
	formParamHostname = "hostname"
	formParamPort     = "port"
)

func Register(serverUrl *url.URL, hostName, port string) error {
	if serverUrl == nil {
		return fmt.Errorf("register server url is empty")
	}
	vals := url.Values{}
	if hostName != "" {
		vals.Add(formParamHostname, hostName)
	}
	if port != "" {
		vals.Add(formParamPort, port)
	}
	res, err := (&http.Client{Timeout: 5 * time.Second}).
		PostForm(
			serverUrl.String(),
			vals,
		)
	if err != nil {
		return err
	}
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("returned status code: %d", res.StatusCode)
	}

	return nil
}
