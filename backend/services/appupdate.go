package services

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"kapital/backend/models"
)

// The update notice (ADR-9 amendment): once a start, the launcher reads the
// download site's version.json, which the site's build writes with the release
// its Download button offers, and compares that tag with its own. A newer one
// opens a popup with the site's address. The address is this constant, never
// one from a manifest or the caller, and the request sends nothing but itself:
// no version, no id, no query.
const (
	// DownloadPage is the download site, the page the popup opens.
	// aislop-ignore-next-line ai-slop/hardcoded-url -- fixed on purpose: no manifest or caller may name another (SECURITY_CHECKLIST S4.9)
	DownloadPage = "https://kapital-launcher.pages.dev/"
	// updateFeed is the site's version file.
	updateFeed = DownloadPage + "version.json"
	// updateCheckTimeout bounds the whole request, the connection included.
	updateCheckTimeout = 8 * time.Second
	// maxUpdateBody is more than the file ever holds.
	maxUpdateBody = 4 << 10
)

// releaseTag is the shape of a release tag, the site's own rule (site/vite.config.js,
// TAG), with the leading v optional so a build's own version reads the same.
var releaseTag = regexp.MustCompile(`^v?(\d{1,6})\.(\d{1,6})\.(\d{1,6})(?:-(alpha|beta)\.(\d{1,6}))?$`)

// releaseVersion is a parsed tag: the three numbers, and for a prerelease its
// channel and number.
type releaseVersion struct {
	nums     [3]int
	pre      string
	preCount int
}

// parseRelease reads a tag, or reports that it is not one. A dev build's
// "1.0.0-dev" is not, so it is never compared.
func parseRelease(s string) (releaseVersion, bool) {
	m := releaseTag.FindStringSubmatch(s)
	if m == nil {
		return releaseVersion{}, false
	}
	var v releaseVersion
	for i := range 3 {
		v.nums[i], _ = strconv.Atoi(m[i+1]) //nolint:errcheck // the pattern holds only digits
	}
	if m[4] != "" {
		v.pre = m[4]
		v.preCount, _ = strconv.Atoi(m[5]) //nolint:errcheck // the pattern holds only digits
	}
	return v, true
}

// preRank orders the channels: an alpha before a beta, both before the release.
var preRank = map[string]int{"alpha": 0, "beta": 1, "": 2}

// newerThan is semantic versioning's precedence for these tags: the numbers,
// then a release above its prereleases, then alpha below beta, then the count.
func (v releaseVersion) newerThan(o releaseVersion) bool {
	for i := range 3 {
		if v.nums[i] != o.nums[i] {
			return v.nums[i] > o.nums[i]
		}
	}
	if preRank[v.pre] != preRank[o.pre] {
		return preRank[v.pre] > preRank[o.pre]
	}
	return v.preCount > o.preCount
}

// CheckUpdate says whether the download site offers a build newer than
// current. A build that is not a release (a dev build) asks nothing. A failure
// is a reason, never an error: the launcher carries on without a notice.
func CheckUpdate(ctx context.Context, current string) models.UpdateInfo {
	return checkUpdate(ctx, newMapClient(updateCheckTimeout), updateFeed, current)
}

// checkUpdate is CheckUpdate with the client and the address, so a test can
// point it at a local server.
func checkUpdate(ctx context.Context, client *http.Client, feed, current string) models.UpdateInfo {
	info := models.UpdateInfo{Current: current, Page: DownloadPage}
	own, ok := parseRelease(current)
	if !ok {
		info.Reason = "dev build"
		return info
	}
	latest, theirs, reason := readUpdateFeed(ctx, client, feed)
	if reason != "" {
		info.Reason = reason
		slog.Info("update check", "reason", reason)
		return info
	}
	info.Latest = latest
	info.Available = theirs.newerThan(own)
	slog.Info("update check", "latest", latest, "available", info.Available)
	return info
}

// readUpdateFeed fetches the version file and returns its tag with the version
// it parses to, or the reason there is none. No redirect is followed, only a 200
// counts, and at most maxUpdateBody bytes are read.
func readUpdateFeed(ctx context.Context, client *http.Client, feed string) (string, releaseVersion, string) {
	none := releaseVersion{}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, feed, nil)
	if err != nil {
		return "", none, "no answer"
	}
	req.Header.Set("User-Agent", prismUserAgent)
	resp, err := client.Do(req)
	if err != nil {
		var netErr net.Error
		if errors.Is(err, context.DeadlineExceeded) || (errors.As(err, &netErr) && netErr.Timeout()) {
			return "", none, "timed out"
		}
		// The error's text is left out: it holds the address and nothing more useful.
		return "", none, "no answer"
	}
	defer resp.Body.Close() //nolint:errcheck // read-only response
	if resp.StatusCode != http.StatusOK {
		return "", none, "no answer"
	}
	var body struct {
		Version string `json:"version"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxUpdateBody)).Decode(&body); err != nil {
		return "", none, "not a version"
	}
	v, ok := parseRelease(body.Version)
	if !ok {
		return "", none, "not a version"
	}
	return body.Version, v, ""
}
