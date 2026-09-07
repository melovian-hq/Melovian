// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

// sign produces a detached Ed25519 signature over a file, base64-encoded to
// <file>.sig. The release workflow uses it to sign the checksums manifest;
// update clients verify it against the pinned public key compiled into the
// binary via -X melovian/internal/update.ReleasePublicKey.
//
//	go run ./build/scripts/sign -genkey           # print a new keypair
//	UPDATE_SIGNING_KEY=<b64> go run ./build/scripts/sign dist/checksums.txt
package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"flag"
	"fmt"
	"os"
	"strings"
)

func main() {
	genkey := flag.Bool("genkey", false, "generate a keypair and exit")
	keyEnv := flag.String("key-env", "UPDATE_SIGNING_KEY", "env var holding the base64 private key")
	flag.Parse()

	if *genkey {
		pub, priv, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			fatal(err)
		}
		fmt.Println("public (compile with -X melovian/internal/update.ReleasePublicKey):")
		fmt.Println(base64.StdEncoding.EncodeToString(pub))
		fmt.Println("private (store as the " + *keyEnv + " GitHub secret):")
		fmt.Println(base64.StdEncoding.EncodeToString(priv))
		return
	}

	if flag.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "usage: sign <file> [file...]")
		os.Exit(2)
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(os.Getenv(*keyEnv)))
	if err != nil || len(raw) != ed25519.PrivateKeySize {
		fatal(fmt.Errorf("%s must hold a base64-encoded %d-byte Ed25519 private key", *keyEnv, ed25519.PrivateKeySize))
	}
	priv := ed25519.PrivateKey(raw)
	for _, path := range flag.Args() {
		data, err := os.ReadFile(path)
		if err != nil {
			fatal(err)
		}
		sig := ed25519.Sign(priv, data)
		out := path + ".sig"
		if err := os.WriteFile(out, []byte(base64.StdEncoding.EncodeToString(sig)+"\n"), 0o644); err != nil {
			fatal(err)
		}
		fmt.Println("signed", path, "->", out)
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "sign:", err)
	os.Exit(1)
}
