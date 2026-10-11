package services

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// A chapter's instance folder that exists under the launcher's name but was
// not set up by it (issue 244, ADR-2's fourteenth amendment) is moved aside
// before Reinstall writes a new one. The folder keeps every file and stays in
// the instances folder, so Prism still lists it and the player can play or
// delete it there; only its name changes.

// stashTries bounds the numbered names tried when a stash of the same day is
// already there.
const stashTries = 99

// stashName is the name a stashed instance gets: the instance id, "stashed" and
// the day of the stash, then -2, -3 and so on for a second stash that day.
func stashName(id string, day time.Time, n int) string {
	name := fmt.Sprintf("%s-stashed-%s", id, day.Format(time.DateOnly))
	if n > 1 {
		name = fmt.Sprintf("%s-%d", name, n)
	}
	return name
}

// StashInstance renames <instancesDir>/<id> to the first free stash name and
// returns that name. Nothing is copied, opened or deleted. It refuses an id
// that is not a plain folder name, a folder that is not there, and a stash
// name that is a folder name Prism could not hold.
func StashInstance(instancesDir, id string, now time.Time) (string, error) {
	if !prismInstanceID.MatchString(id) {
		return "", fmt.Errorf("instance id %q is not a plain folder name", id)
	}
	from := filepath.Join(instancesDir, id)
	info, err := os.Lstat(from)
	if err != nil {
		return "", fmt.Errorf("stash instance %s: %w", id, err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("stash instance %s: not a folder", id)
	}
	for n := 1; n <= stashTries; n++ {
		name := stashName(id, now, n)
		if !prismInstanceID.MatchString(name) {
			return "", fmt.Errorf("stash instance %s: %q is not a plain folder name", id, name)
		}
		to := filepath.Join(instancesDir, name)
		if _, err := os.Lstat(to); err == nil {
			continue
		} else if !errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("stash instance %s: %w", id, err)
		}
		if err := os.Rename(from, to); err != nil {
			return "", fmt.Errorf("stash instance %s: %w", id, err)
		}
		return name, nil
	}
	return "", fmt.Errorf("stash instance %s: %d stashes today already", id, stashTries)
}

// UnstashInstance moves a stash made by StashInstance back to the instance id,
// for an install that failed after the stash. It refuses when a folder of the
// id is there again, so nothing is put over it.
func UnstashInstance(instancesDir, id, stashed string) error {
	if !prismInstanceID.MatchString(id) || !prismInstanceID.MatchString(stashed) {
		return fmt.Errorf("unstash instance %s: not a plain folder name", id)
	}
	to := filepath.Join(instancesDir, id)
	if _, err := os.Lstat(to); err == nil {
		return fmt.Errorf("unstash instance %s: the folder is there again", id)
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("unstash instance %s: %w", id, err)
	}
	if err := os.Rename(filepath.Join(instancesDir, stashed), to); err != nil {
		return fmt.Errorf("unstash instance %s: %w", id, err)
	}
	return nil
}
