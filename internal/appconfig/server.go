// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package appconfig

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
)

type ServerStartupOptions struct {
	NoAuth bool
}

func (o ServerStartupOptions) NoAuthEffective() bool {
	return o.NoAuth || envTruthy(os.Getenv("MELOVIAN_NO_AUTH"))
}

type ServerAuthResolution struct {
	Generated       bool
	NoAuth          bool
	IgnoredSecret   bool
	GeneratedSecret string
}

func GenerateAuthSecret() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate auth secret: %w", err)
	}
	return hex.EncodeToString(buf), nil
}

func PrepareServerAuth(cfg *Config, opts ServerStartupOptions) (ServerAuthResolution, error) {
	if cfg == nil || !cfg.ServerMode {
		return ServerAuthResolution{}, nil
	}
	if cfg.DemoModeEffective() {
		return ServerAuthResolution{}, nil
	}
	if opts.NoAuthEffective() {
		res := ServerAuthResolution{NoAuth: true}
		if strings.TrimSpace(cfg.AuthSecret) != "" {
			res.IgnoredSecret = true
		}
		cfg.AuthSecret = ""
		return res, nil
	}
	if strings.TrimSpace(cfg.AuthSecret) != "" {
		return ServerAuthResolution{}, nil
	}
	secret, err := GenerateAuthSecret()
	if err != nil {
		return ServerAuthResolution{}, err
	}
	cfg.AuthSecret = secret
	return ServerAuthResolution{Generated: true, GeneratedSecret: secret}, nil
}
