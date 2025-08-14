package httpx

import "net/http"

func WithBearer(base http.RoundTripper, token string) http.RoundTripper {
	return &withHeaders{base, map[string]string{"authorization": "Bearer " + token}}
}

type withHeaders struct {
	base    http.RoundTripper
	headers map[string]string
}

// RoundTrip implements http.RoundTripper.
func (t *withHeaders) RoundTrip(req *http.Request) (*http.Response, error) {
	for k, v := range t.headers {
		req.Header.Add(k, v)
	}
	return t.base.RoundTrip(req)
}
