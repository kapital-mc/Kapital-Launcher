// Makes the key pair that signs launcher updates (ADR-6 amendment,
// 2026-10-21). Run it once, on your own machine:
//
//	go run ./scripts/update-key
//
// The public key goes into backend/services/updatekeys.go, where every build
// carries it. The private key goes into the repository's release environment
// as the secret UPDATE_SIGNING_KEY, and nowhere else: not a file, not a chat.
package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
)

func main() {
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		fmt.Fprintf(os.Stderr, "update key: %v\n", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stdout, "Public key, for backend/services/updatekeys.go (safe to share):\n\n  %s\n\n",
		base64.StdEncoding.EncodeToString(public))
	fmt.Fprintf(os.Stdout, "Private key, for the secret UPDATE_SIGNING_KEY (share it with no one):\n\n  %s\n",
		base64.StdEncoding.EncodeToString(private.Seed()))
}
