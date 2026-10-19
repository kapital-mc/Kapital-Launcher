package services

import (
	"strings"
	"testing"

	"kapital/backend/models"
)

func withTailscale() models.Chapter {
	ts := addr("Tailscale", "ts.example")
	ts.Tailscale = true
	return models.Chapter{ID: "frangfurd", Server: server(addr("Global", "global.example"), ts)}
}

func TestTailscaleAddressIsOnlyChosenWithTheSwitchOn(t *testing.T) {
	c := withTailscale()
	chosen := map[string]string{"frangfurd": "Tailscale"}
	if got := ServerAddress(c, models.AppSettings{ServerChoices: chosen}); got != "global.example" {
		t.Errorf("switch off: %q", got)
	}
	on := models.AppSettings{ServerChoices: chosen, TailscaleServers: []string{"frangfurd"}}
	if got := ServerAddress(c, on); got != "ts.example" {
		t.Errorf("switch on: %q", got)
	}
	if got := ServerAddress(c, models.AppSettings{TailscaleServers: []string{"frangfurd"}}); got != "global.example" {
		t.Errorf("switch on, nothing chosen: %q", got)
	}
}

func TestValidateServerHoldsTheTailscaleRules(t *testing.T) {
	ts := addr("Tailscale", "ts.example")
	ts.Tailscale = true
	if err := validateServer("c", withTailscale().Server); err != nil {
		t.Errorf("a valid server: %v", err)
	}
	if err := validateServer("c", server(ts, addr("Global", "global.example"))); err == nil || !strings.Contains(err.Error(), "first") {
		t.Errorf("Tailscale first: %v", err)
	}
	other := addr("Other", "other.example")
	other.Tailscale = true
	if err := validateServer("c", server(addr("Global", "global.example"), ts, other)); err == nil || !strings.Contains(err.Error(), "more than one") {
		t.Errorf("two Tailscale addresses: %v", err)
	}
}

func TestTailscaleSwitchesAreChaptersWithATailscaleAddress(t *testing.T) {
	chapters := []models.Chapter{withTailscale(), twoAddresses(), {ID: "luxemburg"}}
	chapters[1].ID = "lichdenstein"
	if err := ValidateTailscaleServers(chapters, []string{"frangfurd"}); err != nil {
		t.Errorf("a chapter with one: %v", err)
	}
	for _, id := range []string{"lichdenstein", "luxemburg", "nope"} {
		if err := ValidateTailscaleServers(chapters, []string{id}); err == nil {
			t.Errorf("%s: accepted", id)
		}
	}
	if got := PruneTailscaleServers(chapters, []string{"nope", "frangfurd", "luxemburg"}); len(got) != 1 || got[0] != "frangfurd" {
		t.Errorf("pruned: %v", got)
	}
	if a, ok := TailscaleAddress(chapters[0]); !ok || a.Address != "ts.example" {
		t.Errorf("TailscaleAddress: %v %v", a, ok)
	}
	if _, ok := TailscaleAddress(chapters[2]); ok {
		t.Error("a chapter with no server has a Tailscale address")
	}
}

func TestDropTailscaleChoicesWhenTheSwitchIsOff(t *testing.T) {
	chapters := []models.Chapter{withTailscale()}
	off := models.AppSettings{ServerChoices: map[string]string{"frangfurd": "Tailscale"}}
	if got := DropTailscaleChoices(chapters, off); got != nil {
		t.Errorf("switch off: %v", got)
	}
	on := off
	on.TailscaleServers = []string{"frangfurd"}
	if got := DropTailscaleChoices(chapters, on); got["frangfurd"] != "Tailscale" {
		t.Errorf("switch on: %v", got)
	}
	other := models.AppSettings{ServerChoices: map[string]string{"frangfurd": "Global"}}
	if got := DropTailscaleChoices(chapters, other); got["frangfurd"] != "Global" {
		t.Errorf("another choice: %v", got)
	}
}

func TestValidateServerHoldsTheInviteToTailscalesShape(t *testing.T) {
	good := "https://login.tailscale.com/admin/invite/ALNcG9hKCCNaCrtn659n11"
	c := withTailscale()
	c.Server.Addresses[1].Invite = good
	if err := validateServer("c", c.Server); err != nil {
		t.Errorf("a valid invite: %v", err)
	}
	for _, bad := range []string{
		"http://login.tailscale.com/admin/invite/ALNcG9hKCCNaCrtn659n11",
		"https://login.tailscale.com.evil.example/admin/invite/ALNcG9hKCCNaCrtn659n11",
		good + "?next=https://evil.example",
		good + "\nhttps://evil.example",
		"https://evil.example/" + good,
	} {
		c := withTailscale()
		c.Server.Addresses[1].Invite = bad
		if err := validateServer("c", c.Server); err == nil {
			t.Errorf("%q: accepted", bad)
		}
	}
	public := withTailscale()
	public.Server.Addresses[0].Invite = good
	if err := validateServer("c", public.Server); err == nil {
		t.Error("an invite on the public address: accepted")
	}
}
