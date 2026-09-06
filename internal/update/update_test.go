// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package update

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestCompare(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"1.0.0", "1.0.0", 0},
		{"v1.0.0", "1.0.0", 0},
		{"1.0.0", "1.0.1", -1},
		{"1.2.0", "1.10.0", -1},
		{"2.0.0", "1.9.9", 1},
		{"1.0.0-rc.1", "1.0.0", -1},
		{"1.0.0-beta", "1.0.0-rc.1", -1},
		{"1.0.0-rc.1", "1.0.0-rc.2", -1},
		{"abc1234", "v1.0.0", -1},
		{"1.0.0+build.5", "1.0.0", 0},
	}
	for _, c := range cases {
		if got := Compare(c.a, c.b); got != c.want {
			t.Errorf("Compare(%q, %q) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}

func TestIsNewer(t *testing.T) {
	if !IsNewer("1.0.0", "1.0.1") {
		t.Error("expected 1.0.1 newer than 1.0.0")
	}
	if IsNewer("1.0.1", "1.0.0") {
		t.Error("expected 1.0.0 not newer")
	}
	if !IsNewer("abc1234-dirty", "1.0.0") {
		t.Error("dev builds should always update to a tagged release")
	}
}

const testFeed = `<?xml version="1.0" encoding="UTF-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <entry>
    <id>tag:github.com,2008:Repository/1</id>
    <updated>2026-09-01T00:00:00Z</updated>
    <link rel="alternate" type="text/html" href="https://github.com/melovian-hq/Melovian/releases/tag/v0.2.0-rc.1"/>
    <title>v0.2.0-rc.1</title>
  </entry>
  <entry>
    <id>tag:github.com,2008:Repository/2</id>
    <updated>2026-08-15T00:00:00Z</updated>
    <link rel="alternate" type="text/html" href="https://github.com/melovian-hq/Melovian/releases/tag/v0.1.0"/>
    <title>Melovian v0.1.0</title>
  </entry>
  <entry>
    <id>tag:github.com,2008:Repository/3</id>
    <updated>2026-08-01T00:00:00Z</updated>
    <link rel="alternate" type="text/html" href="https://github.com/melovian-hq/Melovian/releases/tag/v0.0.9"/>
    <title>v0.0.9</title>
  </entry>
</feed>`

func serveFeed(t *testing.T, body string) (*httptest.Server, string) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/atom+xml")
		fmt.Fprint(w, body)
	}))
	t.Cleanup(srv.Close)
	old := FeedURL
	FeedURL = srv.URL
	t.Cleanup(func() { FeedURL = old })
	return srv, srv.URL
}

func TestLatestStableSkipsPrerelease(t *testing.T) {
	serveFeed(t, testFeed)
	res, err := Check(context.Background(), "0.0.9", Options{})
	if err != nil {
		t.Fatal(err)
	}
	if res.UpToDate || res.Latest == nil {
		t.Fatal("expected update available")
	}
	if res.Latest.Version != "0.1.0" {
		t.Fatalf("got %s, want 0.1.0 (stable, not rc)", res.Latest.Version)
	}
	if res.Latest.NotesURL == "" {
		t.Error("expected notes url")
	}
}

func TestPrereleaseChannel(t *testing.T) {
	serveFeed(t, testFeed)
	res, err := Check(context.Background(), "0.1.0", Options{Channel: ChannelPrerelease})
	if err != nil {
		t.Fatal(err)
	}
	if res.UpToDate || res.Latest == nil || res.Latest.Version != "0.2.0-rc.1" {
		t.Fatalf("got %+v, want 0.2.0-rc.1", res.Latest)
	}
}

func TestUpToDate(t *testing.T) {
	serveFeed(t, testFeed)
	res, err := Check(context.Background(), "0.1.0", Options{})
	if err != nil {
		t.Fatal(err)
	}
	if !res.UpToDate {
		t.Fatal("expected up to date")
	}
}

func TestPinnedVersion(t *testing.T) {
	serveFeed(t, testFeed)
	res, err := Check(context.Background(), "0.0.9", Options{TargetVersion: "v0.0.9"})
	if err != nil {
		t.Fatal(err)
	}
	// Pinning to the current version reports no update.
	if !res.UpToDate {
		t.Fatalf("got %+v", res)
	}
	if _, err := Check(context.Background(), "0.0.9", Options{TargetVersion: "9.9.9"}); err == nil {
		t.Fatal("expected error for missing version")
	}
}

func TestParseChecksums(t *testing.T) {
	data := "abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789  melovian-v1.0.0-server-linux-amd64.tar.gz\n" +
		"# comment\n" +
		strings.Repeat("f", 64) + " *melovian-v1.0.0-bin-linux-amd64\n\n"
	sums := ParseChecksums([]byte(data))
	if len(sums) != 2 {
		t.Fatalf("got %d entries", len(sums))
	}
	if sums["melovian-v1.0.0-server-linux-amd64.tar.gz"] != "abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789" {
		t.Error("bad parse")
	}
}

func TestSignatureVerify(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	old := ReleasePublicKey
	ReleasePublicKey = base64.StdEncoding.EncodeToString(pub)
	t.Cleanup(func() { ReleasePublicKey = old })

	msg := []byte("checksums content")
	sig := ed25519.Sign(priv, msg)
	sigB64 := []byte(base64.StdEncoding.EncodeToString(sig))
	if err := VerifySignature(msg, sigB64); err != nil {
		t.Fatal(err)
	}
	if err := VerifySignature([]byte("tampered"), sigB64); err == nil {
		t.Fatal("expected signature failure on tampered data")
	}
	// Bad key fails closed.
	other, _, _ := ed25519.GenerateKey(rand.Reader)
	ReleasePublicKey = base64.StdEncoding.EncodeToString(other)
	if err := VerifySignature(msg, sigB64); err == nil {
		t.Fatal("expected signature failure under wrong key")
	}
}

func TestNoPinnedKeyFailsSignature(t *testing.T) {
	old := ReleasePublicKey
	ReleasePublicKey = ""
	t.Cleanup(func() { ReleasePublicKey = old })
	if err := VerifySignature([]byte("x"), []byte("eQ==")); err == nil {
		t.Fatal("expected failure without pinned key")
	}
}

func TestVerifyDigest(t *testing.T) {
	h := sha256.Sum256([]byte("hello"))
	if err := VerifyDigest(strings.NewReader("hello"), hex.EncodeToString(h[:])); err != nil {
		t.Fatal(err)
	}
	if err := VerifyDigest(strings.NewReader("bye"), hex.EncodeToString(h[:])); err == nil {
		t.Fatal("expected digest mismatch")
	}
}

// roundtrip through the bsdiff40 container format: the header writer and
// the patch applier must agree on layout even when blocks are small.
func TestPatchHeaderRoundTrip(t *testing.T) {
	var buf strings.Builder
	if err := WritePatchHeader(&buf, 10, 20, 30); err != nil {
		t.Fatal(err)
	}
	patch := buf.String()
	if patch[:8] != bsdiffMagic {
		t.Fatal("bad magic")
	}
	if got := offtin([]byte(patch[8:16])); got != 10 {
		t.Fatalf("ctrl len %d", got)
	}
	if got := offtin([]byte(patch[16:24])); got != 20 {
		t.Fatalf("diff len %d", got)
	}
	if got := offtin([]byte(patch[24:32])); got != 30 {
		t.Fatalf("new size %d", got)
	}
}

func TestApplyPatchRejectsGarbage(t *testing.T) {
	if _, err := ApplyPatch([]byte("old"), []byte("not a patch")); err == nil {
		t.Fatal("expected error")
	}
}

func TestOfftinNegative(t *testing.T) {
	var buf [8]byte
	offtout(-42, buf[:])
	if got := offtin(buf[:]); got != -42 {
		t.Fatalf("offtin roundtrip: %d", got)
	}
}

func TestPlatformSuffix(t *testing.T) {
	if got := PlatformSuffix("linux-armv6"); got != "linux-armv6" {
		t.Fatalf("override ignored: %s", got)
	}
	got := PlatformSuffix("")
	if !strings.Contains(got, "-") {
		t.Fatalf("bad suffix %q", got)
	}
}

func TestAssetNaming(t *testing.T) {
	if got := ChecksumAssetName("v1.2.3"); got != "melovian-v1.2.3-checksums.txt" {
		t.Fatal(got)
	}
	if got := SigAssetName("v1.2.3"); got != "melovian-v1.2.3-checksums.txt.sig" {
		t.Fatal(got)
	}
	if got := DeltaAssetName("v1.2.3", "v1.2.2", "linux-amd64"); got != "melovian-v1.2.3-patch-linux-amd64-from-v1.2.2.bspatch" {
		t.Fatal(got)
	}
}

var _ = time.Now
