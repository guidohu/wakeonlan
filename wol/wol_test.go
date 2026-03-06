package wol_test

import (
	"bytes"
	"testing"
	"wakeonlan/wol"
)

func TestParseMAC(t *testing.T) {
	tests := []struct {
		name      string
		mac       string
		want      []byte
		wantError bool
	}{
		{
			name:      "Valid Colon Separated",
			mac:       "aa:bb:cc:dd:ee:ff",
			want:      []byte{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff},
			wantError: false,
		},
		{
			name:      "Valid Dash Separated",
			mac:       "00-11-22-33-44-55",
			want:      []byte{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
			wantError: false,
		},
		{
			name:      "Valid Dot Separated",
			mac:       "a1b2.c3d4.e5f6",
			want:      []byte{0xa1, 0xb2, 0xc3, 0xd4, 0xe5, 0xf6},
			wantError: false,
		},
		{
			name:      "Valid Space Separated",
			mac:       "01 23 45 67 89 ab",
			want:      []byte{0x01, 0x23, 0x45, 0x67, 0x89, 0xab},
			wantError: false,
		},
		{
			name:      "Valid No Separators",
			mac:       "aabbccddeeff",
			want:      []byte{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff},
			wantError: false,
		},
		{
			name:      "Valid Mixed Cases",
			mac:       "AA:bb:CC:dd:EE:ff",
			want:      []byte{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff},
			wantError: false,
		},
		{
			name:      "Invalid Length (Too Short)",
			mac:       "aa:bb:cc:dd:ee",
			want:      nil,
			wantError: true,
		},
		{
			name:      "Invalid Length (Too Long)",
			mac:       "aa:bb:cc:dd:ee:ff:00",
			want:      nil,
			wantError: true,
		},
		{
			name:      "Invalid Characters",
			mac:       "xx:yy:zz:ww:qq:pp",
			want:      nil,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := wol.ParseMAC(tt.mac)

			if (err != nil) != tt.wantError {
				t.Fatalf("ParseMAC() error = %v, wantError %v", err, tt.wantError)
			}

			if !tt.wantError && !bytes.Equal(got, tt.want) {
				t.Errorf("ParseMAC() = %x, want %x", got, tt.want)
			}
		})
	}
}
