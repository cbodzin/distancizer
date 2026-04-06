package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type viewState int

const (
	viewMain viewState = iota
	viewAddName
	viewAddAddress
	viewSetOriginName
	viewSetOrigin
	viewCalculating
	viewSuggest
)

type suggestContext int

const (
	suggestForPOI suggestContext = iota
	suggestForOrigin
)

type model struct {
	state          viewState
	store          Store
	results        []CommuteResult
	cursor         int
	input          textinput.Model
	spinner        spinner.Model
	pendingPOI     string // name being added
	pendingAddress string // address being resolved
	suggestions    []GeoResult
	suggestCursor  int
	suggestFor     suggestContext
	calcTotal      int
	statusMsg      string
	statusErr      bool
	width          int
	height         int
}

// Messages
type calcProgressMsg struct {
	result  CommuteResult
	origin  Coord
	pois    []POI
	nextIdx int // index of the next POI to calculate
	total   int
}
type searchDoneMsg struct {
	query   string
	results []GeoResult
	err     error
}
type statusClearMsg struct{}

func initialModel() model {
	ti := textinput.New()
	ti.CharLimit = 200

	s := spinner.New()
	s.Spinner = spinner.Dot

	return model{
		state:   viewMain,
		store:   loadStore(),
		input:   ti,
		spinner: s,
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case statusClearMsg:
		m.statusMsg = ""
		m.statusErr = false
		return m, nil

	case calcProgressMsg:
		m.results = append(m.results, msg.result)
		if msg.nextIdx >= msg.total {
			m.state = viewMain
			m.setStatus("Calculation complete.", false)
			return m, clearStatusAfter(3 * time.Second)
		}
		return m, tea.Batch(m.spinner.Tick, calcPOI(msg.origin, msg.pois, msg.nextIdx, msg.total))

	case searchDoneMsg:
		if msg.err != nil {
			m.state = viewMain
			m.setStatus(fmt.Sprintf("Search failed: %v", msg.err), true)
			return m, clearStatusAfter(4 * time.Second)
		}
		if len(msg.results) == 0 {
			m.state = viewMain
			m.setStatus(fmt.Sprintf("No addresses found for: %s", msg.query), true)
			return m, clearStatusAfter(4 * time.Second)
		}
		if len(msg.results) == 1 {
			return m.acceptSuggestion(msg.results[0])
		}
		m.suggestions = msg.results
		m.suggestCursor = 0
		m.state = viewSuggest
		return m, nil

	case spinner.TickMsg:
		if m.state == viewCalculating {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil
	}

	switch m.state {
	case viewMain:
		return m.updateMain(msg)
	case viewAddName:
		return m.updateInput(msg, m.submitAddName)
	case viewAddAddress:
		return m.updateInput(msg, m.submitAddAddress)
	case viewSetOriginName:
		return m.updateInput(msg, m.submitSetOriginName)
	case viewSetOrigin:
		return m.updateInput(msg, m.submitSetOrigin)
	case viewCalculating:
		return m, nil
	case viewSuggest:
		return m.updateSuggest(msg)
	}
	return m, nil
}

func (m *model) setStatus(s string, isErr bool) {
	m.statusMsg = s
	m.statusErr = isErr
}

func clearStatusAfter(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(time.Time) tea.Msg {
		return statusClearMsg{}
	})
}

// Main view key handling
func (m model) updateMain(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "a":
			m.state = viewAddName
			m.input.SetValue("")
			m.input.Placeholder = "POI name (e.g. Target)"
			m.input.Focus()
			return m, textinput.Blink
		case "o":
			m.state = viewSetOriginName
			m.input.SetValue(m.store.OriginName)
			m.input.Placeholder = "Origin name (e.g. Home, Office)"
			m.input.Focus()
			return m, textinput.Blink
		case "d":
			if len(m.store.POIs) > 0 {
				m.store.POIs = append(m.store.POIs[:m.cursor], m.store.POIs[m.cursor+1:]...)
				saveStore(m.store)
				if m.cursor >= len(m.store.POIs) && m.cursor > 0 {
					m.cursor--
				}
				m.results = nil
			}
			return m, nil
		case "c":
			if len(m.store.POIs) == 0 || m.store.Origin == "" {
				m.setStatus("Need at least one POI and an origin address.", true)
				return m, clearStatusAfter(3 * time.Second)
			}
			if m.store.OriginLat == 0 && m.store.OriginLng == 0 {
				m.setStatus("Origin address not geocoded. Set it again with 'o'.", true)
				return m, clearStatusAfter(3 * time.Second)
			}
			for i := range m.store.POIs {
				if m.store.POIs[i].Lat == 0 && m.store.POIs[i].Lng == 0 {
					m.setStatus("Some POIs are not geocoded. Re-add them.", true)
					return m, clearStatusAfter(3 * time.Second)
				}
			}
			m.state = viewCalculating
			m.results = nil
			m.calcTotal = len(m.store.POIs)
			return m, tea.Batch(m.spinner.Tick, m.startCalculation())
		case "e":
			if len(m.results) == 0 {
				m.setStatus("Nothing to export. Calculate first.", true)
				return m, clearStatusAfter(3 * time.Second)
			}
			path, err := exportCSV(m.store.OriginName, m.store.Origin, m.results)
			if err != nil {
				m.setStatus(fmt.Sprintf("Export failed: %v", err), true)
				return m, clearStatusAfter(4 * time.Second)
			}
			m.setStatus(fmt.Sprintf("Exported to %s", path), false)
			return m, clearStatusAfter(5 * time.Second)
		case "j", "down":
			if m.cursor < len(m.store.POIs)-1 {
				m.cursor++
			}
			return m, nil
		case "k", "up":
			if m.cursor > 0 {
				m.cursor--
			}
			return m, nil
		}
	}
	return m, nil
}

// Generic text input handler
func (m model) updateInput(msg tea.Msg, onSubmit func(model) (model, tea.Cmd)) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "enter":
			return onSubmit(m)
		case "esc":
			m.state = viewMain
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m model) submitAddName(mm model) (model, tea.Cmd) {
	name := strings.TrimSpace(mm.input.Value())
	if name == "" {
		return mm, nil
	}
	mm.pendingPOI = name
	mm.state = viewAddAddress
	mm.input.SetValue("")
	mm.input.Placeholder = "Address (e.g. 123 Main St, City, ST)"
	mm.input.Focus()
	return mm, textinput.Blink
}

func (m model) submitAddAddress(mm model) (model, tea.Cmd) {
	addr := strings.TrimSpace(mm.input.Value())
	if addr == "" {
		return mm, nil
	}
	mm.pendingAddress = addr
	mm.suggestFor = suggestForPOI
	mm.setStatus("Searching for address...", false)
	mm.state = viewMain
	return mm, func() tea.Msg {
		time.Sleep(200 * time.Millisecond)
		results, err := searchAddresses(addr, 5)
		return searchDoneMsg{query: addr, results: results, err: err}
	}
}

func (m model) submitSetOriginName(mm model) (model, tea.Cmd) {
	name := strings.TrimSpace(mm.input.Value())
	if name == "" {
		return mm, nil
	}
	mm.store.OriginName = name
	saveStore(mm.store)
	mm.state = viewSetOrigin
	mm.input.SetValue(mm.store.Origin)
	mm.input.Placeholder = "Origin address"
	mm.input.Focus()
	return mm, textinput.Blink
}

func (m model) submitSetOrigin(mm model) (model, tea.Cmd) {
	origin := strings.TrimSpace(mm.input.Value())
	if origin == "" {
		return mm, nil
	}
	mm.pendingAddress = origin
	mm.suggestFor = suggestForOrigin
	mm.setStatus("Searching for address...", false)
	mm.state = viewMain
	return mm, func() tea.Msg {
		time.Sleep(200 * time.Millisecond)
		results, err := searchAddresses(origin, 5)
		return searchDoneMsg{query: origin, results: results, err: err}
	}
}

func (m model) updateSuggest(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "enter":
			if m.suggestCursor < len(m.suggestions) {
				return m.acceptSuggestion(m.suggestions[m.suggestCursor])
			}
			return m, nil
		case "esc":
			m.state = viewMain
			m.setStatus("Cancelled.", false)
			return m, clearStatusAfter(2 * time.Second)
		case "j", "down":
			if m.suggestCursor < len(m.suggestions)-1 {
				m.suggestCursor++
			}
			return m, nil
		case "k", "up":
			if m.suggestCursor > 0 {
				m.suggestCursor--
			}
			return m, nil
		}
	}
	return m, nil
}

func (m model) acceptSuggestion(geo GeoResult) (model, tea.Cmd) {
	switch m.suggestFor {
	case suggestForPOI:
		poi := POI{
			Name:    m.pendingPOI,
			Address: geo.DisplayName,
			Lat:     geo.Coord.Lat,
			Lng:     geo.Coord.Lng,
		}
		m.store.POIs = append(m.store.POIs, poi)
		saveStore(m.store)
		m.state = viewMain
		m.setStatus(fmt.Sprintf("Added: %s", poi.Name), false)
		return m, clearStatusAfter(3 * time.Second)
	case suggestForOrigin:
		m.store.Origin = geo.DisplayName
		m.store.OriginLat = geo.Coord.Lat
		m.store.OriginLng = geo.Coord.Lng
		saveStore(m.store)
		m.state = viewMain
		m.results = nil
		m.setStatus("Origin set.", false)
		return m, clearStatusAfter(3 * time.Second)
	}
	m.state = viewMain
	return m, nil
}

func calcPOI(origin Coord, pois []POI, idx int, total int) tea.Cmd {
	return func() tea.Msg {
		r := calculateOne(origin, pois[idx])
		return calcProgressMsg{
			result:  r,
			origin:  origin,
			pois:    pois,
			nextIdx: idx + 1,
			total:   total,
		}
	}
}

func (m model) startCalculation() tea.Cmd {
	originCoord := Coord{Lat: m.store.OriginLat, Lng: m.store.OriginLng}
	pois := make([]POI, len(m.store.POIs))
	copy(pois, m.store.POIs)
	return calcPOI(originCoord, pois, 0, len(pois))
}

// ── View ─────────────────────────────────────────────────

var (
	titleStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))
	headerStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("255"))
	dimStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("242"))
	errStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	okStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("82"))
	cursorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	helpStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	valStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("159"))
)

func (m model) View() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("  Distancizer") + "\n")
	b.WriteString(dimStyle.Render("  ─────────────────────────────────────────────") + "\n\n")

	// Input prompts
	switch m.state {
	case viewAddName:
		b.WriteString("  POI Name: " + m.input.View() + "\n")
		b.WriteString(helpStyle.Render("  enter confirm · esc cancel") + "\n\n")
	case viewAddAddress:
		b.WriteString(fmt.Sprintf("  Adding: %s\n", headerStyle.Render(m.pendingPOI)))
		b.WriteString("  Address: " + m.input.View() + "\n")
		b.WriteString(helpStyle.Render("  enter confirm · esc cancel") + "\n\n")
	case viewSetOriginName:
		b.WriteString("  Origin name: " + m.input.View() + "\n")
		b.WriteString(helpStyle.Render("  enter confirm · esc cancel") + "\n\n")
	case viewSetOrigin:
		b.WriteString(fmt.Sprintf("  Setting origin: %s\n", headerStyle.Render(m.store.OriginName)))
		b.WriteString("  Address: " + m.input.View() + "\n")
		b.WriteString(helpStyle.Render("  enter confirm · esc cancel") + "\n\n")
	case viewCalculating:
		done := len(m.results)
		remaining := m.calcTotal - done
		b.WriteString(fmt.Sprintf("  %s Calculating… %d/%d done, %d remaining\n\n",
			m.spinner.View(), done, m.calcTotal, remaining))
	case viewSuggest:
		b.WriteString(headerStyle.Render("  Did you mean?") + "\n")
		b.WriteString(dimStyle.Render(fmt.Sprintf("  Search: %s", m.pendingAddress)) + "\n\n")
		for i, s := range m.suggestions {
			prefix := "  "
			if i == m.suggestCursor {
				prefix = cursorStyle.Render("▸ ")
			}
			b.WriteString(fmt.Sprintf("%s%s\n", prefix, s.DisplayName))
		}
		b.WriteString("\n" + helpStyle.Render("  ↑↓ navigate · enter select · esc cancel") + "\n\n")
	}

	// POI list
	b.WriteString(headerStyle.Render("  Points of Interest") + "\n")
	if len(m.store.POIs) == 0 {
		b.WriteString(dimStyle.Render("  (none — press a to add)") + "\n")
	}
	for i, poi := range m.store.POIs {
		prefix := "  "
		if m.state == viewMain && i == m.cursor {
			prefix = cursorStyle.Render("▸ ")
		}
		geocoded := dimStyle.Render("○")
		if poi.Lat != 0 || poi.Lng != 0 {
			geocoded = okStyle.Render("●")
		}
		name := headerStyle.Render(poi.Name)
		addr := dimStyle.Render(poi.Address)
		b.WriteString(fmt.Sprintf("%s%s %s  %s\n", prefix, geocoded, name, addr))
	}

	// Origin
	b.WriteString("\n")
	if m.store.Origin != "" {
		b.WriteString(fmt.Sprintf("  Origin: %s — %s\n", headerStyle.Render(m.store.OriginName), valStyle.Render(m.store.Origin)))
	} else {
		b.WriteString(dimStyle.Render("  Origin: (not set — press o)") + "\n")
	}

	// Results
	if len(m.results) > 0 {
		b.WriteString("\n" + dimStyle.Render("  ──────────────────────────────────────────────────") + "\n")
		b.WriteString(fmt.Sprintf("  %s  %s  %s\n",
			headerStyle.Render(padRight("Destination", 22)),
			headerStyle.Render(padRight("Off-Peak", 12)),
			headerStyle.Render("Rush Hour*")))
		b.WriteString(dimStyle.Render("  ──────────────────────────────────────────────────") + "\n")

		for _, r := range m.results {
			name := padRight(truncate(r.POIName, 22), 22)
			if r.Error != "" {
				b.WriteString(fmt.Sprintf("  %s  %s\n", name, errStyle.Render(r.Error)))
				continue
			}
			offpeak := formatMins(r.DrivingMins)
			rush := formatMins(r.RushHourMins)
			b.WriteString(fmt.Sprintf("  %s  %s  %s\n", name, padRight(offpeak, 12), rush))
		}
		b.WriteString(dimStyle.Render("  * Rush hour estimated at 1.4x off-peak") + "\n")
	}

	// Status
	if m.statusMsg != "" {
		b.WriteString("\n")
		if m.statusErr {
			b.WriteString("  " + errStyle.Render(m.statusMsg) + "\n")
		} else {
			b.WriteString("  " + okStyle.Render(m.statusMsg) + "\n")
		}
	}

	// Help bar
	b.WriteString("\n")
	if m.state == viewMain {
		b.WriteString(helpStyle.Render("  a add · d delete · o origin · c calculate · e export csv · ↑↓ navigate · q quit") + "\n")
	}

	return b.String()
}

func padRight(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	if max <= 3 {
		return s[:max]
	}
	return s[:max-3] + "..."
}
