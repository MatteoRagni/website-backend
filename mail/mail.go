// SPDX-License-Identifier: MPL-2.0
// Copyright 2025, Matteo Ragni.
// This Source Code Form is subject to the terms of the Mozilla
// Public License, v. 2.0. If a copy of the MPL was not distributed
// with this file, You can obtain one at https://mozilla.org/MPL/2.0/.

package mail

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"net/smtp"
	"regexp"
	"sort"
	"strings"

	config "github.com/matteoragni/website-backend/config"
)

var (
	reHTML = regexp.MustCompile(`<[^>]*>`)
	reURL  = regexp.MustCompile(`https?://\S+`)
	reCtrl = regexp.MustCompile(`[\x00-\x09\x0B\x0C\x0E-\x1F]`)
)

func SendMail(cfg config.SMTPConfig, subject string, payload map[string]interface{}) error {
	addr := fmt.Sprintf("%s:%d", cfg.Server, cfg.Port)
	body := buildEmailBody(payload)

	msg := "From: " + cfg.From + "\r\n" +
		"To: " + cfg.To + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"MIME-Version: 1.0\r\n" +
		"Content-Type: text/html; charset=UTF-8\r\n" +
		"\r\n" + body

	switch strings.ToLower(cfg.Encryption) {
	case "ssl":
		// Verify SSL
		tlsconf := &tls.Config{InsecureSkipVerify: !cfg.VerifyTLS, ServerName: cfg.Server}
		conn, err := tls.Dial("tcp", addr, tlsconf)
		if err != nil {
			return err
		}
		c, err := smtp.NewClient(conn, cfg.Server)
		if err != nil {
			return err
		}
		defer c.Quit()
		if cfg.Username != "" {
			auth := smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.Server)
			if err := c.Auth(auth); err != nil {
				return err
			}
		}
		if err := c.Mail(cfg.From); err != nil {
			return err
		}
		if err := c.Rcpt(cfg.To); err != nil {
			return err
		}
		w, err := c.Data()
		if err != nil {
			return err
		}
		_, err = w.Write([]byte(msg))
		if err != nil {
			return err
		}
		return w.Close()
	case "starttls":
		conn, err := net.Dial("tcp", addr)
		if err != nil {
			return err
		}
		c, err := smtp.NewClient(conn, cfg.Server)
		if err != nil {
			return err
		}
		defer c.Quit()
		tlsconf := &tls.Config{InsecureSkipVerify: !cfg.VerifyTLS, ServerName: cfg.Server}
		if ok, _ := c.Extension("STARTTLS"); ok {
			if err := c.StartTLS(tlsconf); err != nil {
				return err
			}
		}
		if cfg.Username != "" {
			auth := smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.Server)
			if err := c.Auth(auth); err != nil {
				return err
			}
		}
		if err := c.Mail(cfg.From); err != nil {
			return err
		}
		if err := c.Rcpt(cfg.To); err != nil {
			return err
		}
		w, err := c.Data()
		if err != nil {
			return err
		}
		_, err = w.Write([]byte(msg))
		if err != nil {
			return err
		}
		return w.Close()
	default:
		// plain
		if cfg.Username != "" {
			auth := smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.Server)
			return smtp.SendMail(addr, auth, cfg.From, []string{cfg.To}, []byte(msg))
		} else {
			return smtp.SendMail(addr, nil, cfg.From, []string{cfg.To}, []byte(msg))
		}
	}
}

func buildEmailBody(payload map[string]interface{}) string {
	var b strings.Builder
	b.WriteString(formatPayloadAsTable(payload))
	return b.String()
}

func formatPayloadAsTable(payload map[string]interface{}) string {
	var keys []string
	for k := range payload {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var b strings.Builder
	b.WriteString(`<html>
	<h1>New Submission</h1>

  <table width="600" style="border:1px solid #333">
    <thead>
		  <tr><th align="left">Field</th><th align="left">Value</th></tr>
		</thead>
   <tbody>\n`)

	for _, k := range keys {
		v := payload[k]

		// Try to pretty-print as JSON, fall back to fmt if marshal fails.
		j, err := json.MarshalIndent(v, "", "  ")
		var vstr string
		if err != nil {
			vstr = fmt.Sprintf("%v", v)
		} else {
			vstr = string(j)
		}

		// sanitize: remove HTML, URLs and control chars (reuse package regexes)
		vstr = reHTML.ReplaceAllString(vstr, "")
		vstr = reURL.ReplaceAllString(vstr, "")
		vstr = reCtrl.ReplaceAllString(vstr, " ")
		vstr = strings.TrimSpace(vstr)

		// escape pipe characters in keys to keep table valid
		escapedKey := strings.ReplaceAll(k, "<", "&lt;")
		escapedKey = strings.ReplaceAll(escapedKey, ">", "&gt;")

		// write row with fenced code block for the value
		b.WriteString(`<tr>
		  <td><code>` + escapedKey + `</code></td>
			<td><pre>` + vstr + `</pre></td>
		</tr>\n`)
	}
	b.WriteString(`</tbody>
	  </table>
	</html>\n`)

	return b.String()
}
