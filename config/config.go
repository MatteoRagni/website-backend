// SPDX-License-Identifier: MPL-2.0
// Copyright 2025, Matteo Ragni.
// This Source Code Form is subject to the terms of the Mozilla
// Public License, v. 2.0. If a copy of the MPL was not distributed
// with this file, You can obtain one at https://mozilla.org/MPL/2.0/.

package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
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
	var cfturn_sec_env_override = false

	b, err := os.ReadFile(path)
	if err != nil {
		return c, err
	}
	if err := json.Unmarshal(b, &c); err != nil {
		return c, err
	}
	// set defaults body size if needed
	if c.Options.MaxBodySize == 0 {
		c.Options.MaxBodySize = 4096
	}
	// Override CF Turnstile secret via an environment variable if set
	if s, ok := os.LookupEnv("WB_CF_TURN_SECRET"); ok {
		cfturn_sec_env_override = true
		c.CFTurn.Secret = s
	}
	// Check that the secret is 35 characters long. We so not care for
	// runes, in this case the assumption is that the secret is always alphanumeric.
	if c.CFTurn.Secret != "" && len(c.CFTurn.Secret) != 35 {
		return c, errors.New("cfTurnstile.secret must be 35 characters long")
	}
	// Setting default CF Turnstile endpoint
	if c.CFTurn.Endpoint == "" {
		c.CFTurn.Endpoint = "https://challenges.cloudflare.com/turnstile/v0/siteverify"
	}
	if _, err := url.Parse(c.CFTurn.Endpoint); err != nil {
		return c, errors.New("cfTurnstile.endpoint must be a valid URL starting with http:// or https://")
	}

	// Some info output for turnstile configuration, before logging is setup, running in console
	if c.CFTurn.Secret == "" {
		fmt.Println("Turnstile secret is not set. Turnstile verification will be disabled.")
	} else {
		cfturn_secret := c.CFTurn.Secret[:5] + "*******..."
		if cfturn_sec_env_override {
			cfturn_secret += " (from env)"
		}
		fmt.Printf("Turnstile endpoint %s with secret %s\n", c.CFTurn.Endpoint, cfturn_secret)
	}

	// Setting default CTA handler path
	if c.Options.CTAEndpoint == "" {
		c.Options.CTAEndpoint = "/-/cta"
	}
	return c, nil
}
