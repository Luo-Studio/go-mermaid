package fonts

import "testing"

func TestBytesNonEmpty(t *testing.T) {
	for _, st := range []Style{StyleRegular, StyleBold, StyleItalic} {
		b, err := Bytes(st)
		if err != nil {
			t.Fatalf("Bytes(%v): %v", st, err)
		}
		if len(b) < 1000 {
			t.Fatalf("Bytes(%v) too small: %d bytes (expected ~300 KB TTF)", st, len(b))
		}
		magic := uint32(b[0])<<24 | uint32(b[1])<<16 | uint32(b[2])<<8 | uint32(b[3])
		if magic != 0x00010000 && string(b[0:4]) != "OTTO" && string(b[0:4]) != "true" {
			t.Fatalf("Bytes(%v) does not look like a TrueType font: magic=%08x", st, magic)
		}
	}
}

func TestUnknownStyle(t *testing.T) {
	if _, err := Bytes(Style(99)); err == nil {
		t.Fatal("expected error for unknown style")
	}
}
