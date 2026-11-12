package services

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
)

// The play time on the account page is the time a game log shows in a world
// (issue 341, ADR-2's twenty-third amendment). A log is read a line at a time
// for the markers below and for nothing else: a line is matched and dropped,
// and what comes out is a number of seconds.
//
// The markers were read off real logs on 2026-11-08: NeoForge 1.21.1 for all of
// them, and the shape of Fabric 1.20.6's lines, which has no date and no logger.
const (
	markerIntegrated   = "Starting integrated minecraft server"
	markerServerStop   = "Stopping server"
	markerKicked       = "Client disconnected with reason"
	markerDHDisconnect = "Client on ClientOnly mode disconnecting"
	markerChat         = "[CHAT] "
	markerSystemChat   = "[System] [CHAT] "
	markerAway         = " is now AFK"
	markerBack         = " is no longer AFK"

	// afkIdleSeconds is Frangfurd Essentials' afk.minutes as the server ships it:
	// a player announced as AFK has already been idle this long.
	afkIdleSeconds = 5 * 60
	// clockWrapSeconds is how far a log's clock has to fall to be the next day.
	// A smaller fall is the clock being put back.
	clockWrapSeconds = 12 * 60 * 60
	secondsPerDay    = 24 * 60 * 60
	// playLogLineBytes is the longest line looked at. A longer one is a dump, and
	// no marker is that long.
	playLogLineBytes = 64 << 10
)

// connectingLine is Minecraft's own line when it connects to a server, and not
// a mod connecting to something else.
var connectingLine = regexp.MustCompile(`^Connecting to \S+, \d+$`)

// replayMarkers are lines only a Flashback replay writes. A replay is a local
// world to Minecraft, and watching one is not play.
var replayMarkers = []string{"ServerLevel[Replay]", "Discarding replay payload"}

// playRun is what one game log adds up to: the seconds played, and whether the
// log ended with the player still in a world or AFK, for a log that carries on
// from it.
type playRun struct {
	Seconds int64 `json:"seconds"`
	InWorld bool  `json:"inWorld,omitempty"`
	Away    bool  `json:"away,omitempty"`
}

// playParser adds up one log. It keeps seconds and flags, and the player's name
// while it runs, which is compared with and never stored.
type playParser struct {
	// before is how the previous log ended. The game starts a new log at
	// midnight while it runs, and that log picks up where the last one stopped.
	before playRun
	// lookup finds the player's name when this log does not say it, as a log
	// begun at midnight does not.
	lookup func() string
	player string

	started bool
	clock   int64
	now     int64

	connecting bool
	inWorld    bool
	replay     bool
	away       bool
	worldAt    int64
	awayAt     int64
	backAt     int64
	afk        int64
	seconds    int64
}

// measurePlay reads a log's text and returns what it adds up to, given how the
// log before it ended. Nothing past maxUnpackedRunLog is read.
func measurePlay(r io.Reader, before playRun, lookup func() string) (playRun, error) {
	p := playParser{before: before, lookup: lookup}
	br := bufio.NewReaderSize(io.LimitReader(r, maxUnpackedRunLog), playLogLineBytes)
	for {
		line, err := br.ReadSlice('\n')
		switch {
		case err == nil:
			p.line(strings.TrimRight(string(line), "\r\n"))
		case errors.Is(err, bufio.ErrBufferFull):
			if err := skipLine(br); err != nil {
				return playRun{}, err
			}
		case errors.Is(err, io.EOF):
			// What is left has no newline: latest.log is being written.
			return p.finish(), nil
		default:
			return playRun{}, fmt.Errorf("read game log: %w", err)
		}
	}
}

// skipLine reads to the end of a line longer than the reader's buffer.
func skipLine(br *bufio.Reader) error {
	for {
		_, err := br.ReadSlice('\n')
		switch {
		case err == nil, errors.Is(err, io.EOF):
			return nil
		case !errors.Is(err, bufio.ErrBufferFull):
			return fmt.Errorf("read game log: %w", err)
		}
	}
}

// line reads one line of the log. Only a line that opens with a time counts,
// which leaves out a stack trace.
func (p *playParser) line(line string) {
	clock, rest, ok := splitLogLine(line)
	if !ok {
		return
	}
	p.tick(clock, rest)
	if p.player == "" && (strings.Contains(rest, "Setting user: ") || strings.Contains(rest, "--username")) {
		p.player = playerName([]byte(line))
	}
	if at := strings.Index(rest, "]: "); at >= 0 {
		p.message(line, rest, rest[at+len("]: "):])
	}
}

// tick moves the log's time on. A log's clock is only ever read as a time of
// day, because the date before it is written in the player's language and
// Fabric's lines have none.
func (p *playParser) tick(clock int64, rest string) {
	if !p.started {
		p.started = true
		p.clock = clock
		p.begin(rest)
		return
	}
	switch {
	case clock >= p.clock:
		p.now += clock - p.clock
		p.clock = clock
	case p.clock-clock > clockWrapSeconds:
		p.now += clock + secondsPerDay - p.clock
		p.clock = clock
	}
}

// begin is the first line of the log. The game's start is written by its main
// thread, so any other first line is a log the game opened while it ran.
func (p *playParser) begin(rest string) {
	if strings.HasPrefix(rest, " [main/") || !p.before.InWorld {
		return
	}
	p.enter()
	if p.before.Away {
		p.away = true
		p.awayAt = p.now
	}
}

// message acts on a line's text. Chat is other people's words, so a chat line
// can only be an AFK announcement and never one of the other markers.
func (p *playParser) message(line, rest, msg string) {
	if strings.Contains(msg, markerChat) {
		if p.inWorld && strings.HasPrefix(msg, markerSystemChat) {
			p.chat(msg[len(markerSystemChat):])
		}
		return
	}
	switch {
	case strings.HasPrefix(msg, "Connecting to ") && connectingLine.MatchString(msg):
		p.leave()
		p.connecting = true
	case p.connecting && isJoinedLine(rest, msg):
		p.connecting = false
		p.enter()
	case strings.Contains(msg, markerIntegrated):
		p.leave()
		p.connecting = false
		p.enter()
	case isLeftWorldLine(line, msg):
		p.connecting = false
		p.leave()
	case p.inWorld && isReplayLine(msg):
		p.replay = true
	}
}

// isJoinedLine is the client taking in the server's advancements, the first
// line a connection that worked writes. A connection that failed never does.
func isJoinedLine(rest, msg string) bool {
	return strings.HasPrefix(rest, " [Render thread/") &&
		strings.HasPrefix(msg, "Loaded ") && strings.HasSuffix(msg, " advancements")
}

// isLeftWorldLine is any line that says the world is gone. Vanilla writes none
// for leaving a server on purpose: there the line is Distant Horizons'.
func isLeftWorldLine(line, msg string) bool {
	return msg == markerServerStop ||
		strings.Contains(msg, markerKicked) ||
		strings.Contains(msg, markerDHDisconnect) ||
		isStoppingLine(line)
}

func isReplayLine(msg string) bool {
	for _, marker := range replayMarkers {
		if strings.Contains(msg, marker) {
			return true
		}
	}
	return false
}

// chat follows the player's own AFK lines, from the server (a system message).
// The body has to be exactly the announcement, after any tags such as
// "[Server] ", so a player typing the same words does not match.
func (p *playParser) chat(body string) {
	wentAway := strings.HasSuffix(body, markerAway)
	cameBack := strings.HasSuffix(body, markerBack)
	if !wentAway && !cameBack {
		return
	}
	for strings.HasPrefix(body, "[") {
		end := strings.Index(body, "] ")
		if end < 0 {
			break
		}
		body = body[end+len("] "):]
	}
	name := p.name()
	if name == "" {
		return
	}
	switch {
	case wentAway && !p.away && body == name+markerAway:
		p.away = true
		p.awayAt = max(p.worldAt, p.backAt, p.now-afkIdleSeconds)
	case cameBack && p.away && body == name+markerBack:
		p.endAway()
		p.backAt = p.now
	}
}

// name is the player of this log, asked of lookup once if the log has not said.
func (p *playParser) name() string {
	if p.player == "" && p.lookup != nil {
		p.player = p.lookup()
		p.lookup = nil
	}
	return p.player
}

func (p *playParser) enter() {
	p.inWorld = true
	p.replay = false
	p.away = false
	p.worldAt = p.now
	p.backAt = p.now
	p.afk = 0
}

func (p *playParser) endAway() {
	p.afk += p.now - p.awayAt
	p.away = false
}

// leave closes the time in a world, if there is any open.
func (p *playParser) leave() {
	if !p.inWorld {
		return
	}
	if p.away {
		p.endAway()
	}
	if !p.replay {
		p.seconds += max(p.now-p.worldAt-p.afk, 0)
	}
	p.inWorld = false
}

// finish closes what is open at the log's last line and says how the log ended.
func (p *playParser) finish() playRun {
	run := playRun{InWorld: p.inWorld, Away: p.away}
	p.leave()
	run.Seconds = p.seconds
	return run
}

// splitLogLine parses a line's opening time, "[28Oct2026 01:51:49.027]" or
// "[01:51:49]", into seconds of the day and returns what follows it.
func splitLogLine(line string) (int64, string, bool) {
	if line == "" || line[0] != '[' {
		return 0, "", false
	}
	end := strings.IndexByte(line, ']')
	if end < 0 {
		return 0, "", false
	}
	stamp := line[1:end]
	clock, ok := parseLogClock(stamp[strings.LastIndexByte(stamp, ' ')+1:])
	if !ok {
		return 0, "", false
	}
	return clock, line[end+1:], true
}

func parseLogClock(s string) (int64, bool) {
	if len(s) < len("00:00:00") || s[2] != ':' || s[5] != ':' {
		return 0, false
	}
	var clock int64
	for _, part := range []string{s[:2], s[3:5], s[6:8]} {
		n, err := strconv.ParseUint(part, 10, 8)
		if err != nil {
			return 0, false
		}
		clock = clock*60 + int64(n)
	}
	return clock, true
}
