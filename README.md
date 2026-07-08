# TraderMade CLI

Command line client for the [TraderMade](https://tradermade.com) market data
API. Live quotes, historical candles, currency conversion, and WebSocket
streaming for forex, metals, crypto and CFDs.

```
$ tradermade quote EURUSD GBPUSD
SYMBOL  BID      ASK      MID
EURUSD  1.14105  1.14113  1.14109
GBPUSD  1.33489  1.33501  1.33495

as of Wed, 08 Jul 2026 10:29:11 GMT
```

## Install

Grab a binary from [Releases](../../releases), or with Go installed:

```bash
go install github.com/tradermade/go-cli@latest
```

To build from source: `git clone`, then `make build` (or plain
`go build -o tradermade .`).

## API keys

You need a key from [tradermade.com/signup](https://tradermade.com/signup).
If one key covers both REST and streaming:

```bash
tradermade config set-key YOUR_API_KEY
```

Some plans have separate REST and WebSocket keys. Save both and each command
picks the right one:

```bash
tradermade config set-key --rest YOUR_REST_KEY
tradermade config set-key --ws   YOUR_WS_KEY
```

If a key starts with a dash, put `--` before it so it isn't parsed as a
flag: `tradermade config set-key --rest -- -abc123`.

Environment variables override saved keys (handy for CI and Docker):
`TRADERMADE_REST_API_KEY`, `TRADERMADE_WS_API_KEY`, or plain
`TRADERMADE_API_KEY` for both.

Run `tradermade doctor` to check the whole setup in one go.

## Output formats

Every data command takes `--output table|json|csv` (`-o` for short).
Table is the default. JSON is stable for scripts and jq. CSV has a header
row and proper quoting, so redirecting to a file gives you something Excel
and pandas open directly:

```bash
tradermade timeseries EURUSD --last 30d --output csv > eurusd.csv
```

For `live`, json means one JSON object per line (NDJSON) and csv means one
row per tick. Both are safe to redirect while the stream runs.

## Commands and endpoints

| Command | Endpoint | Key |
| --- | --- | --- |
| `quote` | `/api/v1/live` | REST |
| `convert` | `/api/v1/convert` | REST |
| `historical` | `/api/v1/historical` | REST |
| `timeseries` | `/api/v1/timeseries` | REST |
| `candle` | `/api/v1/minute_historical`, `/hour_historical` | REST |
| `symbols` | `/api/v1/live_currencies_list`, `/live_crypto_list` | REST |
| `live` | `wss://stream.tradermade.com/feedAdv` | WS |
| `board` | feedAdv, plus `/historical` at startup and `/live` on the c key | both |
| `doctor` | `/live` probe plus a WS login | both |
| `config`, `version` | none, local only | - |

### quote

Live bid/ask/mid for any number of symbols.

```bash
tradermade quote EURUSD
tradermade quote EURUSD GBPUSD XAUUSD BTCUSD
tradermade quote UK100                          # CFDs too
tradermade quote EURUSD --output json
tradermade quote EURUSD GBPUSD --output csv > prices.csv
```

### convert

`AMOUNT FROM TO` at the live rate.

```bash
tradermade convert 1000 USD INR
tradermade convert 250.50 EUR GBP
tradermade convert 1000 USD JPY --output csv
```

### symbols

What your plan can request. Filtering happens client side.

```bash
tradermade symbols
tradermade symbols --grep GBP
tradermade symbols --market crypto --grep BTC
tradermade symbols --output csv > codes.csv
```

### historical

Daily OHLC for one or more symbols on a date. `--date` takes
`2006-01-02`, `today`, or `yesterday` (the default).

```bash
tradermade historical EURUSD
tradermade historical EURUSD --date 2026-07-01
tradermade historical EURUSD GBPUSD XAUUSD --date yesterday
tradermade historical EURUSD --date 2026-07-01 --output csv
```

### timeseries

A range of candles. Intervals: `daily` (default), `hourly`, `minute`.
`--period` multiplies the interval (minute + period 15 = 15-minute candles).

Relative ranges with `--last` (units: `d` days, `w` weeks, `h` hours,
`m` minutes):

```bash
tradermade timeseries EURUSD --last 7d
tradermade timeseries EURUSD --last 2w
tradermade timeseries GBPUSD --last 12h --interval hourly
tradermade timeseries EURUSD --last 90m --interval minute --period 15
tradermade timeseries EURUSD --last 24h --interval hourly --period 4
```

Or explicit `--start` / `--end`, as a day or an intraday point
(`YYYY-MM-DD-HH:MM`). `--end` defaults to now:

```bash
tradermade timeseries EURUSD --start 2026-06-01 --end 2026-07-01
tradermade timeseries EURUSD --start 2026-06-01
tradermade timeseries EURUSD --start 2026-07-07-09:00 --end 2026-07-07-17:00 --interval minute
```

Export:

```bash
tradermade timeseries EURUSD --last 30d --output csv > eurusd.csv
```

History depth is plan-dependent (roughly: daily 10-20 years, hourly 1-8,
minute 1-5). Requests outside your range come back as a 403 with an
explanation.

### candle

One exact candle at a time you specify.

```bash
tradermade candle EURUSD --at 2026-07-07-14:30           # minute
tradermade candle EURUSD --at 2026-07-07-14:00 --hour    # hour
tradermade candle XAUUSD --at 2026-07-07-14:30 --output csv
```

### live

Streams ticks until Ctrl+C. Reconnects and resubscribes on its own if the
connection drops; keepalive pings mean a dead connection is noticed within
about 75 seconds even when the market is quiet.

```bash
tradermade live EURUSD
tradermade live EURUSD GBPUSD XAUUSD BTCUSD
tradermade live EURUSD --output json > ticks.ndjson
tradermade live EURUSD --output csv  > ticks.csv
```

Status messages go to stderr and data to stdout, so redirects capture only
the data.

### board

Full-screen dashboard. Rows update in place, movement flashes green/red,
with bid/ask, spread, day change and tick age columns plus an up/down
summary in the footer.

```bash
tradermade board add EURUSD GBPUSD XAUUSD    # saved watchlist
tradermade board remove XAUUSD
tradermade board list
tradermade board                             # run the saved list
tradermade board BTCUSD ETHUSD               # one-off, list untouched
tradermade board --sort change               # biggest mover first
```

Keys inside the board:

- `q` quit
- `s` cycle sort: watchlist order, alphabetical, biggest mover
- `c` rest-check: snapshot REST /live and show each symbol's REST mid next
  to the stream price. Deviation beyond 0.05% shows red. Quick way to see
  whether the stream and REST agree right now.

The DAY% column compares against the previous daily close, fetched over
REST at startup (weekends and holidays are skipped automatically). No REST
key: the board still runs and shows change since the session began.

The watchlist is a plain text file, one symbol per line, `#` comments
allowed. Edit it by hand if you like.

### doctor

Checks keys, REST reachability and latency, WebSocket login and plan
limits, and the config file. Exit 0 when everything passes, 1 otherwise,
so it doubles as a CI smoke test. `--output json` gives a report you can
paste into a support ticket.

```
$ tradermade doctor
rest-key  ok  abcd************wxyz (from config.json (rest_key))
rest      ok  live quote in 210ms
ws-key    ok  wsab************wxyz (from config.json (ws_key))
stream    ok  login in 180ms - plan allows 20 symbols
config    ok  ~/.config/tradermade/config.json
```

### config

```bash
tradermade config set-key YOUR_KEY          # one key for both
tradermade config set-key --rest YOUR_KEY
tradermade config set-key --ws YOUR_KEY
tradermade config show                      # masked keys and where they come from
tradermade config path
```

### version

Prints version, commit, build date, Go version and platform.

## A note on streaming latency

There is no polling and no buffering in the streaming path. The server
pushes each tick, the CLI parses it and writes it out in the same read
loop, and stdout is unbuffered. The board renders on every tick as well
(its internal 250ms timer only fades flashes and updates the age column).
Whatever latency you see is network distance to stream.tradermade.com.
Each tick carries the server timestamp (`ts` in the JSON output) if you
want to measure it yourself.

## Errors and exit codes

Exit 0 on success, 1 on any error. API errors get translated:

| HTTP | Meaning |
| --- | --- |
| 401 | bad key, or endpoint not in your plan |
| 204 | no data for those parameters |
| 400 | bad request parameters |
| 403 | market closed (weekend) or date outside your history range |

A WebSocket-scoped key used against REST gets a hint saying so. A rejected
WS login fails immediately rather than retrying forever. Transient network
errors on REST calls are retried twice before giving up.

## Development

```bash
make build     # version-stamped binary
make check     # vet + tests + gofmt, same as CI
make install   # build into GOPATH/bin
```

CI runs vet, tests and a build on Linux, Windows and macOS.

Layout:

```
main.go              entry point
cmd/                 cobra commands, kept thin
internal/api/        REST client (no CLI deps, SDK-extractable)
internal/stream/     WebSocket client: reconnect, resubscribe, keepalive
internal/board/      bubbletea dashboard
internal/watchlist/  watchlist file
internal/dates/      date parsing (yesterday, --last 30d)
internal/config/     key resolution
internal/output/     table / json / csv
```

## Roadmap

Planned: tick history export to CSV, code snippet generation
(`--show python|go|curl`), price alerts with webhooks, packaged releases
(Homebrew, Scoop, Docker), MCP server mode.

## License

MIT, see [LICENSE](LICENSE).
