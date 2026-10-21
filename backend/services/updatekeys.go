package services

import (
	"crypto/ed25519"
	"encoding/base64"
	"log/slog"
)

// UpdatePublicKeys are the ed25519 keys a release's checksums.txt.sig must
// verify against before the launcher installs an update (selfupdate.go), each
// base64 on one line. The private half of each is the release environment's
// UPDATE_SIGNING_KEY secret and nowhere else; `go run ./scripts/update-key`
// makes a pair. A second key may be listed so a lost one can be retired
// without players reinstalling by hand. With none, the launcher offers the
// download site only.
var UpdatePublicKeys = []string{
	// Made 2026-10-21; the private half is the release environment's secret.
	"d6ZQ4S6RsRQwDFXjt5GhLj3x6H+7SLvLueqU++Qz5zI=",
}

// updateKeys parses UpdatePublicKeys, leaving out any that is malformed.
func updateKeys() []ed25519.PublicKey {
	keys := make([]ed25519.PublicKey, 0, len(UpdatePublicKeys))
	for _, s := range UpdatePublicKeys {
		raw, err := base64.StdEncoding.DecodeString(s)
		if err != nil || len(raw) != ed25519.PublicKeySize {
			slog.Error("update key is malformed", "key", s)
			continue
		}
		keys = append(keys, ed25519.PublicKey(raw))
	}
	return keys
}
