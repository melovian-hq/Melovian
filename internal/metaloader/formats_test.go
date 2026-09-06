// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package metaloader

import "testing"

func TestIsVideoFile(t *testing.T) {
	if !IsVideoFile("clip.MP4") {
		t.Fatal("expected mp4 to be video")
	}
	if !IsVideoFile("clip.webm") {
		t.Fatal("expected webm to be video")
	}
	if !IsVideoFile("clip.m4v") {
		t.Fatal("expected m4v to be video")
	}
	if IsVideoFile("song.mp3") {
		t.Fatal("mp3 should not be video")
	}
	if IsVideoFile("movie.mkv") {
		t.Fatal("mkv should not be scanned in v1")
	}
}

func TestIsMediaFile(t *testing.T) {
	if !IsMediaFile("a.flac") || !IsMediaFile("b.mp4") {
		t.Fatal("expected audio and video media files")
	}
	if IsMediaFile("readme.txt") {
		t.Fatal("txt should not be media")
	}
}

func TestMediaKindFromPath(t *testing.T) {
	if MediaKindFromPath("a.mp4") != "video" {
		t.Fatal("expected video kind")
	}
	if MediaKindFromPath("a.mp3") != "audio" {
		t.Fatal("expected audio kind")
	}
}

func TestIsAudioFile(t *testing.T) {
	for _, name := range []string{
		"track.FLAC", "book.m4b", "mix.MKA", "radio.mp2", "pack.wv",
		"sacd.dsf", "sacd.dsd", "tape.aifc", "lossless.tta", "multi.mogg",
	} {
		if !IsAudioFile(name) {
			t.Fatalf("expected %s to be audio", name)
		}
	}
	if IsAudioFile("notes.txt") {
		t.Fatal("expected txt to be ignored")
	}
}
