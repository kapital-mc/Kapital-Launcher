// Signs a release's checksums.txt for the launcher's own update (ADR-6
// amendment, 2026-10-21), in the release workflow:
//
//	UPDATE_SIGNING_KEY=<base64 seed> go run ./scripts/update-sign release/checksums.txt
//
// It writes the signature, base64 on one line, beside the file as
// checksums.txt.sig. A key whose public half is not in
// backend/services/updatekeys.go fails the release, since no launcher could
// install what it signs.
package main

import (
	"crypto/ed25519"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"

	"kapital/backend/services"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: go run ./scripts/update-sign <checksums.txt>")
		os.Exit(2)
	}
	if err := run(os.Args[1], os.Getenv("UPDATE_SIGNING_KEY")); err != nil {
		fmt.Fprintf(os.Stderr, "update sign: %v\n", err)
		os.Exit(1)
	}
}

func run(path, secret string) error {
	seed, err := base64.StdEncoding.DecodeString(strings.TrimSpace(secret))
	if err != nil || len(seed) != ed25519.SeedSize {
		return errors.New("UPDATE_SIGNING_KEY is missing or is not a base64 ed25519 seed")
	}
	key := ed25519.NewKeyFromSeed(seed)
	public := base64.StdEncoding.EncodeToString(key.Public().(ed25519.PublicKey))
	if !slices.Contains(services.UpdatePublicKeys, public) {
		return fmt.Errorf("the signing key's public half %s is not in backend/services/updatekeys.go", public)
	}
	sums, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	sig := base64.StdEncoding.EncodeToString(ed25519.Sign(key, sums))
	if err := os.WriteFile(path+".sig", []byte(sig+"\n"), 0o644); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "signed %s with %s\n", path, public)
	return nil
}
