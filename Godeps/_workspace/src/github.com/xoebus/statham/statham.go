package statham

import "net/http"

func NewTransport(defaultTr http.RoundTripper, mapping Mapping) http.RoundTripper {
	return &roundTripper{
		defaultTripper: defaultTr,
		mapping:        mapping,
	}
}

type Mapping map[string]http.RoundTripper

type roundTripper struct {
	defaultTripper http.RoundTripper
	mapping        Mapping
}

func (rt *roundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	transport, found := rt.mapping[req.URL.Host]
	if !found {
		return rt.defaultTripper.RoundTrip(req)
	}

	return transport.RoundTrip(req)
}
