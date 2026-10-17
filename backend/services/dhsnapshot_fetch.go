package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// EventDHDownload is the Wails event carrying a models.DHDownloadProgress
// while a chapter's terrain downloads (issue 136).
const EventDHDownload = "dh:download"

const (
	// dhSmallTimeout bounds the GET of the pointer and of the index.
	dhSmallTimeout = 15 * time.Second
	// dhStall ends a part's GET that has received nothing for this long; a
	// slow connection that keeps receiving is never cut.
	dhStall = 30 * time.Second
	// dhAttempts is how many times a part is fetched before the download fails.
	dhAttempts = 3
)

// DHSource reads a published snapshot (issue 136). Every request is a GET on
// the host the manifest names, over https, with no redirect followed.
type DHSource struct {
	client *http.Client
	// stall and backoff are fields so a test does not wait them out.
	stall   time.Duration
	backoff time.Duration
}

// NewDHSource is the source the app uses.
func NewDHSource() *DHSource { return NewDHSourceWith(&http.Client{}) }

// NewDHSourceWith reads through the given client, a test server's for one,
// with redirects turned off on it.
func NewDHSourceWith(client *http.Client) *DHSource {
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	return &DHSource{client: client, stall: dhStall, backoff: 2 * time.Second}
}

// DHSnapshot is a snapshot as read: its index, where the index is (the base
// for the parts), and the index's hash, which names this snapshot in a
// download's journal.
type DHSnapshot struct {
	Index     DHIndex
	IndexURL  *url.URL
	IndexHash string
}

// Fetch reads the pointer the manifest names, then the index it points to.
// pointer must have passed checkDHSnapshotURL.
func (s *DHSource) Fetch(ctx context.Context, pointer string) (DHSnapshot, error) {
	pu, err := url.Parse(pointer)
	if err != nil {
		return DHSnapshot{}, err
	}
	data, err := s.getSmall(ctx, pu, dhPointerMax)
	if err != nil {
		return DHSnapshot{}, err
	}
	p, err := ParseDHPointer(data)
	if err != nil {
		return DHSnapshot{}, err
	}
	iu, err := dhIndexURL(pu, p)
	if err != nil {
		return DHSnapshot{}, err
	}
	data, err = s.getSmall(ctx, iu, dhIndexMax)
	if err != nil {
		return DHSnapshot{}, err
	}
	ix, err := ParseDHIndex(data)
	if err != nil {
		return DHSnapshot{}, err
	}
	sum := sha256.Sum256(data)
	return DHSnapshot{Index: ix, IndexURL: iu, IndexHash: hex.EncodeToString(sum[:])}, nil
}

// getSmall is one bounded GET of a small file: 15 s, a 200 and at most limit
// bytes.
func (s *DHSource) getSmall(ctx context.Context, u *url.URL, limit int64) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, dhSmallTimeout)
	defer cancel()
	resp, err := s.get(ctx, u)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close() //nolint:errcheck // read-only response
	data, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil {
		return nil, fmt.Errorf("snapshot: %w", err)
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("snapshot: %s is over %d bytes", u.Path, limit)
	}
	return data, nil
}

// get starts a GET and refuses anything but a 200.
func (s *DHSource) get(ctx context.Context, u *url.URL) (*http.Response, error) {
	if u.Scheme != "https" {
		return nil, fmt.Errorf("snapshot: refusing %s", u.Redacted())
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", prismUserAgent)
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("snapshot: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close() //nolint:errcheck // the status is the answer
		return nil, &dhStatusError{path: u.Path, status: resp.StatusCode}
	}
	return resp, nil
}

// dhStatusError is a response other than 200. A 5xx is worth another try.
type dhStatusError struct {
	path   string
	status int
}

func (e *dhStatusError) Error() string {
	return fmt.Sprintf("snapshot: %s answered %d %s", e.path, e.status, http.StatusText(e.status))
}

// fetchPart writes one part to w, retrying a cut connection, a server error or
// a part whose bytes do not match its hash. received is told of every chunk.
func (s *DHSource) fetchPart(ctx context.Context, u *url.URL, part DHPart, w io.WriterAt, off int64, received func(int64)) error {
	var err error
	for attempt := range dhAttempts {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(s.backoff * time.Duration(attempt)):
			}
		}
		var got int64
		err = s.fetchPartOnce(ctx, u, part, w, off, func(n int64) {
			got += n
			received(n)
		})
		if err == nil || ctx.Err() != nil || !retryable(err) {
			return err
		}
		// What this attempt counted is fetched again.
		received(-got)
	}
	return err
}

func retryable(err error) bool {
	var status *dhStatusError
	if errors.As(err, &status) {
		return status.status >= 500
	}
	return true
}

// fetchPartOnce streams one part to w at off, hashing as it goes. The bytes
// may land before the hash is checked; the caller only counts a part done when
// this returns nil, and a later attempt writes over them.
func (s *DHSource) fetchPartOnce(ctx context.Context, u *url.URL, part DHPart, w io.WriterAt, off int64, received func(int64)) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	stalled := time.AfterFunc(s.stall, cancel)
	defer stalled.Stop()
	resp, err := s.get(ctx, u)
	if err != nil {
		return err
	}
	defer resp.Body.Close() //nolint:errcheck // read-only response
	h := sha256.New()
	counter := &dhCounter{report: func(n int64) {
		stalled.Reset(s.stall)
		received(n)
	}}
	n, err := io.Copy(io.MultiWriter(io.NewOffsetWriter(w, off), h, counter), io.LimitReader(resp.Body, part.Size+1))
	if err != nil {
		return fmt.Errorf("snapshot: part %s: %w", part.SHA256[:12], err)
	}
	if n != part.Size {
		return fmt.Errorf("snapshot: part %s is %d bytes, the index says %d", part.SHA256[:12], n, part.Size)
	}
	if got := hex.EncodeToString(h.Sum(nil)); got != part.SHA256 {
		return fmt.Errorf("snapshot: part %s does not match its hash", part.SHA256[:12])
	}
	return nil
}

// dhCounter reports each chunk written through it.
type dhCounter struct{ report func(int64) }

func (c *dhCounter) Write(b []byte) (int, error) {
	c.report(int64(len(b)))
	return len(b), nil
}
