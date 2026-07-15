# TraderMade CLI

Command-line access to the [TraderMade](https://tradermade.com) REST and
WebSocket APIs for forex, crypto, metals, and CFDs.

## 1. Install

With Go installed:

```bash
go install github.com/tradermade/go-cli@latest
```

Or download a binary from [Releases](../../releases).

To build from source:

```bash
git clone https://github.com/tradermade/go-cli.git
cd go-cli
go build -o tradermade .
```

Confirm the installation:

```console
$ tradermade version
tradermade 0.1.0-dev
commit abc1234, built 2026-07-08, go1.24.0 windows/amd64
```

## 2. Configure API keys

Create keys from [tradermade.com/signup](https://tradermade.com/signup).
REST commands use a REST key; `live` and `board` use a WebSocket key.

```bash
tradermade config set-key --rest YOUR_REST_KEY
tradermade config set-key --ws YOUR_WEBSOCKET_KEY
```

You can save both together:

```bash
tradermade config set-key --rest YOUR_REST_KEY --ws YOUR_WEBSOCKET_KEY
```

Environment variables override saved keys:

- `TRADERMADE_REST_API_KEY`
- `TRADERMADE_WS_API_KEY`
- `TRADERMADE_API_KEY` as a fallback for both

Check the active configuration and connectivity:

```console
$ tradermade config show
rest    abcd********wxyz  (from config.json (rest_key))
stream  wsab********wxyz  (from config.json (ws_key))
```

## 3. Help

Use the long help flag for the complete endpoint list:

```console
$ tradermade --help
REST API:
  candle      One candle (REST /api/v1/minute_historical or /api/v1/hour_historical)
  convert     Currency conversion (REST /api/v1/convert)
  historical  Daily OHLC (REST /api/v1/historical)
  quote       Live quotes (REST /api/v1/live)
  symbols     Codes (REST /api/v1/live_currencies_list or /api/v1/live_crypto_list)
  timeseries  OHLC candle ranges (REST /api/v1/timeseries)

WebSocket API:
  live        Live ticks (WebSocket wss://stream.tradermade.com/feedAdv)

REST + WebSocket:
  board       Dashboard (WebSocket /feedAdv; REST /historical and /live)
  doctor      Connectivity check (REST /live; WebSocket /feedAdv)
```

Use `tradermade COMMAND --help` to see how that endpoint is constructed,
including arguments, API parameters, valid values, limits, flags, and examples:

```bash
tradermade timeseries --help
tradermade live --help
tradermade historical --help
```

Only the long `--help` form is supported. The `tradermade help` command and
`-h` shorthand are intentionally unavailable. The exhaustive command catalog
is in [cmds.md](cmds.md).

## 4. REST endpoints

Table output is the default. Do not pass `--output table`.

### Live quotes - `/api/v1/live`

```console
$ tradermade quote EURUSD GBPUSD
SYMBOL  BID      ASK      MID
EURUSD  1.14105  1.14113  1.14109
GBPUSD  1.33489  1.33501  1.33495

as of 2026-07-08 10:29:11 UTC
```

The timestamp is the server quote timestamp, not local request time.
Use `--save quote.csv` to save the snapshot as CSV.

### Convert - `/api/v1/convert`

```console
$ tradermade convert 1000 USD INR
1000 USD = 85812.5 INR
rate  1 USD = 85.8125 INR
as of 2026-07-08 10:29:11 UTC
```

Arguments map to the API's `amount`, `from`, and `to` parameters.

### Symbol lists - `/api/v1/live_currencies_list`

```console
$ tradermade symbols
AED  UAE Dirham
ARS  Argentine Peso
AUD  Australian Dollar
...

54 codes
```

Use `--market crypto` to select `/api/v1/live_crypto_list`. The complete list
from the selected endpoint is returned.

### Historical daily OHLC - `/api/v1/historical`

```console
$ tradermade historical EURUSD --date 2026-07-01
SYMBOL  OPEN     HIGH     LOW      CLOSE
EURUSD  1.17263  1.18094  1.16836  1.17951

daily candle for 2026-07-01
```

`--date` accepts `YYYY-MM-DD`, `today`, or `yesterday` (the default).

### Timeseries OHLC - `/api/v1/timeseries`

```console
$ tradermade timeseries EURUSD --start 2026-07-01 --end 2026-07-03
DATE        OPEN     HIGH     LOW      CLOSE
2026-07-01  1.17263  1.18094  1.16836  1.17951
2026-07-02  1.17951  1.18291  1.17542  1.18017
2026-07-03  1.18017  1.18135  1.17268  1.17443

EURUSD daily candles, 3 rows
```

The command supports daily, hourly, and minute intervals. Run
`tradermade timeseries --help` for `start_date`, `end_date`, `interval`, `period`,
`weekend`, relative `--last` ranges, and per-request limits.

### Exact minute/hour candle - `/api/v1/minute_historical`

```console
$ tradermade candle EURUSD --at 2026-07-07-14:30
SYMBOL  TIME              OPEN     HIGH     LOW      CLOSE
EURUSD  2026-07-07-14:30  1.17182  1.17204  1.17161  1.17193
```

Add `--hour` to use `/api/v1/hour_historical` instead. `--at` uses
`YYYY-MM-DD-HH:MM`.

## 5. WebSocket endpoint

### Live streaming - `wss://stream.tradermade.com/feedAdv`

```console
$ tradermade live EURUSD
TIME                    SYMBOL                BID            ASK    BID-VOL    ASK-VOL
20260708-10:29:11.104   EURUSD            1.14105        1.14113      100000      100000
20260708-10:29:11.337   EURUSD            1.14106        1.14114      200000      100000
```

Stop with Ctrl+C. The client logs in with `fmt=JSON`, reconnects with backoff,
and resubscribes automatically. `--send-last` requests a cached `LAST_QUOTE`;
`--ladder` requests depth for plans with trader-ladder access. See
`tradermade live --help` for the exact login and subscription messages.

## 6. Commands using REST and WebSocket

### Board

`board` streams prices over `/feedAdv`, fetches previous closes from
`/historical`, and can compare prices with `/live`.

```console
$ tradermade board EURUSD GBPUSD
tradermade board

 SYMBOL                   BID            ASK       SPREAD       DAY%     LAST
 EURUSD                1.14105        1.14113      0.00008      0.21%       0s
 GBPUSD                1.33489        1.33501      0.00012     -0.08%       0s

 connected | 1 up 1 down | 24 ticks | sort: list (s) | c rest-check | q quit
```

Use `board add`, `board remove`, and `board list` to manage a saved watchlist.
Inside the board: `q` quits, `s` changes sorting, and `c` runs a REST comparison.

### Doctor

```console
$ tradermade doctor
rest-key  ok  abcd********wxyz (from config.json (rest_key))
rest      ok  live quote in 210ms
ws-key    ok  wsab********wxyz (from config.json (ws_key))
stream    ok  login in 180ms - plan allows 20 symbols
config    ok  ~/.config/tradermade/config.json
```

`doctor` checks key resolution, REST connectivity, WebSocket login and plan
capabilities, and config-file validity. It exits nonzero if a check fails.

## Output formats

No output flag means table output. Explicit `--output table` is rejected.

- `--output json`: exact TraderMade response body for REST commands; original
  market tick frames for `live`.
- `--output csv`: CLI-generated CSV with a header row.
- `--output raw`: `live` only; includes greetings and WebSocket control frames.

## Saving CSV files

`quote`, `historical`, `timeseries`, and `live` require `--save` to include a
`.csv` filename:

```bash
tradermade quote UK100 --save quote.csv
tradermade historical EURUSD --save historical.csv
tradermade timeseries EURUSD --last 30d --save "C:\data\market exports\timeseries.csv"
tradermade live EURUSD --save ticks.csv
```

Bare filenames are saved in the current working directory, and the absolute
location is reported. A complete path is also accepted when it includes the
filename, such as `C:\Users\Omkar\Downloads\file.csv`. Directory-only targets
are rejected. REST saves overwrite; live appends and reports its path after the
first saved tick.

## Errors and exit codes

Commands exit `0` on success and nonzero on errors. Common REST failures:

| HTTP | Meaning |
| --- | --- |
| 401 | Invalid key or endpoint not included in the plan |
| 204 | No data for the requested parameters |
| 400 | Invalid symbol, date, or other request parameter |
| 403 | Weekend/closed market or history outside the plan range |

## Development

```bash
make build
make check
make install
```

CI runs formatting, tests, vet, and builds on Linux, Windows, and macOS.

## License

MIT, see [LICENSE](LICENSE).
