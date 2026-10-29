# tsaiclient

[![PkgGoDev](https://pkg.go.dev/badge/mod/github.com/thorsphere/tsaiclient)](https://pkg.go.dev/mod/github.com/thorsphere/tsaiclient)
![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/thorsphere/tsaiclient)
![GitHub release (latest by date)](https://img.shields.io/github/v/release/thorsphere/tsaiclient)
![GitHub Top Language](https://img.shields.io/github/languages/top/thorsphere/tsaiclient)
[![CodeFactor](https://www.codefactor.io/repository/github/thorsphere/tsaiclient/badge)](https://www.codefactor.io/repository/github/thorsphere/tsaiclient)
![OSS Lifecycle](https://img.shields.io/osslifecycle/thorsphere/tsaiclient)

---

Go package for unified AI provider integration. Lightweight client to easily abstract and connect services to AI providers.

`tsaiclient` gives you a single, provider-agnostic chat-completion call.
Configure a model, pick a provider, and get back a normalized result with
token usage and cost accounting. All inputs are validated up front, every
call is deadline-bounded, and responses are size-capped, so the package is
safe to expose to untrusted input.

---

## Features

- **Provider-agnostic API**: one `Client.Complete` call regardless of backend;
  providers plug in through the small `Provider` interface.
- **OpenRouter provider included**: full support for the OpenRouter Chat
  Completions API, including app attribution headers
  (`HTTP-Referer`, `X-OpenRouter-Title`, `X-OpenRouter-Categories`,
  `X-OpenRouter-App-Visibility`).
- **Privacy mode**: if enabled, routes exclusively to Zero Data Retention (ZDR)
  endpoints and forbids upstream data collection/training
  (`data_collection: deny`, `zdr: true`).
- **Normalized results**: `Result` reports raw content, truncation status,
  and `Usage` with prompt/completion/reasoning/total tokens.
- **Hardened by default**: per-call deadlines, response
  cap, strict HTTPS URL and hostname validation, and error values that redact
  credentials, query strings, and fragments.
- **Actionable errors**: provider failures come back with recovery guidance
  (e.g. "rate limited; retry later", "check that the API key is set correctly").
- **Prompt-cache affinity**: optional sticky session IDs keep related calls on
  the same upstream routing key.
- **Minimal footprint**: pure standard library plus
  [`tserr`](https://github.com/thorsphere/tserr) for error handling.


---

## Getting Started

Requires Go 1.27 or later.

```sh
go get github.com/thorsphere/tsaiclient
```

Set your provider API key in the environment (twelve-factor style):

```sh
export OPENROUTER_API_KEY="..."
```

---

## Example

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    "github.com/thorsphere/tsaiclient"
)

func main() {
    // Configure the model and generation defaults.
    cfg, err := tsaiclient.NewConfig("google/gemini-3.5-flash-lite")
    if err != nil {
        log.Fatal(err)
    }
    cfg.WithMaxCompletionTokens(512).
        WithTemperature(0.7).
        WithReasoningEffort("medium").
        WithTimeout(30 * time.Second)

    // Configure the provider with optional app attribution.
    provider := tsaiclient.NewOpenRouter().
        WithReferer("https://example.com").
        WithTitle("Example App")

    // Optional: enable strict privacy (ZDR-only routing, no data collection).
    if err := provider.EnablePrivacy(); err != nil {
        log.Fatal(err)
    }

    // Construct the client.
    client, err := tsaiclient.NewClient(cfg, provider)
    if err != nil {
        log.Fatal(err)
    }

    // Build a request; a random session ID is generated automatically.
    req, err := tsaiclient.NewRequest("Explain twelve-factor apps in one paragraph.")
    if err != nil {
        log.Fatal(err)
    }
    req.WithSystemPrompt("You are a concise assistant.")

    // Send the completion request.
    result, err := client.Complete(context.Background(), req)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println(result.Content)
    fmt.Printf("tokens=%d cost=$%.4f truncated=%v\n",
        result.Usage.TotalTokens, result.Usage.Cost, result.Truncated)
}
```

---

## API reference

### Configuration — `Config`

Created with `NewConfig(model)` and customized through chainable `With*`
builders:

| Builder | Description | Constraint |
|---|---|---|
| `WithModel(model)` | Model identifier passed to the provider | Non-empty |
| `WithMaxCompletionTokens(n)` | Cap on generated tokens per call | ≥ 0; `0` means no cap |
| `WithTemperature(t)` | Sampling temperature | `[0.0, 2.0]` |
| `WithReasoningEffort(e)` | Reasoning effort level | `none`, `minimal`, `low`, `medium`, `high`, `xhigh`; empty omits the field |
| `WithTimeout(d)` | Per-call deadline | ≥ 0; `0` falls back to the 60s default |

### Requests — `Request`

Created with `NewRequest(prompt)`, which generates a cryptographically random
session ID. Use `WithSystemPrompt` to add a system instruction and
`WithSessionID`/`NewSessionID` to control sticky routing for prompt-cache
affinity (session IDs are ≤ 256 bytes of printable non-whitespace ASCII).

### Results — `Result` and `Usage`

`Complete` returns a `Result`:

- `Content` — the raw text returned by the model.
- `Truncated` — `true` when generation hit `MaxCompletionTokens`
  (`finish_reason == "length"`).
- `Usage` — `PromptTokens`, `CompletionTokens`, `ReasoningTokens`,
  `TotalTokens`, and `Cost` (USD; `0` when the provider reports none).

### Providers — `Provider` interface

Implement `Provider` to add a backend:

```go
type Provider interface {
    Name() string
    EnablePrivacy() error
    Validate() error
    BuildHTTPRequest(ctx context.Context, cfg *Config, req *Request) (*http.Request, error)
    ParseHTTPResponse(statusCode int, body []byte) (Result, error)
}
```

Built in: `OpenRouter` (`NewOpenRouter()`), with attribution options
`WithReferer` (HTTPS URL, required when any other attribution field is set),
`WithTitle`, `WithCategories` (≤ 2 lowercase-hyphen slugs, ≤ 30 chars each),
and `WithHidden`.

### Error handling

All errors are returned as wrapped [`tserr`](https://github.com/thorsphere/tserr)
values. Provider failures carry actionable guidance (rate limits, auth,
insufficient credits, timeouts, transient outages), and mid-generation
provider errors embedded in `200 OK` bodies are detected and surfaced.

---

## Hardening notes

- `Complete` attempts each call **exactly once**; retrying transient failures
  (429, 5xx, timeouts) is the caller's responsibility.
- The `ctx` deadline bounds the whole call; the per-call timeout
  (default 60s) is layered on top and cancels in-flight requests.
- Response bodies are capped at 1 MiB; oversized bodies are rejected with a
  `TooLarge` error before being returned to the caller.
- The API key is read from `OPENROUTER_API_KEY` at request time and must remain
  server-side.

---

## Known Limitations

- **OpenRouter is the only built-in provider.** Other backends require a custom
  `Provider` implementation.
- **No streaming.** Responses are returned whole after generation completes.
- **First completion choice only.** When a provider returns multiple choices,
  only the first is used; later choices cannot replace an empty or failed first
  choice.
- **No automatic retries.** Transient failure handling is left to the caller so
  retry policy stays under application control.
- **Privacy mode has no geographic restrictions.** OpenRouter does not support
  region-locked (e.g. EU-only) routing; privacy mode covers ZDR endpoints and
  data-collection denial only.
- **Non-streaming reasoning models** may consume `MaxCompletionTokens` on
  reasoning before emitting content; this is reported as an error rather than
  an empty result.

---

## Documentation & Resources

- [Go Package Documentation](https://pkg.go.dev/github.com/thorsphere/tsaiclient): Complete API reference
- [Open Source Insights](https://deps.dev/go/github.com%2Fthorsphere%2Ftsaiclient): Dependency analysis

---

## ⚖️ License & Commercial Usage

Copyright (c) 2026 thorsphere. All rights reserved.

This project is licensed under the **Functional Source License v1.1 (FSL-1.1-ALv2)**. 

* The use, modification, and redistribution of this Go package is completely free for private, educational, non-commercial, and internal purposes. 
* If you are a company or institution looking to use this package in a commercial product, service, or business environment, you must secure a commercial license.
* Each version of this software automatically converts to the fully open-source Apache License, Version 2.0 on the second anniversary of its release.

For full details, please see the [LICENSE](LICENSE.md) file.

### 💼 Commercial Licensing & Inquiries

To purchase a commercial license or discuss support options, please reach out directly:

* 📩 **Contact:** business at thorsphere dot com
* 💬 **Response Time:** Usually within a couple of business days.

*Please include your company name and a brief overview of your use case so I can provide the right licensing details.*
