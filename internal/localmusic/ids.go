// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package localmusic

import (
	"encoding/binary"
	"encoding/hex"
	"strings"

	"github.com/cespare/xxhash/v2"
)

const (
	ArtistIDPrefix = "art_"
	AlbumIDPrefix  = "alb_"
)

func NormalizeName(value string) string {
	return strings.TrimSpace(strings.ToLower(value))
}

func ArtistName(raw string) string {
	name := strings.TrimSpace(raw)
	if name == "" {
		return "Unknown Artist"
	}
	return name
}

func AlbumName(raw string) string {
	name := strings.TrimSpace(raw)
	if name == "" {
		return "Unknown Album"
	}
	return name
}

func ArtistID(name string) string {
	return hashID(ArtistIDPrefix, NormalizeName(ArtistName(name)))
}

func ArtistIDNormalized(sortName string) string {
	return hashID(ArtistIDPrefix, sortName)
}

func AlbumID(artist, album string) string {
	key := NormalizeName(ArtistName(artist)) + "\x00" + NormalizeName(AlbumName(album))
	return hashID(AlbumIDPrefix, key)
}

func AlbumIDNormalized(sortArtist, sortAlbum string) string {
	h := xxhash.New()
	_, _ = h.WriteString(sortArtist)
	_, _ = h.Write([]byte{0})
	_, _ = h.WriteString(sortAlbum)
	return hashIDSum(AlbumIDPrefix, h.Sum64())
}

func hashID(prefix, key string) string {
	return hashIDSum(prefix, xxhash.Sum64String(key))
}

func hashIDSum(prefix string, sum uint64) string {
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], sum)
	return prefix + hex.EncodeToString(buf[:])
}
