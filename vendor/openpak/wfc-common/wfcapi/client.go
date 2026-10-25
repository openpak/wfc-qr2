package wfcapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

var client = &http.Client{Timeout: 3 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}

func Call(ctx context.Context, env, path string, in, out any) error {
	raw := os.Getenv(env)
	u, e := url.Parse(raw)
	if e != nil || u.Host == "" || u.User != nil || u.Scheme != "http" || u.Path != "" || u.RawQuery != "" || u.Fragment != "" {
		return fmt.Errorf("%s must be an internal HTTP origin", env)
	}
	body, e := json.Marshal(in)
	if e != nil {
		return e
	}
	req, e := http.NewRequestWithContext(ctx, "POST", strings.TrimRight(raw, "/")+path, bytes.NewReader(body))
	if e != nil {
		return e
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-WFC-Internal-Key", os.Getenv("WFC_INTERNAL_KEY"))
	res, e := client.Do(req)
	if e != nil {
		return fmt.Errorf("%s unavailable: %w", env, e)
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return fmt.Errorf("%s returned %d", env, res.StatusCode)
	}
	if out == nil {
		_, e = io.Copy(io.Discard, io.LimitReader(res.Body, 2<<20))
		return e
	}
	return json.NewDecoder(io.LimitReader(res.Body, 2<<20)).Decode(out)
}
func Decode(w http.ResponseWriter, r *http.Request, v any) bool {
	d := json.NewDecoder(http.MaxBytesReader(w, r.Body, 256<<10))
	d.DisallowUnknownFields()
	if e := d.Decode(v); e != nil {
		http.Error(w, "invalid request", 400)
		return false
	}
	var extra any
	if e := d.Decode(&extra); e != io.EOF {
		http.Error(w, "invalid request", 400)
		return false
	}
	return true
}
func Reply(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	json.NewEncoder(w).Encode(v)
}
