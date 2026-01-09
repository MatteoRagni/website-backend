// SPDX-License-Identifier: MPL-2.0
// Copyright 2025, Matteo Ragni.
// This Source Code Form is subject to the terms of the Mozilla
// Public License, v. 2.0. If a copy of the MPL was not distributed
// with this file, You can obtain one at https://mozilla.org/MPL/2.0/.

package turnstile

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	config "github.com/matteoragni/website-backend/config"
)

func VerifyTurnstile(cfg config.CFTurnConfig, token, ip string) (bool, error) {
	type TurnstileRequest struct {
		Secret   string `json:"secret"`
		Response string `json:"response"`
		RemoteIP string `json:"remoteip"`
	}

	if cfg.Secret == "" || cfg.Endpoint == "" {
		return false, errors.New("turnstile not configured")
	}

	data := TurnstileRequest{cfg.Secret, token, ip}
	jsonData, err := json.Marshal(data)
	if err != nil {
		return false, err
	}
	jsonReader := strings.NewReader(string(jsonData))
	resp, err := http.Post(cfg.Endpoint, "application/json", jsonReader)
	if err != nil {
		return false, err
	}

	// !200 response (400, 500, ...)
	// This should happen only for a configuration error
	if resp.StatusCode > 299 {
		defer resp.Body.Close()
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return false, errors.New("turnstile verification failed with status: " + resp.Status + " (cannot read body)")
		}
		return false, errors.New("turnstile verification failed with status: " + resp.Status + " Error Body: `" + string(bodyBytes) + "`")
	}
	// 200 response
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
