// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package appconfig

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func LoadDotEnv(path string) (bool, error) {
	file, err := os.Open(path) //#nosec G304 -- caller supplies explicit env file path
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("open %s: %w", path, err)
	}
	defer func() { _ = file.Close() }()

	scanner := bufio.NewScanner(file)
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			return true, fmt.Errorf("%s:%d: invalid line (expected KEY=VALUE)", path, lineNo)
		}
		key = strings.TrimSpace(key)
		if key == "" {
			return true, fmt.Errorf("%s:%d: empty key", path, lineNo)
		}
		value = strings.TrimSpace(value)
		if unquoted, ok := unquoteDotEnvValue(value); ok {
			value = unquoted
		}
		if os.Getenv(key) == "" {
			if err := os.Setenv(key, value); err != nil {
				return true, fmt.Errorf("set %s: %w", key, err)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return true, fmt.Errorf("read %s: %w", path, err)
	}
	return true, nil
}

func unquoteDotEnvValue(raw string) (string, bool) {
	if len(raw) < 2 {
		return raw, false
	}
	quote := raw[0]
	if quote != '"' && quote != '\'' {
		return raw, false
	}
	if raw[len(raw)-1] != quote {
		return raw, false
	}
	return raw[1 : len(raw)-1], true
}
