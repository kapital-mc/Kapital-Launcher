package services

import (
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

var playBase = time.Date(2026, 10, 27, 12, 0, 0, 0, time.UTC)

type playInstance struct {
	t    *testing.T
	dir  string
	logs string
}

func newPlayInstance(t *testing.T) playInstance {
	t.Helper()
	dir := t.TempDir()
	logs := filepath.Join(dir, "minecraft", "logs")
	if err := os.MkdirAll(logs, 0o755); err != nil {
		t.Fatal(err)
	}
	return playInstance{t: t, dir: dir, logs: logs}
}

func gzipped(t *testing.T, text string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	if _, err := zw.Write([]byte(text)); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func (p playInstance) put(name string, data []byte, hour int) {
	p.t.Helper()
	path := filepath.Join(p.logs, name)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		p.t.Fatal(err)
	}
	at := playBase.Add(time.Duration(hour) * time.Hour)
	if err := os.Chtimes(path, at, at); err != nil {
		p.t.Fatal(err)
	}
}

func (p playInstance) archive(name, text string, hour int) {
	p.t.Helper()
	p.put(name, gzipped(p.t, text), hour)
}

func (p playInstance) remove(name string) {
	p.t.Helper()
	if err := os.Remove(filepath.Join(p.logs, name)); err != nil {
		p.t.Fatal(err)
	}
}

func wantPlay(t *testing.T, l *PlayLedger, inst playInstance, want int64) {
	t.Helper()
	got, err := l.Count("frangfurd", inst.dir)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("count = %d, want %d", got, want)
	}
}

func playedFor(seconds int) string {
	return logBoot(start) + logConnect(start) + logJoined(start) + logLeft(start+seconds) + logStopping(start+seconds+5)
}

func TestPlayLedgerAddsTheLogsAndTheLatest(t *testing.T) {
	inst := newPlayInstance(t)
	inst.archive("2026-10-27-1.log.gz", playedFor(100), 1)
	inst.archive("2026-10-27-2.log.gz", playedFor(200), 2)
	inst.put(latestLogName, []byte(playedFor(30)), 3)
	wantPlay(t, NewPlayLedger(t.TempDir()), inst, 330)
}

func TestPlayLedgerIgnoresTheDebugLogs(t *testing.T) {
	inst := newPlayInstance(t)
	inst.archive("debug-1.log.gz", playedFor(999), 1)
	inst.put("debug.log", []byte(playedFor(999)), 2)
	inst.archive("2026-10-27-1.log.gz", playedFor(100), 3)
	wantPlay(t, NewPlayLedger(t.TempDir()), inst, 100)
}

func TestPlayLedgerKeepsTheTimeOfADeletedLog(t *testing.T) {
	data := t.TempDir()
	inst := newPlayInstance(t)
	inst.archive("2026-10-27-1.log.gz", playedFor(100), 1)
	wantPlay(t, NewPlayLedger(data), inst, 100)
	inst.remove("2026-10-27-1.log.gz")
	wantPlay(t, NewPlayLedger(data), inst, 100)
}

func TestPlayLedgerReadsAFinishedLogOnce(t *testing.T) {
	data := t.TempDir()
	inst := newPlayInstance(t)
	name := "2026-10-27-1.log.gz"
	inst.archive(name, playedFor(100), 1)
	l := NewPlayLedger(data)
	wantPlay(t, l, inst, 100)
	info, err := os.Stat(filepath.Join(inst.logs, name))
	if err != nil {
		t.Fatal(err)
	}
	// Junk of the same size: reading it again would count nothing or fail.
	inst.put(name, bytes.Repeat([]byte("x"), int(info.Size())), 1)
	wantPlay(t, l, inst, 100)
}

func TestPlayLedgerCountsAReusedNameThatIsAnotherLog(t *testing.T) {
	inst := newPlayInstance(t)
	l := NewPlayLedger(t.TempDir())
	name := "2026-10-27-1.log.gz"
	inst.archive(name, playedFor(100), 1)
	wantPlay(t, l, inst, 100)
	// The storage page's delete frees the name for a run that day.
	inst.remove(name)
	inst.archive(name, playedFor(200)+logPlain(start+900), 2)
	wantPlay(t, l, inst, 300)
}

func TestPlayLedgerMeasuresTheLatestAfreshAndCountsItOnceArchived(t *testing.T) {
	inst := newPlayInstance(t)
	l := NewPlayLedger(t.TempDir())
	text := logBoot(start) + logConnect(start) + logJoined(start) + logPlain(start+60)
	inst.put(latestLogName, []byte(text), 1)
	wantPlay(t, l, inst, 60)
	text += logLeft(start + 90)
	inst.put(latestLogName, []byte(text), 2)
	wantPlay(t, l, inst, 90)
	inst.remove(latestLogName)
	inst.archive("2026-10-27-1.log.gz", text, 2)
	inst.put(latestLogName, []byte(logBoot(start)), 3)
	wantPlay(t, l, inst, 90)
}

func TestPlayLedgerCarriesASessionOverMidnightByTheFileTimes(t *testing.T) {
	inst := newPlayInstance(t)
	l := NewPlayLedger(t.TempDir())
	// The archive's number is not the order: -2 is the older of the two.
	inst.archive("2026-10-27-2.log.gz", logBoot(start)+logConnect(start)+logJoined(start)+logPlain(start+100), 1)
	inst.archive("2026-10-27-1.log.gz", logPlain(10)+logChat(310, "[System]", "Player is now AFK")+logPlain(320), 2)
	inst.put(latestLogName, []byte(logPlain(0)+logChat(50, "[System]", "Player is no longer AFK")+logLeft(250)), 3)
	wantPlay(t, l, inst, 100+0+200)
	// And again from the ledger, which has the first two.
	wantPlay(t, l, inst, 300)
}

func TestPlayLedgerTakesTheNameFromAnEarlierLog(t *testing.T) {
	inst := newPlayInstance(t)
	inst.archive("2026-10-27-1.log.gz", logBoot(start)+logConnect(start)+logJoined(start)+logPlain(start+100), 1)
	inst.archive("2026-10-28-1.log.gz", logPlain(0)+logChat(1000, "[System]", "Player is now AFK")+logPlain(1500), 2)
	wantPlay(t, NewPlayLedger(t.TempDir()), inst, 100+700)
}

func TestPlayLedgerAsksAgainAboutALogItCouldNotRead(t *testing.T) {
	inst := newPlayInstance(t)
	l := NewPlayLedger(t.TempDir())
	name := "2026-10-27-1.log.gz"
	whole := gzipped(t, playedFor(100))
	inst.put(name, whole[:len(whole)/2], 1)
	wantPlay(t, l, inst, 0)
	inst.put(name, whole, 1)
	wantPlay(t, l, inst, 100)
}

func TestPlayLedgerKeepsChaptersApart(t *testing.T) {
	data := t.TempDir()
	one, two := newPlayInstance(t), newPlayInstance(t)
	one.archive("2026-10-27-1.log.gz", playedFor(100), 1)
	two.archive("2026-10-27-1.log.gz", playedFor(100), 1)
	l := NewPlayLedger(data)
	wantPlay(t, l, one, 100)
	if got, err := l.Count("lichdenstein", two.dir); err != nil || got != 100 {
		t.Fatalf("%v %d", err, got)
	}
}

func TestPlayLedgerWithNoGameFolderHasWhatItHolds(t *testing.T) {
	l := NewPlayLedger(t.TempDir())
	inst := newPlayInstance(t)
	inst.archive("2026-10-27-1.log.gz", playedFor(100), 1)
	wantPlay(t, l, inst, 100)
	if got, err := l.Count("frangfurd", filepath.Join(t.TempDir(), "gone")); err != nil || got != 100 {
		t.Fatalf("%v %d", err, got)
	}
	if got, err := l.Count("luxemburg", ""); err != nil || got != 0 {
		t.Fatalf("%v %d", err, got)
	}
}

func TestPlayLedgerRecountsWhenItsFileIsBroken(t *testing.T) {
	data := t.TempDir()
	inst := newPlayInstance(t)
	inst.archive("2026-10-27-1.log.gz", playedFor(100), 1)
	if err := os.WriteFile(filepath.Join(data, playLedgerFile), []byte("{ broken"), 0o600); err != nil {
		t.Fatal(err)
	}
	wantPlay(t, NewPlayLedger(data), inst, 100)
	raw, err := os.ReadFile(filepath.Join(data, playLedgerFile))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "2026-10-27-1.log.gz:") {
		t.Fatalf("not rewritten: %s", raw)
	}
}

// The ledger holds numbers and file names, never a line or the player's name.
func TestPlayLedgerKeepsNothingOfTheLog(t *testing.T) {
	data := t.TempDir()
	inst := newPlayInstance(t)
	inst.archive("2026-10-27-1.log.gz", playedFor(100)+logChat(start+200, "[System]", "Player is now AFK"), 1)
	wantPlay(t, NewPlayLedger(data), inst, 100)
	raw, err := os.ReadFile(filepath.Join(data, playLedgerFile))
	if err != nil {
		t.Fatal(err)
	}
	for _, leak := range []string{"Player", "play.example", "CHAT"} {
		if strings.Contains(string(raw), leak) {
			t.Fatalf("%q in the ledger: %s", leak, raw)
		}
	}
	if !strings.Contains(string(raw), `"seconds":`+strconv.Itoa(100)) {
		t.Fatalf("no seconds: %s", raw)
	}
}
