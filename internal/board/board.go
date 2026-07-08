// Package board is the live watchlist dashboard (bubbletea TUI).
//
// Stream ticks arrive via Program.Send from the stream goroutine. A 250ms
// frame timer handles flash decay and tick ages; it does not gate data.
// With a REST key we also fetch previous daily closes at startup so the
// change column shows day change instead of session change.
package board

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/tradermade/tradermade-cli/internal/api"
	"github.com/tradermade/tradermade-cli/internal/dates"
	"github.com/tradermade/tradermade-cli/internal/stream"
)

// flashDuration is how long a row highlights after a tick.
const flashDuration = 300 * time.Millisecond

// Sort modes, cycled with the "s" key.
const (
	SortList   = "list"   // watchlist order
	SortSymbol = "symbol" // alphabetical
	SortChange = "change" // biggest mover first
)

// Options configures a board session.
type Options struct {
	Key     string // WebSocket key (required)
	RESTKey string // optional - enables day-change vs previous close
	Symbols []string
	Sort    string // initial sort mode; defaults to SortList
}

// Run blocks until the user quits (q / Esc / Ctrl+C) or the stream fails
// permanently (e.g. invalid API key).
func Run(ctx context.Context, opts Options) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	if opts.Sort == "" {
		opts.Sort = SortList
	}
	p := tea.NewProgram(newModel(opts.Symbols, opts.Sort, opts.RESTKey), tea.WithAltScreen(), tea.WithContext(ctx))

	if opts.RESTKey != "" {
		go fetchPrevCloses(ctx, p, opts.RESTKey, opts.Symbols)
	}

	streamErr := make(chan error, 1)
	go func() {
		err := stream.Run(ctx, stream.Options{
			Key:      opts.Key,
			Symbols:  opts.Symbols,
			SendLast: true, // slow symbols show their cached price immediately
			OnTick: func(t stream.Tick, _ []byte) {
				p.Send(tickMsg{tick: t, at: time.Now()})
			},
			Logf: func(f string, a ...any) {
				p.Send(statusMsg(fmt.Sprintf(f, a...)))
			},
		})
		streamErr <- err
		if err != nil {
			p.Send(fatalMsg{err})
		}
	}()

	_, uiErr := p.Run()
	cancel() // stop the stream once the UI is gone

	if err := <-streamErr; err != nil {
		return err
	}
	if uiErr != nil && ctx.Err() == nil {
		return uiErr
	}
	return nil
}

// fetchPrevCloses walks back from yesterday (up to 6 days, skipping weekends
// and holidays that 403) until every symbol has a previous daily close.
func fetchPrevCloses(ctx context.Context, p *tea.Program, restKey string, symbols []string) {
	client := api.New(restKey)
	closes := make(map[string]float64, len(symbols))
	day := time.Now().UTC()
	for i := 0; i < 6 && len(closes) < len(symbols); i++ {
		day = day.AddDate(0, 0, -1)
		resp, err := client.Historical(ctx, symbols, day.Format(dates.DayFormat))
		if err != nil {
			continue // weekend/holiday - walk further back
		}
		for _, q := range resp.Quotes {
			sym := q.Symbol()
			if _, seen := closes[sym]; seen {
				continue // keep the most recent close
			}
			if f, err := strconv.ParseFloat(q.Close.String(), 64); err == nil && f != 0 {
				closes[sym] = f
			}
		}
	}
	if len(closes) > 0 && ctx.Err() == nil {
		p.Send(prevCloseMsg(closes))
	}
}

// Messages flowing into the Bubble Tea update loop.
type (
	tickMsg struct {
		tick stream.Tick
		at   time.Time
	}
	statusMsg    string
	fatalMsg     struct{ err error }
	frameMsg     time.Time
	prevCloseMsg map[string]float64
	restCheckMsg struct {
		mids map[string]float64 // REST mid by symbol
		at   time.Time
		err  error
	}
)

// fetchRestCheck grabs a REST /live snapshot for the "c" key. Runs as a
// tea.Cmd so the UI never blocks.
func fetchRestCheck(restKey string, symbols []string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		resp, err := api.New(restKey).Live(ctx, symbols)
		if err != nil {
			return restCheckMsg{err: err}
		}
		mids := make(map[string]float64, len(resp.Quotes))
		for _, q := range resp.Quotes {
			mids[q.Symbol()] = q.Mid
		}
		return restCheckMsg{mids: mids, at: time.Now()}
	}
}

// row is the live state of one symbol.
type row struct {
	bid, ask  string  // raw wire strings - full precision preserved
	bidF      float64 // parsed, for direction and change math
	openBid   float64 // first bid seen this session - fallback change basis
	dir       int     // -1 down, 0 flat/unknown, +1 up (persists between ticks)
	flashedAt time.Time
	lastTick  time.Time
}

type model struct {
	order     []string        // watchlist order = order given by the user
	rows      map[string]*row // keyed by symbol
	prevClose map[string]float64
	sortMode  string
	status    string
	err       error
	now       time.Time
	width     int
	ticks     int // total ticks this session, for the footer

	// REST cross-check state ("c" key).
	restKey  string
	restMids map[string]float64
	restAt   time.Time
	checking bool
}

func newModel(symbols []string, sortMode, restKey string) model {
	rows := make(map[string]*row, len(symbols))
	order := make([]string, 0, len(symbols))
	for _, s := range symbols {
		s = strings.ToUpper(strings.TrimSpace(s))
		if _, ok := rows[s]; s == "" || ok {
			continue
		}
		rows[s] = &row{}
		order = append(order, s)
	}
	return model{
		order:     order,
		rows:      rows,
		prevClose: map[string]float64{},
		sortMode:  sortMode,
		restKey:   restKey,
		status:    "connecting...",
		now:       time.Now(),
	}
}

func frameCmd() tea.Cmd {
	return tea.Tick(250*time.Millisecond, func(t time.Time) tea.Msg { return frameMsg(t) })
}

func (m model) Init() tea.Cmd {
	return frameCmd()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			return m, tea.Quit
		case "s":
			m.sortMode = nextSort(m.sortMode)
		case "c":
			if m.restKey == "" {
				m.status = "rest-check needs a REST key - tradermade config set-key --rest"
			} else if !m.checking {
				m.checking = true
				m.status = "checking against REST..."
				return m, fetchRestCheck(m.restKey, m.order)
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width

	case tickMsg:
		sym := msg.tick.Symbol
		r, ok := m.rows[sym]
		if !ok {
			// Server sent something we didn't ask for - show it anyway.
			r = &row{}
			m.rows[sym] = r
			m.order = append(m.order, sym)
		}
		if bid, err := strconv.ParseFloat(msg.tick.Bid, 64); err == nil {
			if r.openBid == 0 {
				r.openBid = bid
			}
			switch {
			case r.bidF != 0 && bid > r.bidF:
				r.dir = 1
			case r.bidF != 0 && bid < r.bidF:
				r.dir = -1
			}
			r.bidF = bid
		}
		r.bid, r.ask = msg.tick.Bid, msg.tick.Ask
		r.flashedAt = msg.at
		r.lastTick = msg.at
		m.ticks++

	case prevCloseMsg:
		for sym, close := range msg {
			m.prevClose[sym] = close
		}

	case restCheckMsg:
		m.checking = false
		if msg.err != nil {
			m.status = "rest-check failed: " + msg.err.Error()
		} else {
			m.restMids = msg.mids
			m.restAt = msg.at
			m.status = "REST snapshot loaded - press c to refresh"
		}

	case statusMsg:
		m.status = string(msg)

	case fatalMsg:
		m.err = msg.err
		return m, tea.Quit

	case frameMsg:
		m.now = time.Time(msg)
		return m, frameCmd()
	}
	return m, nil
}

func nextSort(mode string) string {
	switch mode {
	case SortList:
		return SortSymbol
	case SortSymbol:
		return SortChange
	default:
		return SortList
	}
}

// changeBasis returns the reference price for the Δ column: previous daily
// close when known, otherwise the session's first bid.
func (m model) changeBasis(sym string) float64 {
	if pc := m.prevClose[sym]; pc != 0 {
		return pc
	}
	if r := m.rows[sym]; r != nil {
		return r.openBid
	}
	return 0
}

// sortedOrder returns the display order for the current sort mode.
func sortedOrder(order []string, changeVal func(string) float64, mode string) []string {
	out := make([]string, len(order))
	copy(out, order)
	switch mode {
	case SortSymbol:
		sort.Strings(out)
	case SortChange:
		sort.SliceStable(out, func(i, j int) bool {
			return changeVal(out[i]) > changeVal(out[j])
		})
	}
	return out
}

// changeVal is the sortable day-change fraction for one symbol; rows without
// data sort to the bottom.
func (m model) changeVal(sym string) float64 {
	r := m.rows[sym]
	basis := m.changeBasis(sym)
	if r == nil || r.bidF == 0 || basis == 0 {
		return -1e18
	}
	return (r.bidF - basis) / basis
}

// Styles - basic ANSI colors only, so every terminal theme renders sanely.
var (
	titleStyle  = lipgloss.NewStyle().Bold(true)
	accentStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	headerStyle = lipgloss.NewStyle().Faint(true)
	upStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	downStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	flashUp     = lipgloss.NewStyle().Foreground(lipgloss.Color("0")).Background(lipgloss.Color("2"))
	flashDown   = lipgloss.NewStyle().Foreground(lipgloss.Color("0")).Background(lipgloss.Color("1"))
	waitStyle   = lipgloss.NewStyle().Faint(true)
	footerStyle = lipgloss.NewStyle().Faint(true)
)

func (m model) View() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("tradermade") + accentStyle.Render(" board") + "\n\n")

	deltaLabel := "OPEN%"
	if len(m.prevClose) > 0 {
		deltaLabel = "DAY%"
	}
	header := fmt.Sprintf(" %-10s %2s %14s %14s %12s %10s %8s",
		"SYMBOL", "", "BID", "ASK", "SPREAD", deltaLabel, "LAST")
	if m.restMids != nil {
		header += fmt.Sprintf(" %14s %9s", "REST MID", "DIFF")
	}
	b.WriteString(headerStyle.Render(header) + "\n")

	up, down := 0, 0
	for _, sym := range sortedOrder(m.order, m.changeVal, m.sortMode) {
		r := m.rows[sym]
		if r.lastTick.IsZero() {
			b.WriteString(fmt.Sprintf(" %-10s %2s ", sym, "") +
				waitStyle.Render(fmt.Sprintf("%14s %14s %12s %10s %8s", "-", "-", "-", "-", "waiting")) + "\n")
			continue
		}

		arrow, style := " ", lipgloss.NewStyle()
		switch r.dir {
		case 1:
			arrow, style = "↑", upStyle
			up++
		case -1:
			arrow, style = "↓", downStyle
			down++
		}
		// Flash the whole price block briefly on every tick.
		if m.now.Sub(r.flashedAt) < flashDuration {
			if r.dir < 0 {
				style = flashDown
			} else {
				style = flashUp
			}
		}

		prices := fmt.Sprintf("%14s %14s %12s", r.bid, r.ask, spread(r.bid, r.ask))
		line := fmt.Sprintf(" %-10s %2s ", sym, style.Render(arrow)) +
			style.Render(prices) +
			fmt.Sprintf(" %10s %8s", changePct(m.changeBasis(sym), r.bidF), age(m.now, r.lastTick))

		if m.restMids != nil {
			line += restCheckCells(m.restMids[sym], r)
		}
		b.WriteString(line + "\n")
	}

	summary := fmt.Sprintf(" %s  |  %d↑ %d↓  |  %d ticks  |  sort: %s (s)",
		m.status, up, down, m.ticks, m.sortMode)
	if !m.restAt.IsZero() {
		summary += fmt.Sprintf("  |  rest %s ago", age(m.now, m.restAt))
	}
	summary += "  |  c rest-check  |  q quit"
	b.WriteString("\n" + footerStyle.Render(summary) + "\n")
	return b.String()
}

// restDivergenceThreshold flags stream-vs-REST gaps worth a second look.
// 0.05% on a mid price is far beyond normal snapshot skew for liquid pairs.
const restDivergenceThreshold = 0.05

// restCheckCells renders the REST MID and DIFF columns for one row.
// DIFF is the stream mid's deviation from the REST snapshot mid; large
// gaps render in red as a data-quality flag.
func restCheckCells(restMid float64, r *row) string {
	if restMid == 0 {
		return fmt.Sprintf(" %14s %9s", "-", "-")
	}
	streamMid := r.bidF
	if ask, err := strconv.ParseFloat(r.ask, 64); err == nil && r.bidF != 0 {
		streamMid = (r.bidF + ask) / 2
	}
	if streamMid == 0 {
		return fmt.Sprintf(" %14s %9s", strconv.FormatFloat(restMid, 'f', decimals(r.bid), 64), "-")
	}
	diff := (streamMid - restMid) / restMid * 100
	cell := fmt.Sprintf("%+.3f%%", diff)
	if diff > restDivergenceThreshold || diff < -restDivergenceThreshold {
		cell = downStyle.Render(cell)
	} else {
		cell = footerStyle.Render(cell)
	}
	return fmt.Sprintf(" %14s %9s", strconv.FormatFloat(restMid, 'f', decimals(r.bid), 64), cell)
}

// spread computes ask-bid, keeping the same decimal precision the wire uses.
func spread(bid, ask string) string {
	b, err1 := strconv.ParseFloat(bid, 64)
	a, err2 := strconv.ParseFloat(ask, 64)
	if err1 != nil || err2 != nil {
		return "-"
	}
	return strconv.FormatFloat(a-b, 'f', decimals(bid), 64)
}

// decimals counts fraction digits in a wire price string, capped at 8.
func decimals(price string) int {
	if i := strings.IndexByte(price, '.'); i >= 0 {
		if n := len(price) - i - 1; n <= 8 {
			return n
		}
		return 8
	}
	return 0
}

// changePct is the move against the change basis (prev close or session open).
func changePct(basis, current float64) string {
	if basis == 0 || current == 0 {
		return "-"
	}
	return fmt.Sprintf("%+.3f%%", (current-basis)/basis*100)
}

// age renders how long ago the last tick arrived.
func age(now, last time.Time) string {
	d := now.Sub(last)
	switch {
	case d < time.Second:
		return "now"
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	default:
		return fmt.Sprintf("%dh", int(d.Hours()))
	}
}
