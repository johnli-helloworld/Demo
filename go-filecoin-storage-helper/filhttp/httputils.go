package filhttp

import (
	"net/http"
	gohttp "net/http"
)

type HttpAPI struct {
	url     string
	httpcli gohttp.Client
	Headers http.Header
}

func Newhttp(url string) *HttpAPI {
	if url == "" {
		url = "127.0.0.1:3453"
	}
	return &HttpAPI{
		url: url,
	}
}

func (api *HttpAPI) Request(command string, args ...string) RequestBuilder {
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

func (api *HttpAPI) Storage() Storage {
	return (*StorageAPI)(api)
}
