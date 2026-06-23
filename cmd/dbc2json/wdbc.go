package main

import (
	"encoding/binary"
	"fmt"
	"math"
	"os"
)

// DBC is a minimal reader for the classic WDBC format used by 1.x WoW clients:
// a 20-byte header, then fixed-size records of 4-byte fields, then a string
// block that string fields reference by byte offset.
type DBC struct {
	RecordCount int
	FieldCount  int
	RecordSize  int

	data       []byte
	recordsOff int
	stringsOff int
}

// Open reads and validates a .dbc file.
func Open(path string) (*DBC, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(b) < 20 || string(b[0:4]) != "WDBC" {
		return nil, fmt.Errorf("%s: not a WDBC file", path)
	}
	rc := int(binary.LittleEndian.Uint32(b[4:8]))
	fc := int(binary.LittleEndian.Uint32(b[8:12]))
	rs := int(binary.LittleEndian.Uint32(b[12:16]))
	ss := int(binary.LittleEndian.Uint32(b[16:20]))

	recordsOff := 20
	stringsOff := recordsOff + rc*rs
	if stringsOff+ss > len(b) {
		return nil, fmt.Errorf("%s: truncated (header expects %d bytes, file has %d)", path, stringsOff+ss, len(b))
	}
	return &DBC{
		RecordCount: rc,
		FieldCount:  fc,
		RecordSize:  rs,
		data:        b,
		recordsOff:  recordsOff,
		stringsOff:  stringsOff,
	}, nil
}

func (d *DBC) fieldOffset(rec, field int) int {
	return d.recordsOff + rec*d.RecordSize + field*4
}

// Uint32 returns field as an unsigned 32-bit integer.
func (d *DBC) Uint32(rec, field int) uint32 {
	off := d.fieldOffset(rec, field)
	if off+4 > len(d.data) {
		return 0
	}
	return binary.LittleEndian.Uint32(d.data[off : off+4])
}

// Int32 returns field as a signed 32-bit integer.
func (d *DBC) Int32(rec, field int) int32 { return int32(d.Uint32(rec, field)) }

// Float32 returns field interpreted as an IEEE-754 float.
func (d *DBC) Float32(rec, field int) float32 { return math.Float32frombits(d.Uint32(rec, field)) }

// Str returns the null-terminated string the field points at in the string block.
func (d *DBC) Str(rec, field int) string {
	start := d.stringsOff + int(d.Uint32(rec, field))
	if start < d.stringsOff || start >= len(d.data) {
		return ""
	}
	end := start
	for end < len(d.data) && d.data[end] != 0 {
		end++
	}
	return string(d.data[start:end])
}
