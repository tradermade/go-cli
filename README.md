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

## 2. Configure the REST API key

Create a REST key from [tradermade.com/signup](https://tradermade.com/signup),
then save it locally:

```bash
tradermade config set-key --rest YOUR_REST_KEY
```

For CI or automation, set `TRADERMADE_REST_API_KEY` instead. Environment
variables override saved keys.

Check which REST key is active and where it came from:

```console
$ tradermade config show
rest    abcd********wxyz  (from config.json (rest_key))
```

## 3. Help

Use the long help flag for the complete endpoint list:

```console
$ tradermade --help
REST API:
  candle      One candle (REST /api/v1/minute_historical or /api/v1/hour_historical)
  convert     Currency conversion (REST /api/v1/convert)
  historical  Daily OHLC (REST /api/v1/historical)
  live        Live rates (REST /api/v1/live)
  symbols     Codes (REST /api/v1/live_currencies_list or /api/v1/live_crypto_list)
  timeseries  OHLC candle ranges (REST /api/v1/timeseries)

WebSocket API:
  stream      Live ticks (WebSocket wss://stream.tradermade.com/feedAdv)

REST + WebSocket:
  board       Dashboard (WebSocket /feedAdv; REST /historical and /live)
  doctor      Connectivity check (REST /live; WebSocket /feedAdv)
```

Use `tradermade COMMAND --help` to see how that endpoint is constructed,
including arguments, API parameters, valid values, limits, flags, and examples:

```bash
tradermade timeseries --help
tradermade stream --help
tradermade historical --help
```

Only the long `--help` form is supported. The `tradermade help` command and
`-h` shorthand are intentionally unavailable. The exhaustive command catalog
is in [cmds.md](cmds.md).

## 4. Live REST rates

Table output is the default. Do not pass `--output table`.

Command: `live`

Endpoint: `GET /api/v1/live`

Request:

```console
$ tradermade live EURUSD
```

Response:

```text
SYMBOL  BID      ASK      MID
EURUSD  1.14361  1.14367  1.14364

as of 2026-07-15 14:58:35 UTC
```

Save the same response as CSV by adding a filename:

```bash
tradermade live EURUSD --save eurusd-live.csv
```

The CLI saves a bare filename in the current working directory and reports its
absolute path. If `eurusd-live.csv` already exists, its previous contents are
replaced.

## 5. Timeseries OHLC

Command: `timeseries`

Endpoint: `GET /api/v1/timeseries`

Request:

```console
$ tradermade timeseries EURUSD --start 2026-07-01 --end 2026-07-03
```

Response:

```text
DATE        OPEN     HIGH     LOW      CLOSE
2026-07-01  1.14225  1.1423   1.13618  1.13785
2026-07-02  1.13785  1.14728  1.1375   1.14342
2026-07-03  1.14341  1.14622  1.14206  1.14382

EURUSD daily candles, 3 rows
```

The command supports daily, hourly, and minute intervals. Run
`tradermade timeseries --help` for `start_date`, `end_date`, `interval`, `period`,
`weekend`, relative `--last` ranges, and per-request limits.

Save the returned candles as CSV:

```bash
tradermade timeseries EURUSD --start 2026-07-01 --end 2026-07-03 --save eurusd-timeseries.csv
```

If the file already exists, the REST save replaces its previous contents.

## 6. Configure the WebSocket API key

Save the WebSocket streaming key before using `stream` or `board`:

```bash
tradermade config set-key --ws YOUR_WEBSOCKET_KEY
```

For CI or automation, set `TRADERMADE_WS_API_KEY` instead.

## 7. WebSocket stream

Command: `stream`

Endpoint: `wss://stream.tradermade.com/feedAdv`

Request:

```console
$ tradermade stream EURUSD
```

Response:

```text
connected - plan allows 5000 simultaneous symbols
subscribed: EURUSD:QUOTE
TIME                   SYMBOL                BID            ASK    BID-VOL    ASK-VOL
20260715-14:59:12.238  EURUSD        1.143490000    1.143540000    3000000    1000000
```

Stop with Ctrl+C. The client logs in with `fmt=JSON`, reconnects with backoff,
and resubscribes automatically. `--send-last` requests a cached `LAST_QUOTE`;
`--ladder` requests depth for plans with trader-ladder access. See
`tradermade stream --help` for the exact login and subscription messages.

Save ticks while keeping the terminal stream visible:

```bash
tradermade stream EURUSD --save ticks.csv
```

WebSocket saving appends. If `ticks.csv` already exists, new ticks are added
after its existing rows and the CSV header is not written again.

## 9. Commands using REST and WebSocket

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
  market tick frames for `stream`.
- `--output csv`: CLI-generated CSV with a header row.
- `--output raw`: `stream` only; includes greetings and WebSocket control frames.

## Saving CSV files

`live`, `historical`, `timeseries`, and `stream` require `--save` to include a
`.csv` filename:

```bash
tradermade live UK100 --save rates.csv
tradermade historical EURUSD --save historical.csv
tradermade timeseries EURUSD --last 30d --save "C:\data\market exports\timeseries.csv"
tradermade stream EURUSD --save ticks.csv
```

Bare filenames are saved in the current working directory, and the absolute
location is reported. A complete path is also accepted when it includes the
filename, such as `C:\Users\Omkar\Downloads\file.csv`. Directory-only targets
are rejected. REST saves overwrite; stream appends and reports its path after the
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
