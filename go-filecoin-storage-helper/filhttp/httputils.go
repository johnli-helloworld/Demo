package filhttp

import (
	"net/http"
	gohttp "net/http"
)

type Httputils struct {
	url     string
	httpcli gohttp.Client
	Headers http.Header
}

func Newhttp(url string) *Httputils {
	if url == "" {
		url = "192.168.1.189:3453"
	}
	return &Httputils{
		url: url,
	}
}

func (api *Httputils) Request(command string, args ...string) RequestBuilder {
	headers := make(map[string]string)
	if api.Headers != nil {
		for k := range api.Headers {
			headers[k] = api.Headers.Get(k)
		}
	}
	return &requestBuilder{
		command: command,
		args:    args,
		shell:   api,
		headers: headers,
	}
}
