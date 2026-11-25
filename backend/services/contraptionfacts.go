package services

import (
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"

	"kapital/backend/models"
)

// The facts a contraption file carries, read from its gzipped NBT. A Tool Gun
// file states its own (format enxv_aeronautics_plot_print_v8/v9: name,
// sublevels, AST_MaterialSummary with total_blocks, unique_block_types and the
// blocks and items as {id, count}); a schematic is the vanilla structure shape
// (size, palette, blocks of {state, pos}) plus Create's sub_levels list, each a
// structure of its own, and its blocks are counted here.

const (
	// maxContraptionTree bounds the decompressed tree; a file beyond it is
	// listed without facts.
	maxContraptionTree = 64 << 20
	toolgunFormat      = "enxv_aeronautics_plot_print_v"
)

// notBlocks are the palette entries a schematic stores that are not blocks.
var notBlocks = map[string]bool{
	"minecraft:air":            true,
	"minecraft:cave_air":       true,
	"minecraft:void_air":       true,
	"minecraft:structure_void": true,
}

func readContraptionFacts(kind string, data []byte, c *models.Contraption) error {
	root, err := readGzipNBT(data)
	if err != nil {
		return err
	}
	if kind == models.ContraptionToolgun {
		return toolgunFacts(root, c)
	}
	return schematicFacts(root, c)
}

func readGzipNBT(data []byte) (nbtTag, error) {
	zr, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nbtTag{}, errors.New("not a gzipped NBT file")
	}
	raw, err := io.ReadAll(io.LimitReader(zr, maxContraptionTree+1))
	if err != nil {
		return nbtTag{}, errors.New("not a gzipped NBT file")
	}
	if len(raw) > maxContraptionTree {
		return nbtTag{}, errors.New("too large to read")
	}
	_, root, err := readNBT(raw)
	return root, err
}

func toolgunFacts(root nbtTag, c *models.Contraption) error {
	if !strings.HasPrefix(root.stringField("format"), toolgunFormat) {
		return errors.New("not a Tool Gun blueprint")
	}
	subs, ok := root.field("sublevels")
	if !ok || subs.Type != nbtList || len(subs.List) == 0 {
		return errors.New("a Tool Gun blueprint with no sub-levels")
	}
	c.SubLevels = len(subs.List)
	summary, ok := root.field("AST_MaterialSummary")
	if !ok || summary.Type != nbtCompound {
		return nil
	}
	c.Blocks = nbtIntField(summary, "total_blocks", -1)
	c.BlockTypes = int(nbtIntField(summary, "unique_block_types", -1))
	c.Materials = countList(summary, "blocks")
	c.Items = countList(summary, "items")
	return nil
}

func schematicFacts(root nbtTag, c *models.Contraption) error {
	size, ok := root.field("size")
	if !ok || size.Type != nbtList || len(size.List) != 3 {
		return errors.New("not a schematic")
	}
	for _, v := range size.List {
		n, ok := v.Value.(int32)
		if !ok {
			return errors.New("not a schematic")
		}
		c.Size = append(c.Size, int(n))
	}
	counts := map[string]int64{}
	if err := countStructure(root, counts); err != nil {
		return err
	}
	c.SubLevels = 0
	if subs, ok := root.field("sub_levels"); ok && subs.Type == nbtList {
		c.SubLevels = len(subs.List)
		for i := range subs.List {
			if err := countStructure(subs.List[i], counts); err != nil {
				return fmt.Errorf("sub-level %d: %w", i+1, err)
			}
		}
	}
	c.Blocks, c.BlockTypes = 0, len(counts)
	c.Materials = make([]models.MaterialCount, 0, len(counts))
	for id, n := range counts {
		c.Blocks += n
		c.Materials = append(c.Materials, models.MaterialCount{ID: id, Count: n})
	}
	sortCounts(c.Materials)
	return nil
}

// countStructure adds one structure's blocks to counts by block id.
func countStructure(s nbtTag, counts map[string]int64) error {
	palette, ok := s.field("palette")
	blocks, ok2 := s.field("blocks")
	if !ok || !ok2 || palette.Type != nbtList || blocks.Type != nbtList {
		return errors.New("a schematic with no palette or blocks")
	}
	for i := range blocks.List {
		state := nbtIntField(&blocks.List[i], "state", -1)
		if state < 0 || state >= int64(len(palette.List)) {
			return errors.New("a block names no palette entry")
		}
		if id := palette.List[state].stringField("Name"); id != "" && !notBlocks[id] {
			counts[id]++
		}
	}
	return nil
}

// nbtIntField reads an integer field of any width, or def.
func nbtIntField(t *nbtTag, name string, def int64) int64 {
	f, ok := t.field(name)
	if !ok {
		return def
	}
	switch v := f.Value.(type) {
	case int8:
		return int64(v)
	case int16:
		return int64(v)
	case int32:
		return int64(v)
	case int64:
		return v
	}
	return def
}

// countList reads a list of {id, count} compounds, most first.
func countList(t *nbtTag, name string) []models.MaterialCount {
	out := []models.MaterialCount{}
	list, ok := t.field(name)
	if !ok || list.Type != nbtList {
		return out
	}
	for i := range list.List {
		id := list.List[i].stringField("id")
		if n := nbtIntField(&list.List[i], "count", 0); id != "" && n > 0 {
			out = append(out, models.MaterialCount{ID: id, Count: n})
		}
	}
	sortCounts(out)
	return out
}

func sortCounts(m []models.MaterialCount) {
	sort.Slice(m, func(i, j int) bool {
		if m[i].Count != m[j].Count {
			return m[i].Count > m[j].Count
		}
		return m[i].ID < m[j].ID
	})
}
