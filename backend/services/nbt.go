package services

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

// A minimal reader and writer for Minecraft's NBT, enough to change
// servers.dat: an uncompressed, big-endian tree of named tags. Every tag type
// is read and written back as it was, so a file the game or another tool
// wrote round-trips with only the entries the launcher means to change.

const (
	nbtEnd byte = iota
	nbtByte
	nbtShort
	nbtInt
	nbtLong
	nbtFloat
	nbtDouble
	nbtByteArray
	nbtString
	nbtList
	nbtCompound
	nbtIntArray
	nbtLongArray
)

const (
	maxNBTDepth = 64
	maxNBTItems = 1 << 20 // per array or list; servers.dat holds a few dozen entries
)

// nbtTag is one tag's value. A compound keeps its children in file order.
type nbtTag struct {
	Type     byte
	Value    any        // the scalar or array value
	ListType byte       // the element type of a list
	List     []nbtTag   // a list's elements
	Fields   []nbtField // a compound's children
}

type nbtField struct {
	Name string
	Tag  nbtTag
}

// field returns a compound's child by name.
func (t *nbtTag) field(name string) (*nbtTag, bool) {
	for i := range t.Fields {
		if t.Fields[i].Name == name {
			return &t.Fields[i].Tag, true
		}
	}
	return nil, false
}

// stringField returns a compound's string child, "" when there is none.
func (t *nbtTag) stringField(name string) string {
	f, ok := t.field(name)
	if !ok || f.Type != nbtString {
		return ""
	}
	if s, ok := f.Value.(string); ok {
		return s
	}
	return ""
}

func nbtText(s string) nbtTag { return nbtTag{Type: nbtString, Value: s} }

// readNBT reads a whole file: one named root tag, a compound.
func readNBT(data []byte) (string, nbtTag, error) {
	r := bufio.NewReader(bytes.NewReader(data))
	typ, err := r.ReadByte()
	if err != nil {
		return "", nbtTag{}, fmt.Errorf("nbt: %w", err)
	}
	if typ != nbtCompound {
		return "", nbtTag{}, fmt.Errorf("nbt: root is type %d, not a compound", typ)
	}
	name, err := readNBTString(r)
	if err != nil {
		return "", nbtTag{}, err
	}
	root, err := readNBTPayload(r, typ, 0)
	if err != nil {
		return "", nbtTag{}, err
	}
	return name, root, nil
}

func readNBTString(r io.Reader) (string, error) {
	var n uint16
	if err := binary.Read(r, binary.BigEndian, &n); err != nil {
		return "", fmt.Errorf("nbt: string length: %w", err)
	}
	b := make([]byte, n)
	if _, err := io.ReadFull(r, b); err != nil {
		return "", fmt.Errorf("nbt: string: %w", err)
	}
	return string(b), nil
}

func readNBTLength(r io.Reader) (int, error) {
	var n int32
	if err := binary.Read(r, binary.BigEndian, &n); err != nil {
		return 0, fmt.Errorf("nbt: length: %w", err)
	}
	if n < 0 || n > maxNBTItems {
		return 0, fmt.Errorf("nbt: length %d out of range", n)
	}
	return int(n), nil
}

func readNBTPayload(r *bufio.Reader, typ byte, depth int) (nbtTag, error) {
	if depth > maxNBTDepth {
		return nbtTag{}, errors.New("nbt: nested too deep")
	}
	t := nbtTag{Type: typ}
	var err error
	switch typ {
	case nbtByte:
		var v int8
		err = binary.Read(r, binary.BigEndian, &v)
		t.Value = v
	case nbtShort:
		var v int16
		err = binary.Read(r, binary.BigEndian, &v)
		t.Value = v
	case nbtInt:
		var v int32
		err = binary.Read(r, binary.BigEndian, &v)
		t.Value = v
	case nbtLong:
		var v int64
		err = binary.Read(r, binary.BigEndian, &v)
		t.Value = v
	case nbtFloat:
		var v float32
		err = binary.Read(r, binary.BigEndian, &v)
		t.Value = v
	case nbtDouble:
		var v float64
		err = binary.Read(r, binary.BigEndian, &v)
		t.Value = v
	case nbtString:
		t.Value, err = readNBTString(r)
	case nbtByteArray, nbtIntArray, nbtLongArray:
		t.Value, err = readNBTArray(r, typ)
	case nbtList:
		return readNBTList(r, depth)
	case nbtCompound:
		return readNBTCompound(r, depth)
	default:
		return nbtTag{}, fmt.Errorf("nbt: unknown tag type %d", typ)
	}
	if err != nil {
		return nbtTag{}, fmt.Errorf("nbt: %w", err)
	}
	return t, nil
}

func readNBTArray(r io.Reader, typ byte) (any, error) {
	n, err := readNBTLength(r)
	if err != nil {
		return nil, err
	}
	switch typ {
	case nbtByteArray:
		v := make([]int8, n)
		return v, binary.Read(r, binary.BigEndian, v)
	case nbtIntArray:
		v := make([]int32, n)
		return v, binary.Read(r, binary.BigEndian, v)
	default:
		v := make([]int64, n)
		return v, binary.Read(r, binary.BigEndian, v)
	}
}

func readNBTList(r *bufio.Reader, depth int) (nbtTag, error) {
	elem, err := r.ReadByte()
	if err != nil {
		return nbtTag{}, fmt.Errorf("nbt: list type: %w", err)
	}
	n, err := readNBTLength(r)
	if err != nil {
		return nbtTag{}, err
	}
	t := nbtTag{Type: nbtList, ListType: elem}
	for range n {
		item, err := readNBTPayload(r, elem, depth+1)
		if err != nil {
			return nbtTag{}, err
		}
		t.List = append(t.List, item)
	}
	return t, nil
}

func readNBTCompound(r *bufio.Reader, depth int) (nbtTag, error) {
	t := nbtTag{Type: nbtCompound}
	for {
		typ, err := r.ReadByte()
		if err != nil {
			return nbtTag{}, fmt.Errorf("nbt: compound: %w", err)
		}
		if typ == nbtEnd {
			return t, nil
		}
		if len(t.Fields) >= maxNBTItems {
			return nbtTag{}, errors.New("nbt: compound too large")
		}
		name, err := readNBTString(r)
		if err != nil {
			return nbtTag{}, err
		}
		child, err := readNBTPayload(r, typ, depth+1)
		if err != nil {
			return nbtTag{}, err
		}
		t.Fields = append(t.Fields, nbtField{Name: name, Tag: child})
	}
}

// writeNBT writes a whole file: the named root compound.
func writeNBT(name string, root nbtTag) ([]byte, error) {
	var b bytes.Buffer
	b.WriteByte(nbtCompound)
	if err := writeNBTString(&b, name); err != nil {
		return nil, err
	}
	if err := writeNBTPayload(&b, root); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func writeNBTString(b *bytes.Buffer, s string) error {
	if len(s) > 0xFFFF {
		return errors.New("nbt: string too long")
	}
	b.Write([]byte{byte(len(s) >> 8), byte(len(s))})
	b.WriteString(s)
	return nil
}

func writeNBTPayload(b *bytes.Buffer, t nbtTag) error {
	switch t.Type {
	case nbtString:
		s, ok := t.Value.(string)
		if !ok {
			return fmt.Errorf("nbt: not a string: %T", t.Value)
		}
		return writeNBTString(b, s)
	case nbtByteArray, nbtIntArray, nbtLongArray:
		return writeNBTArray(b, t.Value)
	case nbtList:
		b.WriteByte(t.ListType)
		if err := binary.Write(b, binary.BigEndian, int32(len(t.List))); err != nil {
			return err
		}
		for _, item := range t.List {
			if err := writeNBTPayload(b, item); err != nil {
				return err
			}
		}
		return nil
	case nbtCompound:
		for _, f := range t.Fields {
			b.WriteByte(f.Tag.Type)
			if err := writeNBTString(b, f.Name); err != nil {
				return err
			}
			if err := writeNBTPayload(b, f.Tag); err != nil {
				return err
			}
		}
		b.WriteByte(nbtEnd)
		return nil
	default:
		return binary.Write(b, binary.BigEndian, t.Value)
	}
}

func writeNBTArray(b *bytes.Buffer, v any) error {
	var n int
	switch a := v.(type) {
	case []int8:
		n = len(a)
	case []int32:
		n = len(a)
	case []int64:
		n = len(a)
	default:
		return fmt.Errorf("nbt: not an array: %T", v)
	}
	if err := binary.Write(b, binary.BigEndian, int32(n)); err != nil {
		return err
	}
	return binary.Write(b, binary.BigEndian, v)
}
