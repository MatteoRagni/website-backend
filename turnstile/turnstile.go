// SPDX-License-Identifier: MPL-2.0
// Copyright 2025, Matteo Ragni.
// This Source Code Form is subject to the terms of the Mozilla
// Public License, v. 2.0. If a copy of the MPL was not distributed
// with this file, You can obtain one at https://mozilla.org/MPL/2.0/.

package turnstile

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"

	config "github.com/matteoragni/website-backend/config"
)

func VerifyTurnstile(cfg config.CFTurnConfig, token, ip string) (bool, error) {
	if cfg.Secret == "" || cfg.Endpoint == "" {
		return false, errors.New("turnstile not configured")
	}
	data := "secret=" + cfg.Secret + "&response=" + token + "&remoteip=" + ip
	data = url.QueryEscape(data)
	resp, err := http.Post(cfg.Endpoint, "application/x-www-form-urlencoded", strings.NewReader(data))
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	var res struct {
		Success bool `json:"success"`
	}
	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&res); err != nil {
		return false, err
	}
	return res.Success, nil
}
