// Builds the Prism instance zips a release attaches (#251): one per chapter
// with a hosted pack, for a player who uses Prism without the launcher. The
// zip is what Install writes for a fresh instance (services.PrismZip), built
// from the same code, so the two cannot drift apart. It reads each chapter's
// hosted pack.toml and downloads the pinned packwiz jars, so it needs network.
//
// Run: go run ./scripts/prism-instances <out dir>
package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"kapital/backend/services"
)

const manifestPath = "data/launcher.json"

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: go run ./scripts/prism-instances <out dir>")
		os.Exit(2)
	}
	if err := run(os.Args[1]); err != nil {
		fmt.Fprintf(os.Stderr, "prism instances: %v\n", err)
		os.Exit(1)
	}
}

func run(outDir string) error {
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return err
	}
	manifest, err := services.ParseManifest(data)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}
	cache, err := os.MkdirTemp("", "prism-instances-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(cache) //nolint:errcheck // a temporary jar cache

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	creator := services.NewInstanceCreator(cache)
	built := 0
	for _, chapter := range manifest.Chapters {
		if chapter.Pack.Packwiz == nil {
			fmt.Printf("%s: no hosted pack, no zip\n", chapter.ID)
			continue
		}
		name, err := services.PrismZipName(chapter)
		if err != nil {
			return err
		}
		zip, err := creator.PrismZip(ctx, chapter)
		if err != nil {
			return fmt.Errorf("%s: %w", chapter.ID, err)
		}
		if err := os.WriteFile(filepath.Join(outDir, name), zip, 0o644); err != nil {
			return err
		}
		fmt.Printf("%s: %s, %d bytes\n", chapter.ID, name, len(zip))
		built++
	}
	// A release with no zip at all is a manifest that lost its packs, not a
	// release without them.
	if built == 0 {
		return fmt.Errorf("no chapter in %s has a hosted pack", manifestPath)
	}
	return nil
}
