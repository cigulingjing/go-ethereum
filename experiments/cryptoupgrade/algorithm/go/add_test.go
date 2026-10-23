package main

import (
	"bytes"
	"testing"
)

func TestAddInt256(t *testing.T) {
	tests := []struct {
		name string
		a, b string
		want string
	}{
		{"small positive", "01", "02", "03"},
		{"zero", "00", "00", "00"},
		{"negatives", "ff", "ff", "fe"},
		{"cancel", "01", "ff", "00"},
		{"carry across bytes", "00ff", "0001", "0100"},
		{"wrap to sign bit", "7f" + repeatHex("ff", 31), "01", "80" + repeatHex("00", 31)},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := Add(int256FromHex(t, test.a), int256FromHex(t, test.b))
			want := int256FromHex(t, test.want)
			if got != want {
				t.Fatalf("Add(%s, %s) = %x, want %x", test.a, test.b, got, want)
			}
		})
	}
}

func int256FromHex(t *testing.T, hex string) Int256 {
	t.Helper()
	raw := make([]byte, len(hex)/2)
	for i := 0; i < len(raw); i++ {
		raw[i] = hexByte(t, hex[i*2:i*2+2])
	}
	var word Int256
	if len(raw) > len(word) {
		t.Fatalf("value does not fit in int256: %s", hex)
	}
	// 短输入按补码符号扩展，和 ABI 解码负整数时的高位填充一致。
	if len(raw) > 0 && raw[0]&0x80 != 0 {
		for i := range word {
			word[i] = 0xff
		}
	}
	copy(word[len(word)-len(raw):], raw)
	return word
}

func hexByte(t *testing.T, pair string) byte {
	t.Helper()
	var value byte
	for _, char := range pair {
		value <<= 4
		switch {
		case char >= '0' && char <= '9':
			value |= byte(char - '0')
		case char >= 'a' && char <= 'f':
			value |= byte(char-'a') + 10
		default:
			t.Fatalf("bad hex %q", pair)
		}
	}
	return value
}

func repeatHex(part string, count int) string {
	return string(bytes.Repeat([]byte(part), count))
}
