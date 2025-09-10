## YouTube Botguard (BgUtils) – overview and usage

This project provides optional Botguard attestation support for YouTube InnerTube requests.

- Modes: `off` (default), `auto` (retry on 403; preflight in some flows), `force` (always attest before requests)
- Backends: pure-Go interface with optional JS execution via `goja` behind a build tag
- Caching: in-memory or file-backed
- Debug: optional verbose logs for dev and troubleshooting

### Build

The JS-backed solver requires the `botguard` build tag:

```bash
go build -tags=botguard ./cmd/ytdlp
```

Without this tag, the binary builds successfully but the JS solver constructors are stubs.

### CLI flags

- `--botguard=off|auto|force`: enable and choose strategy (default: `off`)
- `--botguard-script=<path>`: path to a JS file that defines `bgAttest(input)` (requires `-tags=botguard`)
- `--botguard-cache=mem|file`: cache mode (default: `mem`)
- `--botguard-cache-dir=<dir>`: directory for file cache (default: temp dir)
- `--botguard-ttl=<duration>`: default TTL when solver does not return expiry (e.g., `30m`)
- `--debug-botguard`: enable verbose Botguard logs

Example:

```bash
go build -tags=botguard ./cmd/ytdlp
./ytdlp --botguard=auto \
        --botguard-script ./internal/botguard/examples/bg_attest_example.js \
        --botguard-cache=file \
        --botguard-cache-dir ~/.cache/ytdlp/bg \
        --botguard-ttl 30m \
        --debug-botguard \
        https://www.youtube.com/watch?v=dQw4w9WgXcQ
```

### JS solver contract

When built with `-tags=botguard`, the binary can execute a user-provided JS file via `goja`.

The script must export a global function:

```js
function bgAttest(input) { /* ... */ }
```

Where `input` is a plain object with fields like:
- `userAgent`, `pageURL`, `clientName`, `clientVersion`, `visitorID`

Return value options:
- A string token: `return "token-string";`
- An object: `return { token: "token-string", ttlSeconds: 900 };`

See example script at `internal/botguard/examples/bg_attest_example.js`.

### Caching

- In-memory cache keeps tokens per process
- File cache persists tokens across runs (one JSON file per derived key)
- Keys are derived from the Botguard input (UA, client name/version, visitorID)

### Debugging and troubleshooting

- Add `--debug-botguard` to see cache hits/misses and retry steps
- If you get 403 repeatedly, verify:
  - `--botguard` mode is set to `auto` or `force`
  - JS script is valid and exposes `bgAttest`
  - Token is returned and not empty
  - Consider increasing `--botguard-ttl` if the solver omits expiry

### Security and notes

- Do not log sensitive tokens or cookies in production
- This feature is best-effort; YouTube may change Botguard behavior at any time
- The JS solver runs untrusted code – only use scripts you trust



