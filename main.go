package main

import (
	"fmt"
	"math"
	"math/rand"
	"os"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type tickMsg time.Time

type star struct {
	x, y  float64
	seed  float64
	size  int
	glyph string
}

type comet struct {
	x, y float64
	vx   float64
	vy   float64
	life int
}

type model struct {
	width, height int
	frame         int
	stars         []star
	comets        []comet
	rand          *rand.Rand
}

var (
	bgStyle = lipgloss.NewStyle().Background(lipgloss.Color("#050711"))

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#F8E7A2")).
			Background(lipgloss.Color("#17172E")).
			Padding(0, 2).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#8A7CFF"))

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7F8DBA")).
			Background(lipgloss.Color("#050711"))

	nebulaColors = []string{"#513B91", "#8A4FFF", "#FF6AD5", "#65D9FF", "#F8E7A2", "#FFFFFF"}
	starGlyphs   = []string{"·", "•", "✦", "✧", "✶", "✷", "*"}
)

func main() {
	p := tea.NewProgram(newModel(), tea.WithFPS(60))
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "stars: %v\n", err)
		os.Exit(1)
	}
}

func newModel() model {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	return model{rand: r}
}

func (m model) Init() tea.Cmd {
	return tick()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit
		case "r", "space":
			m.seedStars()
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.seedStars()
	case tickMsg:
		m.frame++
		m.updateComets()
		return m, tick()
	}
	return m, nil
}

func (m model) View() tea.View {
	if m.width < 20 || m.height < 8 {
		v := tea.NewView("Make terminal bigger · q quits")
		v.AltScreen = true
		return v
	}

	content := m.renderScene()
	view := tea.NewView(content)
	view.AltScreen = true
	view.WindowTitle = "stars"
	view.BackgroundColor = lipgloss.Color("#050711")
	return view
}

func tick() tea.Cmd {
	return tea.Tick(55*time.Millisecond, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func (m *model) seedStars() {
	if m.width <= 0 || m.height <= 0 {
		return
	}

	count := max(40, (m.width*m.height)/18)
	m.stars = make([]star, count)
	for i := range m.stars {
		m.stars[i] = star{
			x:     m.rand.Float64(),
			y:     m.rand.Float64(),
			seed:  m.rand.Float64() * math.Pi * 2,
			size:  m.rand.Intn(3),
			glyph: starGlyphs[m.rand.Intn(len(starGlyphs))],
		}
	}
	m.comets = nil
}

func (m *model) updateComets() {
	if m.width == 0 || m.height == 0 {
		return
	}

	if len(m.comets) < 3 && m.rand.Float64() < 0.055 {
		m.comets = append(m.comets, comet{
			x:    float64(m.rand.Intn(max(1, m.width/2))),
			y:    float64(m.rand.Intn(max(1, m.height/3))),
			vx:   1.7 + m.rand.Float64()*1.5,
			vy:   0.35 + m.rand.Float64()*0.45,
			life: 18 + m.rand.Intn(18),
		})
	}

	live := m.comets[:0]
	for _, c := range m.comets {
		c.x += c.vx
		c.y += c.vy
		c.life--
		if c.life > 0 && int(c.x) < m.width+8 && int(c.y) < m.height {
			live = append(live, c)
		}
	}
	m.comets = live
}

func (m model) renderScene() string {
	skyH := m.height - 3
	canvas := make([][]cell, skyH)
	for y := range canvas {
		canvas[y] = make([]cell, m.width)
		for x := range canvas[y] {
			canvas[y][x] = cell{ch: " ", color: nebula(x, y, m.width, skyH, m.frame)}
		}
	}

	for _, s := range m.stars {
		x := int(s.x * float64(max(1, m.width-1)))
		y := int(s.y * float64(max(1, skyH-1)))
		pulse := (math.Sin(float64(m.frame)*0.13+s.seed) + 1) / 2
		canvas[y][x] = cell{ch: s.glyph, color: starColor(pulse, s.size), bold: pulse > 0.72}
	}

	for _, c := range m.comets {
		m.paintComet(canvas, c)
	}

	var b strings.Builder
	for y := range canvas {
		for x := range canvas[y] {
			b.WriteString(renderCell(canvas[y][x]))
		}
		if y < len(canvas)-1 {
			b.WriteByte('\n')
		}
	}

	help := helpStyle.Render("q quit · r reseed")
	return bgStyle.Width(m.width).Height(skyH).Render(b.String()) + "\n" + centerLine(help, m.width)
}

type cell struct {
	ch    string
	color string
	bold  bool
}

func renderCell(c cell) string {
	style := lipgloss.NewStyle().Foreground(lipgloss.Color(c.color)).Background(lipgloss.Color("#050711"))
	if c.bold {
		style = style.Bold(true)
	}
	return style.Render(c.ch)
}

func nebula(x, y, w, h, frame int) string {
	nx := float64(x)/float64(max(1, w)) - 0.5
	ny := float64(y)/float64(max(1, h)) - 0.5
	wave := math.Sin(nx*10+float64(frame)*0.025) + math.Cos(ny*8-float64(frame)*0.018)
	dist := math.Sqrt(nx*nx + ny*ny)
	switch {
	case dist < 0.18 && wave > 1.1:
		return "#17172E"
	case wave > 1.45:
		return "#101A33"
	case wave < -1.35:
		return "#0B1024"
	default:
		return "#050711"
	}
}

func starColor(pulse float64, size int) string {
	idx := int(pulse*float64(len(nebulaColors)-1)) + size
	if idx >= len(nebulaColors) {
		idx = len(nebulaColors) - 1
	}
	return nebulaColors[idx]
}

func (m model) paintComet(canvas [][]cell, c comet) {
	for i := 0; i < 10; i++ {
		x := int(c.x - float64(i)*1.25)
		y := int(c.y - float64(i)*0.35)
		if y < 0 || y >= len(canvas) || x < 0 || x >= len(canvas[y]) {
			continue
		}
		glyph := "═"
		color := "#65D9FF"
		bold := false
		if i == 0 {
			glyph = "✦"
			color = "#FFFFFF"
			bold = true
		} else if i > 5 {
			glyph = "·"
			color = "#513B91"
		}
		canvas[y][x] = cell{ch: glyph, color: color, bold: bold}
	}
}

func centerLine(s string, width int) string {
	plainWidth := lipgloss.Width(s)
	if plainWidth >= width {
		return s
	}
	left := strings.Repeat(" ", (width-plainWidth)/2)
	return left + s
}
