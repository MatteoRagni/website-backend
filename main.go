// SPDX-License-Identifier: MPL-2.0
// Copyright 2025, Matteo Ragni.
// This Source Code Form is subject to the terms of the Mozilla
// Public License, v. 2.0. If a copy of the MPL was not distributed
// with this file, You can obtain one at https://mozilla.org/MPL/2.0/.

package main

import (
	"encoding/json"
	"flag"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	config "github.com/matteoragni/website-backend/config"
	mail "github.com/matteoragni/website-backend/mail"
	turnstile "github.com/matteoragni/website-backend/turnstile"
)

// rate limiter
type rateInfo struct {
	timestamps []time.Time
}

var (
	cfg    config.Config            // global configuration
	rlMu   sync.Mutex               // rate limiter mutex
	rlimap = map[string]*rateInfo{} // rate limiter database
)

func main() {
	cfgFile := flag.String("config", "config.json", "path to JSON config")
	addr := flag.String("listen", ":8080", "listen address")
	flag.Parse()

	var err error
	cfg, err = config.LoadConfig(*cfgFile)
	if err != nil {
		_, _ = os.Stderr.WriteString("config load failed: " + err.Error() + "\n")
		os.Exit(1)
	}

	setupLogger(cfg.Log)

	mux := http.NewServeMux()

	// static locations
	for pattern, site := range cfg.Locations {
		switch strings.ToLower(site.Type) {

		case "spa":
			if _, err := os.Stat(site.Dir); os.IsNotExist(err) {
				log.Fatalf("spa directory does not exist for %s: %s", pattern, site.Dir)
			}
			baseFile := filepath.Join(site.Dir, site.Basepage)
			if _, err := os.Stat(baseFile); os.IsNotExist(err) {
				log.Fatalf("spa base file does not exist for %s: %s", pattern, baseFile)
			}
			log.Infof("serving [%s] %s -> %s", site.Type, pattern, site.Dir)
			for _, path := range site.Paths {
				newPattern := pattern + path + "/"
				mux.HandleFunc(newPattern, func(w http.ResponseWriter, r *http.Request) {
					http.ServeFile(w, r, baseFile)
				})
				log.Infof("serving [%s] %s -> %s", site.Type, newPattern, baseFile)
			}

			handle := http.StripPrefix(pattern, http.FileServer(http.Dir(site.Dir)))
			mux.Handle(pattern, handle)
			continue
		case "static":
			if _, err := os.Stat(site.Dir); os.IsNotExist(err) {
				log.Fatalf("static directory does not exist for %s: %s", pattern, site.Dir)
			}
			handle := http.StripPrefix(pattern, http.FileServer(http.Dir(site.Dir)))
			mux.Handle(pattern, handle)
			log.Infof("serving [%s] %s -> %s", site.Type, site.Dir, pattern)
			continue

		default:
			log.Fatalf("invalid site type for %s: %s", pattern, site.Type)
		}
	}

	ctaEndpoint := cfg.Options.CTAEndpoint
	if ctaEndpoint == "" {
		ctaEndpoint = "/-/cta"
	}
	mux.HandleFunc(ctaEndpoint, ctaHandler)

	srv := &http.Server{Addr: *addr, Handler: limitMiddleware(mux)}
	log.Infof("listening %s", *addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}
}

func setupLogger(l config.LogConfig) {
	level, err := log.ParseLevel(strings.ToLower(l.MinLevel))
	if err != nil {
		level = log.InfoLevel
	}
	log.SetLevel(level)
	if l.Destination != "" {
		f, err := os.OpenFile(l.Destination, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err == nil {
			log.SetOutput(f)
		} else {
			log.Warnf("failed open log file, using stderr: %v", err)
		}
	}
	log.SetFormatter(&log.TextFormatter{FullTimestamp: true})
}

// limitMiddleware to add basic protections
func limitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// limit request body size via Content-Length too
		if cfg.Options.MaxBodySize >= 0 && r.ContentLength > cfg.Options.MaxBodySize && r.ContentLength != -1 {
			warnRefuse(r, "content-length too large")
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func ctaHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	// User-Agent check
	ua := r.Header.Get("User-Agent")
	if ua == "" || isFakeUA(ua) {
		warnRefuse(r, "fake user agent")
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	// rate limit by IP
	ip := clientIP(r)
	if !allowRate(ip) {
		warnRefuse(r, "rate limit")
		http.Error(w, "invalid request", http.StatusTooManyRequests)
		return
	}

	// enforce max bytes
	r.Body = http.MaxBytesReader(w, r.Body, cfg.Options.MaxBodySize)
	defer r.Body.Close()

	var body struct {
		Token   string                 `json:"token"`
		Payload map[string]interface{} `json:"payload"`
	}
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&body); err != nil && err != io.EOF {
		warnRefuse(r, "bad json: %v"+err.Error())
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	if body.Token == "" {
		warnRefuse(r, "missing token")
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	ok, err := turnstile.VerifyTurnstile(cfg.CFTurn, body.Token, ip)
	if err != nil || !ok {
		warnRefuse(r, "turnstile failed: "+err.Error())
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	// build sanitized email body
	if err := mail.SendMail(cfg.SMTP, "New CTA Submission", body.Payload); err != nil {
		log.Warnf("send mail failed: %v", err)
		http.Error(w, "invalid request", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func isFakeUA(ua string) bool {
	if !cfg.Options.BlockBotUserAgents {
		return false
	}
	l := strings.ToLower(ua)
	if strings.Contains(l, "curl/") || strings.Contains(l, "python-requests") || strings.Contains(l, "bot") {
		return true
	}
	return false
}

func clientIP(r *http.Request) string {
	// prefer X-Forwarded-For
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func allowRate(ip string) bool {
	if !cfg.Options.EnableRateLimiting {
		return true
	}

	const limit = 5            // TODO: This should be configurable
	const window = time.Minute // TODO: This should be configurable

	rlMu.Lock()
	defer rlMu.Unlock()

	now := time.Now()
	ri, ok := rlimap[ip]
	if !ok {
		ri = &rateInfo{timestamps: []time.Time{now}}
		rlimap[ip] = ri
		return true
	}
	// prune
	t := ri.timestamps
	var nt []time.Time
	for _, ts := range t {
		if now.Sub(ts) <= window {
			nt = append(nt, ts)
		}
	}
	nt = append(nt, now)
	ri.timestamps = nt
	if len(nt) > limit {
		return false
	}
	return true
}

func warnRefuse(r *http.Request, reason string) {
	ip := clientIP(r)
	log.WithFields(log.Fields{"ip": ip, "path": r.URL.Path, "reason": reason}).Warn("refused submit")
}
