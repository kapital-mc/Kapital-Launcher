package services

import (
	"fmt"
	"strings"
	"testing"
)

// The lines as the game writes them, at a time of day in seconds.
func stamp(at int) string {
	return fmt.Sprintf("[28Oct2026 %02d:%02d:%02d.027]", at/3600%24, at/60%60, at%60)
}

func logBoot(at int) string {
	return stamp(at) + " [main/INFO] [cpw.mods.modlauncher.Launcher/MODLAUNCHER]: ModLauncher running: args [--username, Player, --version, 1.21.1]\n"
}

func logRender(at int, logger, msg string) string {
	return stamp(at) + " [Render thread/INFO] [" + logger + "/]: " + msg + "\n"
}

func logConnect(at int) string {
	return logRender(at, "net.minecraft.client.gui.screens.ConnectScreen", "Connecting to play.example, 25565")
}

func logJoined(at int) string {
	return logRender(at, "net.minecraft.advancements.AdvancementTree", "Loaded 2654 advancements")
}

func logLeft(at int) string {
	return logRender(at, "DistantHorizons-ClientApi", "Client on ClientOnly mode disconnecting.")
}

func logStopping(at int) string {
	return logRender(at, "net.minecraft.client.Minecraft", "Stopping!")
}

func logChat(at int, tag, body string) string {
	return logRender(at, "net.minecraft.client.gui.components.ChatComponent", tag+" [CHAT] "+body)
}

func logServer(at int, logger, msg string) string {
	return stamp(at) + " [Server thread/INFO] [" + logger + "/]: " + msg + "\n"
}

func logPlain(at int) string { return logRender(at, "some.mod", "ticking") }

const start = 10 * 3600

func TestMeasurePlayCountsTheTimeInAWorld(t *testing.T) {
	const integrated = "net.minecraft.client.server.IntegratedServer"
	cases := map[string]struct {
		lines string
		want  int64
	}{
		"a server, joined and left": {
			logBoot(start) + logConnect(start+100) + logJoined(start+110) + logLeft(start+1910) + logStopping(start+2000),
			1800,
		},
		"menu time on either side counts for nothing": {
			logBoot(start) + logPlain(start+900) + logStopping(start+1800),
			0,
		},
		"a connection that failed": {
			logBoot(start) + logConnect(start+100) + logStopping(start+900),
			0,
		},
		"advancements with no connection": {
			logBoot(start) + logJoined(start+100) + logStopping(start+900),
			0,
		},
		"another mod connecting": {
			logBoot(start) + logRender(start+10, "Essential", "Connecting to Essential Connection Manager...") + logJoined(start+20) + logStopping(start+900),
			0,
		},
		"a connector thread writes the connect": {
			logBoot(start) + stamp(start+100) + " [Server Connector #1/INFO] [ConnectScreen/]: Connecting to play.example, 25565\n" + logJoined(start+110) + logLeft(start+710),
			600,
		},
		"a second join with no line for leaving the first": {
			logBoot(start) + logConnect(start) + logJoined(start) + logConnect(start+600) + logJoined(start+610) + logStopping(start+1210),
			1200,
		},
		"kicked": {
			logBoot(start) + logConnect(start) + logJoined(start) + logRender(start+300, "ClientCommonPacketListenerImpl", "Client disconnected with reason: Server closed") + logPlain(start+900),
			300,
		},
		"the game closed in the world": {
			logBoot(start) + logConnect(start) + logJoined(start) + logStopping(start+420),
			420,
		},
		"the log simply ends": {
			logBoot(start) + logConnect(start) + logJoined(start) + logPlain(start+77),
			77,
		},
		"singleplayer": {
			logBoot(start) + logServer(start+60, integrated, "Starting integrated minecraft server version 1.21.1") + logServer(start+660, "net.minecraft.server.MinecraftServer", "Stopping server") + logPlain(start+900),
			600,
		},
		"a replay": {
			logBoot(start) + logServer(start+60, integrated, "Starting integrated minecraft server version 1.21.1") + logServer(start+70, "AbstractDhServerLevel", "Started DhClientServerLevel for Wrapped{ServerLevel[Replay]@minecraft:overworld}") + logServer(start+660, "net.minecraft.server.MinecraftServer", "Stopping server"),
			0,
		},
		"a stack trace between lines": {
			logBoot(start) + logConnect(start) + logJoined(start) + "java.lang.RuntimeException: x\n\tat a.b.C(C.java:1)\n[not a time] nope\n" + logLeft(start+60),
			60,
		},
		"past midnight in one log": {
			logBoot(start) + logConnect(86000) + logJoined(86000) + logLeft(86400+100),
			500,
		},
		"the clock put back an hour": {
			logBoot(start) + logConnect(start) + logJoined(start) + logPlain(start+600) + logPlain(start-3000) + logLeft(start+900),
			900,
		},
		"a date in another language": {
			strings.ReplaceAll(logBoot(start)+logConnect(start)+logJoined(start)+logLeft(start+30), "28Oct2026", "28Okt2026"),
			30,
		},
		"Fabric's lines, with CRLF": {
			"[00:14:49] [main/INFO]: Loading Minecraft 1.20.6 with Fabric Loader 0.19.5\r\n" +
				"[00:15:00] [Server Connector #1/INFO]: Connecting to play.example, 25565\r\n" +
				"[00:15:05] [Render thread/INFO]: Loaded 1200 advancements\r\n" +
				"[00:20:05] [Render thread/INFO]: Stopping!\r\n",
			300,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got, err := measurePlay(strings.NewReader(tc.lines), playRun{}, nil)
			if err != nil {
				t.Fatal(err)
			}
			if got.Seconds != tc.want {
				t.Fatalf("seconds = %d, want %d", got.Seconds, tc.want)
			}
		})
	}
}

func TestMeasurePlayLeavesOutAFK(t *testing.T) {
	head := logBoot(start) + logConnect(start) + logJoined(start)
	cases := map[string]struct {
		lines string
		want  int64
	}{
		"the announced stretch and the idle minutes before it": {
			head + logChat(start+1800, "[System]", "Player is now AFK") + logChat(start+3000, "[System]", "Player is no longer AFK") + logLeft(start+3600),
			2100,
		},
		"the idle minutes do not reach before the join": {
			head + logChat(start+120, "[System]", "Player is now AFK") + logChat(start+180, "[System]", "Player is no longer AFK") + logLeft(start+300),
			120,
		},
		"the idle minutes do not reach into the last AFK": {
			head + logChat(start+1000, "[System]", "Player is now AFK") + logChat(start+1100, "[System]", "Player is no longer AFK") +
				logChat(start+1200, "[System]", "Player is now AFK") + logChat(start+1300, "[System]", "Player is no longer AFK") + logLeft(start+1600),
			1000,
		},
		"left while AFK": {
			head + logChat(start+1000, "[System]", "Player is now AFK") + logLeft(start+1500),
			700,
		},
		"a tag before the name": {
			head + logChat(start+1000, "[System]", "[Server] Player is now AFK") + logChat(start+1100, "[System]", "[Server] [A] Player is no longer AFK") + logLeft(start+1600),
			1200,
		},
		"another player": {
			head + logChat(start+1000, "[System]", "Someone is now AFK") + logLeft(start+1600),
			1600,
		},
		"a name that ends the same": {
			head + logChat(start+1000, "[System]", "NotPlayer is now AFK") + logLeft(start+1600),
			1600,
		},
		"someone typing the words": {
			head + logChat(start+1000, "[System]", "[A] Someone: Player is now AFK") + logChat(start+1100, "[Not Secure]", "Player is now AFK") + logLeft(start+1600),
			1600,
		},
		"announced outside a world": {
			logBoot(start) + logChat(start+1000, "[System]", "Player is now AFK") + logConnect(start+1100) + logJoined(start+1100) + logLeft(start+1600),
			500,
		},
		"said twice": {
			head + logChat(start+1000, "[System]", "Player is now AFK") + logChat(start+1100, "[System]", "Player is now AFK") + logChat(start+1200, "[System]", "Player is no longer AFK") + logLeft(start+1600),
			1100,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got, err := measurePlay(strings.NewReader(tc.lines), playRun{}, nil)
			if err != nil {
				t.Fatal(err)
			}
			if got.Seconds != tc.want {
				t.Fatalf("seconds = %d, want %d", got.Seconds, tc.want)
			}
		})
	}
}

// The game starts a new log at midnight while it runs: seen on 7 of 74 logs of
// the author's Frangfurd instance.
func TestMeasurePlayCarriesOverALogBegunAtMidnight(t *testing.T) {
	asked := 0
	lookup := func() string { asked++; return "Player" }
	cases := map[string]struct {
		lines   string
		before  playRun
		want    int64
		inWorld bool
		away    bool
	}{
		"still in the world": {
			logPlain(10) + logLeft(610), playRun{InWorld: true}, 600, false, false,
		},
		"was not in a world": {
			logPlain(10) + logLeft(610), playRun{}, 0, false, false,
		},
		"a new game does not carry on": {
			logBoot(10) + logPlain(610), playRun{InWorld: true}, 0, false, false,
		},
		"still AFK until they come back": {
			logPlain(10) + logChat(310, "[System]", "Player is no longer AFK") + logLeft(610), playRun{InWorld: true, Away: true}, 300, false, false,
		},
		"going AFK needs the name from before": {
			logPlain(0) + logChat(1000, "[System]", "Player is now AFK") + logPlain(1200), playRun{InWorld: true}, 700, true, true,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got, err := measurePlay(strings.NewReader(tc.lines), tc.before, lookup)
			if err != nil {
				t.Fatal(err)
			}
			if got.Seconds != tc.want || got.InWorld != tc.inWorld || got.Away != tc.away {
				t.Fatalf("got %+v, want %d in world %v away %v", got, tc.want, tc.inWorld, tc.away)
			}
		})
	}
	if asked == 0 {
		t.Fatal("the name was never asked for")
	}
}

func TestMeasurePlayTakesNoMarkerFromChat(t *testing.T) {
	head := logBoot(start) + logConnect(start) + logJoined(start)
	for _, said := range []string{
		"<Someone> Stopping!",
		"<Someone> Client on ClientOnly mode disconnecting",
		"<Someone> Client disconnected with reason: x",
		"<Someone> Starting integrated minecraft server",
		"<Someone> Discarding replay payload",
		"Stopping!",
	} {
		t.Run(said, func(t *testing.T) {
			lines := head + logChat(start+100, "[System]", said) + logChat(start+200, "[Not Secure]", said) + logLeft(start+1000)
			got, err := measurePlay(strings.NewReader(lines), playRun{}, nil)
			if err != nil || got.Seconds != 1000 {
				t.Fatalf("%v %+v", err, got)
			}
		})
	}
}

func TestMeasurePlayAsksForTheNameOnlyWhenNeeded(t *testing.T) {
	asked := 0
	lookup := func() string { asked++; return "" }
	lines := logPlain(0) + logChat(100, "[System]", "Player is now AFK") + logChat(200, "[System]", "Player is now AFK") + logPlain(600)
	got, err := measurePlay(strings.NewReader(lines), playRun{InWorld: true}, lookup)
	if err != nil {
		t.Fatal(err)
	}
	if asked != 1 || got.Seconds != 600 {
		t.Fatalf("asked %d times, seconds %d", asked, got.Seconds)
	}
}

func TestMeasurePlaySkipsALineLongerThanItsBuffer(t *testing.T) {
	huge := stamp(start+1) + " [Render thread/INFO] [x/]: " + strings.Repeat("y", 3*playLogLineBytes) + "\n"
	lines := logBoot(start) + logConnect(start) + logJoined(start) + huge + logLeft(start+60)
	got, err := measurePlay(strings.NewReader(lines), playRun{}, nil)
	if err != nil || got.Seconds != 60 {
		t.Fatalf("%v %+v", err, got)
	}
}

// latest.log is being written while it is read.
func TestMeasurePlayIgnoresAHalfWrittenLastLine(t *testing.T) {
	lines := logBoot(start) + logConnect(start) + logJoined(start) + logPlain(start+60) + stamp(start + 900)[:12]
	got, err := measurePlay(strings.NewReader(lines), playRun{}, nil)
	if err != nil || got.Seconds != 60 || !got.InWorld {
		t.Fatalf("%v %+v", err, got)
	}
}

func TestParseLogClock(t *testing.T) {
	for in, want := range map[string]int64{"00:00:00": 0, "01:51:49.027": 6709, "23:59:59": 86399} {
		if got, ok := parseLogClock(in); !ok || got != want {
			t.Fatalf("%q = %d %v, want %d", in, got, ok, want)
		}
	}
	for _, bad := range []string{"", "1:2:3", "aa:bb:cc", "01-51-49", "not a time"} {
		if _, ok := parseLogClock(bad); ok {
			t.Fatalf("%q parsed", bad)
		}
	}
}
