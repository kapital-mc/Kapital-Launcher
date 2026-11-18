package main

import (
	"os"
	"path/filepath"
	"testing"

	"kapital/backend/models"
)

// Play time is read for the installed chapters only: the seconds from the game
// logs (issue 341), not Prism's total, and the last start from instance.cfg
// (issue 192).
func TestGetPlayTimeReadsInstalledChaptersOnly(t *testing.T) {
	app := newTestApp(t)
	root := t.TempDir()
	if err := app.SaveSettings(models.AppSettings{Theme: "dark", PrismRoot: root}); err != nil {
		t.Fatal(err)
	}
	if got, err := app.GetPlayTime(); err != nil || len(got) != 0 {
		t.Fatalf("nothing installed: %v %+v", err, got)
	}
	inst := filepath.Join(root, "instances", "kapital-frangfurd")
	if err := os.MkdirAll(inst, 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := "[General]\nname=Frangfurd\ntotalTimePlayed=229200\nlastLaunchTime=1759660800000\n"
	if err := os.WriteFile(filepath.Join(inst, "instance.cfg"), []byte(cfg), 0o600); err != nil {
		t.Fatal(err)
	}
	logs := filepath.Join(inst, "minecraft", "logs")
	if err := os.MkdirAll(logs, 0o755); err != nil {
		t.Fatal(err)
	}
	latest := "[28Oct2026 10:00:00.000] [main/INFO] [x/]: ModLauncher running\n" +
		"[28Oct2026 10:01:00.000] [Render thread/INFO] [ConnectScreen/]: Connecting to play.example, 25565\n" +
		"[28Oct2026 10:01:10.000] [Render thread/INFO] [AdvancementTree/]: Loaded 2654 advancements\n" +
		"[28Oct2026 10:31:10.000] [Render thread/INFO] [net.minecraft.client.Minecraft/]: Stopping!\n"
	if err := os.WriteFile(filepath.Join(logs, "latest.log"), []byte(latest), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := app.GetPlayTime()
	if err != nil {
		t.Fatal(err)
	}
	want := models.PlayTime{ChapterID: "frangfurd", TotalSeconds: 1800, LastLaunchMs: 1759660800000}
	if len(got) != 1 || got[0] != want {
		t.Fatalf("got %+v, want [%+v]", got, want)
	}
}
