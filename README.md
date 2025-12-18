# 🗿 Website Backend

Lightweight Go web server that serves static content and accepts a single validated CTA POST endpoint. Designed for easy deployment and local testing (Mailpit/MailHog). Structured logging, Cloudflare Turnstile verification and simple per‑IP rate limiting. Designed for minimal dependencies.


## 🎱 🏎️Features 

 * Serve multiple static locations (including SPA/Vite-built `index.html` fallback)
 * POST endpoint `/-/cta` (configurable) that accepts JSON payloads and a Turnstile token
 * Cloudflare Turnstile verification before accepting submissions
 * Sanitization of submitted fields (strips HTML, links and control characters)
 * Email delivery via SMTP relay (supports plain/SSL/STARTTLS)
 * Structured logging with configurable destination and level
 * Simple per-IP sliding-window rate limiter and bot User-Agent blocking options
 * TLS Termination on reverse-proxy or WAP

## 🏎️ Quick start

1. Copy the sample configuration and adapt values:

```bash
cp sample/sample.json config.json
# edit config.json (turnstile secret, smtp server, paths...)
```

2. Run locally (development):

```bash
# build
go build -o website-backend

# run with your config file
./website-backend -config config.json -listen :8080
```

3. Test emails locally: run Mailpit or MailHog and point `smtp.server`/`smtp.port` in the config to it.

## 🛠️ Configuration

The server reads all configuration from a JSON file (path provided with `-config`). Minimal example:

```jsonc
{
  "locations": {
    "/": {  // A base pattern for the site
      "type": "spa", // use this type for single page application with multiple 
                     // paths that needs to go on index.html. Example, vite build
      "basepage": "index.html", // Define the file to serve for the substitution paths
      "dir": "spa/dist", // the distribution directory
      "paths" [  // List of route in your SPA to be redirected on basepath
        "about",
        "careers",
        "products/prod-A",
        "products/prod-B",
      ]
    },
    "/assets": { // This is an example for a static site
      "type": "static",
      "dir": "assets/public"
    }
    // Add how many site you desire, but be carefull with name clashing.
  },
  "cfTurnstile": {  // Turnstile configuration
                    // you will need to include the component in your page
                    // see ./sample/home/page/index.html for an example
    "endpoint": "https://challenges.cloudflare.com/turnstile/v0/siteverify",
    "secret": "<your-secret>"
  },
  "smtp": {
    "server": "mail.example",
    "port": 25,
    "encryption": "plain", // enum: "plain", "ssl", "starttls"
    "verifyTls": true, // verify certitificates
    "username": "john.doe",
    "password": "<password>",
    "fromAddress": "john@example.com", // Avoid CTA address in CTA. Use internal one
    "toAddress": "info@example.com" // Address that will receive the mail
  },
  "log": {
    "destination": "logfile.log", // Log file
    "minLevel": "INFO" // Log level (minimum)
  },
  "options": {
    "enableRateLimiting": false, // Enable IP rate limiting
    "maxBodySize": 8192, // Maximum body sizes for requests
    "blockBotUserAgents": false, // Block bot user agent
    "ctaEndpoint": "/-/cta" // Customize CTA endpoint
  }
}
```

Notes:
 * `locations.dir` maps URL prefixes to filesystem directories. Be carefule
 * `locations.type` accepts `spa` or `static`
 * `cfTurnstyle.endpoint` and `secret` are used for server-side Turnstile verification.
 * `smtp.encryption` accepts `plain`, `ssl`, or `starttls`.
 * `log.minLevel` supports standard levels (DEBUG, INFO, WARN, ERROR).

There are sample configurations under `sample/` (for local testing) and `config.sample.json` in repository root.

## 📬 Endpoint: CTA POST

- URL: `POST /-/cta` (can be customized)
- Body: JSON up to configured size (suggested maximum ~8192 bytes). Sample body:

```json
{
  "token": "turnstile-token",
  "payload": {
    "name": "Alice",
    "email": "alice@example.com",
    "message": "Hello"
  }
}
```

Server behavior:
 * Limits content length and wraps the request body in a MaxBytesReader.
 * Refuses obvious fake user agents (configurable) and rate-limits by IP (configurable).
 * Verifies `token` with Cloudflare Turnstile.
 * Sanitizes payload values (removes HTML, links and control characters) and formats them into an email.
 * Sends the sanitized submission to the configured SMTP `toAddress`.
 * Returns a generic error message on failure to avoid giving useful feedback to attackers.
 * On success returns 200 with no content.


## 🔩 Development & testing

 * Use `sample/sample.json` to run locally with Mailpit on port `1025`.
 * Example Mailpit command (Docker):

   ```bash
   docker run --rm -p 1025:1025 -p 8025:8025 axllent/mailpit
   ```
   
 * Then run your server pointing to that SMTP server and submit a test `POST /-/cta`.

## 💬 Logging

Logs are written to the configured destination in structured text format. Adjust `log.minLevel` in your config to filter verbosity.

## 🏎️ Deployment

* Build a static binary: `go build -o website-backend` and deploy the binary plus configuration.
* Recommended to run behind a reverse proxy (nginx/Caddy) for TLS and additional hardening.

Example systemd unit (optional):

```ini
[Unit]
Description=Website Backend

[Service]
ExecStart=/opt/website-backend/website-backend -config /etc/website-backend/config.json -listen :8080
Restart=on-failure
User=www-data

[Install]
WantedBy=multi-user.target
```

## 🔐 Security considerations

* The service intentionally returns generic error messages to avoid leaking internal state.
* Keep the Turnstile `secret` and SMTP credentials out of version control.
* Validate and rotate SMTP credentials and Turnstile secrets as part of normal ops.

## 🤝 Contributing

Contributions welcome. Open issues or PRs for bugfixes and improvements. Please keep changes focused and add tests where appropriate.

## 🗿 License

> SPDX-License-Identifier: MPL-2.0
> Copyright 2025, Matteo Ragni.
> This Source Code Form is subject to the terms of the Mozilla
> Public License, v. 2.0. If a copy of the MPL was not distributed
> with this file, You can obtain one at https://mozilla.org/MPL/2.0/.
