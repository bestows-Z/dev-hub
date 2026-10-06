package auth

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"net/http"
	"strings"
	"testing"
)

func TestDecodeAvatarReencodesAndRejectsUnsafeInput(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	img.Set(1, 1, color.RGBA{R: 200, G: 70, B: 20, A: 255})
	var jpegBody bytes.Buffer
	if err := jpeg.Encode(&jpegBody, img, nil); err != nil {
		t.Fatal(err)
	}
	encoded, err := decodeAvatar(jpegBody.Bytes())
	if err != nil {
		t.Fatalf("valid JPEG rejected: %v", err)
	}
	if got := http.DetectContentType(encoded); got != "image/png" {
		t.Fatalf("output type = %q", got)
	}
	if _, err := png.Decode(bytes.NewReader(encoded)); err != nil {
		t.Fatalf("PNG output invalid: %v", err)
	}

	if _, err := decodeAvatar([]byte("<svg onload='alert(1)'/>")); err == nil {
		t.Fatal("SVG should be rejected")
	}
	if _, err := decodeAvatar(make([]byte, maxAvatarBytes+1)); err == nil {
		t.Fatal("large upload should be rejected")
	}
	var wideBody bytes.Buffer
	if err := png.Encode(&wideBody, image.NewRGBA(image.Rect(0, 0, 2049, 1))); err != nil {
		t.Fatal(err)
	}
	if _, err := decodeAvatar(wideBody.Bytes()); err == nil {
		t.Fatal("oversized dimensions should be rejected")
	}
}

func TestProfileWebsiteRequiresHttpURLWithoutCredentials(t *testing.T) {
	for _, site := range []string{"", "https://example.com/about", "http://localhost:3000"} {
		if !validWebsite(site) {
			t.Errorf("valid URL rejected: %q", site)
		}
	}
	for _, site := range []string{"javascript:alert(1)", "//example.com", "https://user:pass@example.com", "file:///etc/passwd"} {
		if validWebsite(site) {
			t.Errorf("unsafe URL allowed: %q", site)
		}
	}
}

func TestProfileLengthsCountCharacters(t *testing.T) {
	input := profileInput{Email: "a@example.com", DisplayName: strings.Repeat("中", 60), Bio: strings.Repeat("文", 500)}
	if !validProfileLengths(input) {
		t.Fatal("valid Chinese profile rejected")
	}
	input.DisplayName += "字"
	if validProfileLengths(input) {
		t.Fatal("name over 60 characters accepted")
	}
	input.DisplayName = ""
	input.Bio += "字"
	if validProfileLengths(input) {
		t.Fatal("bio over 500 characters accepted")
	}
}
