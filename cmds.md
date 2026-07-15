# TraderMade CLI command catalog

This file lists every supported command shape and valid flag family. Replace
example symbols, dates, paths, and `YOUR_*_KEY` placeholders with real values.

## Help

```powershell
tradermade --help
tradermade quote --help
tradermade convert --help
tradermade historical --help
tradermade timeseries --help
tradermade candle --help
tradermade symbols --help
tradermade live --help
tradermade board --help
tradermade board add --help
tradermade board remove --help
tradermade board list --help
tradermade doctor --help
tradermade config --help
tradermade config set-key --help
tradermade config show --help
tradermade config path --help
tradermade version --help
tradermade completion --help
```

Only `--help` is supported. The `tradermade help` command and `-h` shorthand
are intentionally not supported.

## Global output forms

Data commands already default to table output, so no output flag is needed.
Use `--output` (or `-o`) only when selecting another format.

```powershell
tradermade quote EURUSD --output json
tradermade quote EURUSD --output csv
tradermade quote EURUSD -o json
```

For REST commands, JSON is the exact server response and CSV is constructed
locally. REST commands do not accept `--output raw`. For `live`, JSON prints
market ticks while raw also prints greetings and control frames.

Do not pass `--output table`; table is implicit and that explicit form is
rejected to keep commands concise.

## Version and diagnostics

```powershell
tradermade version
tradermade doctor
tradermade doctor --output json
```

## API-key configuration

```powershell
tradermade config set-key --rest YOUR_REST_KEY
tradermade config set-key --ws YOUR_WEBSOCKET_KEY
tradermade config set-key --rest YOUR_REST_KEY --ws YOUR_WEBSOCKET_KEY
tradermade config show
tradermade config path
```

Keys can instead come from `TRADERMADE_REST_API_KEY`,
`TRADERMADE_WS_API_KEY`, or `TRADERMADE_API_KEY`.

## Live REST quotes: `/api/v1/live`

One or more symbols are accepted.

```powershell
tradermade quote EURUSD
tradermade quote EURUSD GBPUSD XAUUSD BTCUSD
tradermade quote UK100
tradermade quote EURUSD --output json
tradermade quote EURUSD GBPUSD --output csv
tradermade quote UK100 --save quote.csv
tradermade quote UK100 --save "C:\data\market exports\quote.csv"
```

The target must include a `.csv` filename. Existing files are overwritten and
the absolute saved location is reported.

## Convert: `/api/v1/convert`

Syntax is `AMOUNT FROM TO`.

```powershell
tradermade convert 1000 USD INR
tradermade convert 250.50 EUR GBP
tradermade convert 1000 USD JPY --output json
tradermade convert 1000 USD JPY --output csv
```

## Symbol lists

`--market` is `forex` (default) or `crypto`. The complete selected list is
returned.

```powershell
tradermade symbols
tradermade symbols --market forex
tradermade symbols --market crypto
tradermade symbols --output json
tradermade symbols --market crypto --output json
tradermade symbols --output csv
```

## Historical daily OHLC: `/api/v1/historical`

`--date` accepts `YYYY-MM-DD`, `today`, or `yesterday` (default).

```powershell
tradermade historical EURUSD
tradermade historical EURUSD --date 2026-07-06
tradermade historical EURUSD --date today
tradermade historical EURUSD GBPUSD XAUUSD --date yesterday
tradermade historical EURUSD --output json
tradermade historical EURUSD --date yesterday --output csv
tradermade historical EURUSD --save historical.csv
tradermade historical EURUSD --save "C:\data\market exports\historical.csv"
```

A bare filename is created in the current working directory. A complete path
must include its `.csv` filename. Existing files are overwritten.

## Timeseries OHLC: `/api/v1/timeseries`

Choose either `--last SPAN` or `--start` with optional `--end`; never combine
the two range forms. Span units are `d`, `w`, `h`, and `m`.

Daily (maximum one year per call; period omitted or `1`):

```powershell
tradermade timeseries EURUSD --last 7d
tradermade timeseries EURUSD --last 30d
tradermade timeseries EURUSD --last 2w
tradermade timeseries EURUSD --start 2026-06-01
tradermade timeseries EURUSD --start 2026-06-01 --end 2026-07-01
tradermade timeseries EURUSD --last 30d --interval daily --period 1
```

Hourly (maximum one month; period `1`, `2`, `4`, `6`, `8`, or `24`):

```powershell
tradermade timeseries GBPUSD --last 12h --interval hourly
tradermade timeseries EURUSD --last 24h --interval hourly --period 4
tradermade timeseries EURUSD --start 2026-07-01-09:00 --end 2026-07-02-17:00 --interval hourly --period 1
```

Minute (maximum two days; period `1`, `5`, `10`, `15`, or `30`):

```powershell
tradermade timeseries EURUSD --last 90m --interval minute --period 15
tradermade timeseries EURUSD --start 2026-07-07-09:00 --end 2026-07-07-17:00 --interval minute
tradermade timeseries EURUSD --start 2026-07-07-09:00 --end 2026-07-07-17:00 --interval minute --period 15
tradermade timeseries BTCUSD --last 24h --interval minute --period 30 --weekend
```

Output and saving:

```powershell
tradermade timeseries EURUSD --last 30d --output json
tradermade timeseries EURUSD --last 30d --output csv
tradermade timeseries EURUSD --last 30d --save timeseries.csv
tradermade timeseries EURUSD --last 30d --save "C:\data\market exports\month.csv"
```

The target must include a `.csv` filename. Existing files are overwritten.

## One minute/hour candle

`--at` is required and uses `YYYY-MM-DD-HH:MM`. Default is the
`minute_historical` endpoint; `--hour` selects `hour_historical`.

```powershell
tradermade candle EURUSD --at 2026-07-07-14:30
tradermade candle EURUSD --at 2026-07-07-14:00 --hour
tradermade candle XAUUSD --at 2026-07-07-14:30 --output json
tradermade candle XAUUSD --at 2026-07-07-14:30 --output csv
```

## WebSocket v2 live stream

One or more symbols are accepted. Stop with Ctrl+C.

```powershell
tradermade live EURUSD
tradermade live EURUSD GBPUSD XAUUSD BTCUSD
tradermade live EURUSD --send-last
tradermade live EURUSD --ladder
tradermade live EURUSD --ladder --send-last
tradermade live EURUSD --output json
tradermade live EURUSD --output csv
tradermade live EURUSD --output raw
tradermade live EURUSD --save live.csv
tradermade live EURUSD --save "C:\data\market exports\ticks.csv"
tradermade live EURUSD --save "C:\Users\Omkar\Downloads\file.csv"
tradermade live EURUSD --send-last --save live.csv --output json
```

The target must include a `.csv` filename. Live saving appends so a restarted
capture continues the same file. The absolute path is printed after the first
saved tick.

## Dashboard and watchlist

Board keys: `q` quit, `s` change sorting, `c` compare with REST `/live`.

```powershell
tradermade board add EURUSD
tradermade board add EURUSD GBPUSD XAUUSD BTCUSD ETHUSD
tradermade board remove ETHUSD
tradermade board remove BTCUSD ETHUSD
tradermade board list
tradermade board
tradermade board --sort list
tradermade board --sort symbol
tradermade board --sort change
tradermade board BTCUSD ETHUSD
tradermade board BTCUSD ETHUSD --sort change
```

## Shell completion

```powershell
tradermade completion powershell
tradermade completion bash
tradermade completion zsh
tradermade completion fish
```
