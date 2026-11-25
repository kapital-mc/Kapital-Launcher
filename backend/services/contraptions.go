package services

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"kapital/backend/models"
)

// A chapter's saved contraptions: the Tool Gun's blueprints
// (create_aeronautics_toolgun 0.3.6 keeps each in
// <game>/enxv_aeronautics_structures/<name>.excraft, BlueprintFileRepository)
// and Create's schematics (<game>/schematics/<name>.nbt). The launcher lists
// them with the facts each file carries, and renames, deletes, imports and
// exports them whole. It never writes inside a file. Every write goes through
// an os.Root on the kind's folder and touches only a regular file named by a
// plain file name, so nothing outside that folder is reached.

const (
	toolgunDir    = "enxv_aeronautics_structures"
	schematicsDir = "schematics"
	toolgunExt    = ".excraft"
	schematicExt  = ".nbt"
	// maxContraptionFile is the Tool Gun's own cap on an archive
	// (BlueprintArchiveCodec, 64 MiB); a schematic gets the same.
	maxContraptionFile = 64 << 20
	// maxContraptionName is in characters. The Tool Gun sets none; a file name
	// this long is already past what its list shows.
	maxContraptionName = 64
	// maxContraptions bounds a listing; each file is read for its facts.
	maxContraptions = 500
)

// contraptionKinds are the kinds in the order the page lists them.
var contraptionKinds = []string{models.ContraptionToolgun, models.ContraptionSchematic}

// contraptionFolder is a kind's folder under the game folder and its extension.
func contraptionFolder(kind string) (dir, ext string, err error) {
	switch kind {
	case models.ContraptionToolgun:
		return toolgunDir, toolgunExt, nil
	case models.ContraptionSchematic:
		return schematicsDir, schematicExt, nil
	}
	return "", "", fmt.Errorf("%q is not a kind of contraption", kind)
}

// ContraptionFolder is the folder a kind's files live in, under gameDir.
func ContraptionFolder(gameDir, kind string) (string, error) {
	dir, _, err := contraptionFolder(kind)
	if err != nil {
		return "", err
	}
	return filepath.Join(gameDir, dir), nil
}

// ListContraptions lists both kinds' files in gameDir, newest first. A folder
// that is not there holds nothing; a file whose facts cannot be read is listed
// with Unread set.
func ListContraptions(gameDir string) ([]models.Contraption, error) {
	out := []models.Contraption{}
	for _, kind := range contraptionKinds {
		dir, ext, _ := contraptionFolder(kind)
		entries, err := os.ReadDir(filepath.Join(gameDir, dir))
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		for _, e := range entries {
			if len(out) >= maxContraptions {
				break
			}
			if !e.Type().IsRegular() || !strings.EqualFold(filepath.Ext(e.Name()), ext) {
				continue
			}
			info, err := e.Info()
			if err != nil {
				continue
			}
			out = append(out, contraptionEntry(filepath.Join(gameDir, dir), kind, info))
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Modified > out[j].Modified })
	return out, nil
}

func contraptionEntry(dir, kind string, info fs.FileInfo) models.Contraption {
	c := models.Contraption{
		Kind:       kind,
		File:       info.Name(),
		Name:       strings.TrimSuffix(info.Name(), filepath.Ext(info.Name())),
		Bytes:      info.Size(),
		Modified:   info.ModTime().UnixMilli(),
		Blocks:     -1,
		BlockTypes: -1,
		SubLevels:  -1,
		Materials:  []models.MaterialCount{},
		Items:      []models.MaterialCount{},
	}
	if info.Size() > maxContraptionFile {
		c.Unread = true
		return c
	}
	data, err := os.ReadFile(filepath.Join(dir, info.Name()))
	if err == nil {
		err = readContraptionFacts(kind, data, &c)
	}
	c.Unread = err != nil
	return c
}

// ContraptionName checks a name the player typed and returns it trimmed. The
// rule is the Tool Gun's own (BlueprintFileRepository.normalizeName): letters,
// digits, spaces, '_' and '-', so a renamed file still loads in game.
func ContraptionName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", errors.New("a name is needed")
	}
	if utf8.RuneCountInString(name) > maxContraptionName {
		return "", fmt.Errorf("a name has at most %d characters", maxContraptionName)
	}
	for _, r := range name {
		if !contraptionRune(r) {
			return "", fmt.Errorf("%q is not allowed in a name: letters, digits, spaces, _ and - only", r)
		}
	}
	return name, nil
}

func contraptionRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == ' ' || r == '_' || r == '-'
}

// contraptionFile checks a file name the page passed back: a plain name with
// the kind's extension, nothing that could leave the folder.
func contraptionFile(file, ext string) error {
	if file == "" || file != filepath.Base(file) || strings.ContainsAny(file, "/\\:\x00") ||
		!strings.EqualFold(filepath.Ext(file), ext) || file == ext {
		return fmt.Errorf("%q is not a contraption file", file)
	}
	return nil
}

// openContraptionRoot opens a kind's folder and checks file names a regular
// file in it.
func openContraptionRoot(gameDir, kind, file string) (*os.Root, string, error) {
	dir, ext, err := contraptionFolder(kind)
	if err != nil {
		return nil, "", err
	}
	if err := contraptionFile(file, ext); err != nil {
		return nil, "", err
	}
	root, err := os.OpenRoot(filepath.Join(gameDir, dir))
	if err != nil {
		return nil, "", err
	}
	info, err := root.Lstat(file)
	if err != nil {
		root.Close() //nolint:errcheck // read only, nothing to flush
		return nil, "", err
	}
	if !info.Mode().IsRegular() {
		root.Close() //nolint:errcheck // read only, nothing to flush
		return nil, "", fmt.Errorf("%s is not a regular file", file)
	}
	return root, ext, nil
}

// RenameContraption gives a file a new name (without extension) and returns
// its new file name. A name another file already has is refused; a change of
// case alone is allowed.
func RenameContraption(gameDir, kind, file, name string) (string, error) {
	name, err := ContraptionName(name)
	if err != nil {
		return "", err
	}
	root, ext, err := openContraptionRoot(gameDir, kind, file)
	if err != nil {
		return "", err
	}
	defer root.Close() //nolint:errcheck // a rename has nothing to flush
	target := name + ext
	if target == file {
		return file, nil
	}
	if _, err := root.Lstat(target); err == nil && !strings.EqualFold(target, file) {
		return "", fmt.Errorf("%s already exists", name)
	}
	if err := root.Rename(file, target); err != nil {
		return "", err
	}
	return target, nil
}

// DeleteContraption removes one file.
func DeleteContraption(gameDir, kind, file string) error {
	root, _, err := openContraptionRoot(gameDir, kind, file)
	if err != nil {
		return err
	}
	defer root.Close() //nolint:errcheck // a removal has nothing to flush
	return root.Remove(file)
}

// ExportContraption copies one file to dest, a path the player picked in the
// save dialog.
func ExportContraption(gameDir, kind, file, dest string) error {
	root, _, err := openContraptionRoot(gameDir, kind, file)
	if err != nil {
		return err
	}
	defer root.Close() //nolint:errcheck // read only, nothing to flush
	src, err := root.Open(file)
	if err != nil {
		return err
	}
	defer src.Close() //nolint:errcheck // read only, nothing to flush
	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, io.LimitReader(src, maxContraptionFile+1)); err != nil {
		out.Close() //nolint:errcheck // the copy error is the one returned
		return err
	}
	return out.Close()
}

// ImportContraption copies a file the player picked into its kind's folder,
// the kind told by its extension. It is read whole and checked first: a
// gzipped NBT tree of the Tool Gun's format or a schematic's shape, at most
// maxContraptionFile. Its name is cleaned to the Tool Gun's rule and numbered
// when taken, so an import never replaces a file.
func ImportContraption(gameDir, src string) (models.Contraption, error) {
	kind, ext := "", strings.ToLower(filepath.Ext(src))
	switch ext {
	case toolgunExt:
		kind = models.ContraptionToolgun
	case schematicExt:
		kind = models.ContraptionSchematic
	default:
		return models.Contraption{}, fmt.Errorf("%s is not a .excraft or .nbt file", filepath.Base(src))
	}
	info, err := os.Lstat(src)
	if err != nil {
		return models.Contraption{}, err
	}
	if !info.Mode().IsRegular() || info.Size() > maxContraptionFile {
		return models.Contraption{}, fmt.Errorf("%s is not a file of at most 64 MB", filepath.Base(src))
	}
	data, err := os.ReadFile(src)
	if err != nil {
		return models.Contraption{}, err
	}
	probe := models.Contraption{}
	if err := readContraptionFacts(kind, data, &probe); err != nil {
		return models.Contraption{}, fmt.Errorf("%s: %w", filepath.Base(src), err)
	}
	dir, _, _ := contraptionFolder(kind)
	if err := os.MkdirAll(filepath.Join(gameDir, dir), 0o755); err != nil {
		return models.Contraption{}, err
	}
	root, err := os.OpenRoot(filepath.Join(gameDir, dir))
	if err != nil {
		return models.Contraption{}, err
	}
	defer root.Close() //nolint:errcheck // the file is closed on its own below
	file, err := writeNewContraption(root, cleanContraptionName(filepath.Base(src)), ext, data)
	if err != nil {
		return models.Contraption{}, err
	}
	written, err := root.Lstat(file)
	if err != nil {
		return models.Contraption{}, err
	}
	return contraptionEntry(filepath.Join(gameDir, dir), kind, written), nil
}

// writeNewContraption writes data as "<name><ext>", or "<name> 2<ext>" and on
// when the name is taken, never over an existing file.
func writeNewContraption(root *os.Root, name, ext string, data []byte) (string, error) {
	for n := 1; n <= 99; n++ {
		file := name + ext
		if n > 1 {
			file = fmt.Sprintf("%s %d%s", name, n, ext)
		}
		f, err := root.OpenFile(file, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if errors.Is(err, fs.ErrExist) {
			continue
		}
		if err != nil {
			return "", err
		}
		if _, err := f.Write(data); err != nil {
			f.Close()         //nolint:errcheck // the write error is the one returned
			root.Remove(file) //nolint:errcheck // best effort: a half file is worse than none
			return "", err
		}
		return file, f.Close()
	}
	return "", fmt.Errorf("%s: too many files of that name", name)
}

// cleanContraptionName turns a picked file's name into one the Tool Gun
// accepts: its extension dropped, other characters removed, cut to length.
func cleanContraptionName(base string) string {
	base = strings.TrimSuffix(base, filepath.Ext(base))
	var b strings.Builder
	n := 0
	for _, r := range base {
		if contraptionRune(r) && n < maxContraptionName-3 {
			b.WriteRune(r)
			n++
		}
	}
	if name := strings.TrimSpace(b.String()); name != "" {
		return name
	}
	return "Imported"
}
