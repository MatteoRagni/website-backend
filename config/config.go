// SPDX-License-Identifier: MPL-2.0
// Copyright 2025, Matteo Ragni.
// This Source Code Form is subject to the terms of the Mozilla
// Public License, v. 2.0. If a copy of the MPL was not distributed
// with this file, You can obtain one at https://mozilla.org/MPL/2.0/.

package config

import (
	"encoding/json"
	"os"
)

type Config struct {
	Locations map[string]SiteConfig `json:"locations"`
	CFTurn    CFTurnConfig          `json:"cfTurnstile"`
	SMTP      SMTPConfig            `json:"smtp"`
	Log       LogConfig             `json:"log"`
	Options   OptionsConfig         `json:"options"`
}

type SiteConfig struct {
	Dir      string   `json:"dir"`
	Type     string   `json:"type"`     // "static" or "spa"
	Basepage string   `json:"basepage"` // Valid only for "spa" type
	Paths    []string `json:"paths"`    // Valid only for "spa" type
}

type CFTurnConfig struct {
	Endpoint string `json:"endpoint"`
	Secret   string `json:"secret"`
}

type SMTPConfig struct {
	Server     string `json:"server"`
	Port       int    `json:"port"`
	Encryption string `json:"encryption"`
	Username   string `json:"username"`
	Password   string `json:"password"`
	VerifyTLS  bool   `json:"verifyTls"`
	From       string `json:"fromAddress"`
	To         string `json:"toAddress"`
}

type LogConfig struct {
	Destination string `json:"destination"`
	MinLevel    string `json:"minLevel"`
}

type OptionsConfig struct {
	EnableRateLimiting bool   `json:"enableRateLimiting"`
	MaxBodySize        int64  `json:"maxBodySize"`
	BlockBotUserAgents bool   `json:"blockBotUserAgents"`
	CTAEndpoint        string `json:"ctaEndpoint"`
}

func LoadConfig(path string) (Config, error) {
	var c Config
	b, err := os.ReadFile(path)
	if err != nil {
		return c, err
	}
	if err := json.Unmarshal(b, &c); err != nil {
		return c, err
	}
	// set defaults if needed
	if c.Options.MaxBodySize == 0 {
		c.Options.MaxBodySize = 4096
	}
	return c, nil
}
