# UI tests (Playwright)

End-to-end tests for the vmprober web dashboard. They drive a real browser
against a running vmprober and assert on the live dashboard (stats grid, job
cards, protocol distribution, filters, search) plus the JSON/health endpoints.

No VictoriaMetrics stack is needed: `vmprober.config.yaml` runs **pull-only** with
two TCP targets — one always up (`127.0.0.1:8429`, vmprober's own port) and one
always down (`127.0.0.1:9`) — which populates every dashboard widget.

## Run

```bash
# 1. Build the binary (from repo root)
make build

# 2. Install deps + browser (first time)
cd tests/ui
npm ci
npx playwright install chromium      # add --with-deps on Linux

# 3. Run
npx playwright test                  # Linux: uses ../../bin/vmprober
```

Playwright starts vmprober itself (see `webServer` in `playwright.config.ts`) and
waits for `/health` before running. Locally, if vmprober is already running on
:8429 it is reused.

On **Windows**, Playwright's `webServer` runs via `cmd.exe`, which can't launch the
forward-slash path — start vmprober yourself and let Playwright reuse it:

```bash
# from tests/ui/, in one shell:
../../bin/vmprober.exe --config vmprober.config.yaml
# in another shell:
npx playwright test
```

Reports: `npx playwright show-report` (HTML report is uploaded as a CI artifact
on every run).

## CI

`.github/workflows/ui.yml` builds the binary, installs chromium, and runs this
suite on any change to `internal/server/**`, `cmd/vmprober/**`, or `tests/ui/**`.
