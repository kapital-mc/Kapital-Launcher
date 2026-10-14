package services

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestReleaseOrderIsSemanticVersioning(t *testing.T) {
	// Each is newer than the one before it.
	order := []string{
		"v0.9.9", "v1.0.0-alpha.1", "v1.0.0-alpha.2", "v1.0.0-beta.1", "v1.0.0-beta.2",
		"v1.0.0-beta.10", "v1.0.0", "v1.0.1-beta.1", "v1.0.1", "v1.1.0", "v2.0.0",
	}
	for i := range order {
		for j := range order {
			a, aok := parseRelease(order[i])
			b, bok := parseRelease(order[j])
			if !aok || !bok {
				t.Fatalf("%q or %q does not parse", order[i], order[j])
			}
			if got, want := a.newerThan(b), i > j; got != want {
				t.Errorf("%s newer than %s = %v, want %v", order[i], order[j], got, want)
			}
		}
	}
}

func TestParseReleaseTakesTagsAndRefusesTheRest(t *testing.T) {
	for _, ok := range []string{"v1.0.0", "1.0.0", "v1.0.0-beta.3", "v10.20.30-alpha.4"} {
		if _, got := parseRelease(ok); !got {
			t.Errorf("%q refused", ok)
		}
	}
	for _, bad := range []string{
		"", "1.0.0-dev", "v1.0", "v1.0.0-rc.1", "v1.0.0-beta", "v1.0.0 ", "latest",
		"v1.0.0-beta.1+meta", "<script>", "v1.0.0\n",
	} {
		if _, got := parseRelease(bad); got {
			t.Errorf("%q accepted", bad)
		}
	}
}

// feed serves body with status at /version.json and counts the requests.
func feed(t *testing.T, status int, body string) (string, *atomic.Int32) {
	t.Helper()
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		if r.URL.RawQuery != "" || r.Method != http.MethodGet {
			t.Errorf("request %s %s", r.Method, r.URL)
		}
		w.WriteHeader(status)
		if _, err := w.Write([]byte(body)); err != nil {
			t.Error(err)
		}
	}))
	t.Cleanup(srv.Close)
	return srv.URL + "/version.json", &hits
}

func TestCheckUpdateComparesTheSitesTagWithTheBuild(t *testing.T) {
	cases := map[string]struct {
		current, served string
		available       bool
	}{
		"a newer beta":                          {"v1.0.0-beta.2", "v1.0.0-beta.3", true},
		"the release":                           {"v1.0.0-beta.2", "v1.0.0", true},
		"the same":                              {"v1.0.0-beta.2", "v1.0.0-beta.2", false},
		"an older one":                          {"v1.0.0-beta.3", "v1.0.0-beta.2", false},
		"a later prerelease of a later version": {"v1.0.0", "v1.1.0-beta.1", true},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			url, _ := feed(t, http.StatusOK, `{"version":"`+tc.served+`"}`)
			info := checkUpdate(context.Background(), newMapClient(time.Second), url, tc.current)
			if info.Available != tc.available || info.Latest != tc.served || info.Reason != "" {
				t.Fatalf("got %+v", info)
			}
			if info.Current != tc.current || info.Page != DownloadPage {
				t.Fatalf("got %+v", info)
			}
		})
	}
}

func TestCheckUpdateAsksNothingForADevBuild(t *testing.T) {
	url, hits := feed(t, http.StatusOK, `{"version":"v9.9.9"}`)
	info := checkUpdate(context.Background(), newMapClient(time.Second), url, "1.0.0-dev")
	if info.Available || info.Reason != "dev build" || hits.Load() != 0 {
		t.Fatalf("got %+v after %d requests", info, hits.Load())
	}
}

func TestCheckUpdateShowsNothingForAnAnswerThatIsNotAVersion(t *testing.T) {
	cases := map[string]struct {
		status int
		body   string
		reason string
	}{
		"not found":     {http.StatusNotFound, `{"version":"v9.9.9"}`, "no answer"},
		"a redirect":    {http.StatusFound, ``, "no answer"},
		"not JSON":      {http.StatusOK, `<html>`, "not a version"},
		"no version":    {http.StatusOK, `{}`, "not a version"},
		"an odd tag":    {http.StatusOK, `{"version":"v9.9.9-rc.1"}`, "not a version"},
		"a tag too big": {http.StatusOK, `{"version":"` + strings.Repeat("9", 5000) + `"}`, "not a version"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			url, _ := feed(t, tc.status, tc.body)
			info := checkUpdate(context.Background(), newMapClient(time.Second), url, "v1.0.0-beta.2")
			if info.Available || info.Latest != "" || info.Reason != tc.reason {
				t.Fatalf("got %+v", info)
			}
		})
	}
}

func TestCheckUpdateGivesUpOnASiteThatDoesNotAnswer(t *testing.T) {
	block := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { <-block }))
	t.Cleanup(func() { close(block); srv.Close() })
	info := checkUpdate(context.Background(), newMapClient(100*time.Millisecond), srv.URL, "v1.0.0")
	if info.Available || info.Reason != "timed out" {
		t.Fatalf("got %+v", info)
	}
}

func TestTheUpdatePreviewIsMadeUpAndOnlyItsOwn(t *testing.T) {
	info, ok := PreviewUpdate(PreviewLauncherUpdate, "v1.0.0-beta.2")
	if !ok || !info.Available || info.Latest != previewLauncherLatest || info.Page != DownloadPage {
		t.Fatalf("got %+v, %v", info, ok)
	}
	for _, other := range []string{"", PreviewPrismUpdate, PreviewPrismMissing} {
		if _, ok := PreviewUpdate(other, "v1.0.0"); ok {
			t.Errorf("%q gave an update", other)
		}
	}
	if _, ok := PreviewEngine(PreviewLauncherUpdate); ok {
		t.Error("the launcher update preview fakes the engine")
	}
}
