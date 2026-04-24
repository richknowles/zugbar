package main

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// === CONFIGURATION ===
const (
	StateFile   = "/tmp/zugbar_state"
	HistoryFile = "/tmp/zugbar_history"
	MaxHistory = 30
)

// DPI-independent block characters (21 levels)
var blocks = []string{
	"░", "▁", "▂", "▃", "▄", "▅", "▆", "▇", "█",
	"█▁", "█▂", "█▃", "█▄", "█▅", "█▆", "█▇", "██",
	"██▁", "██▂", "██▃", "███",
}

// Color thresholds
var colors = []struct {
	color     string
	threshold int
	name      string
}{
	{"#6272a4", 1024, "idle"},
	{"#8be9fd", 10240, "light"},
	{"#50fa7b", 102400, "moderate"},
	{"#ffb86c", 524288, "heavy"},
	{"#ff5555", 999999999, "extreme"},
}

// Target configuration
type Target struct {
	Name  string
	Host  string
	Label string
}

var targets = []Target{
	{Name: "BadBitch", Host: "10.0.0.15", Label: "BB"},
	{Name: "Router", Host: "10.0.0.1", Label: "RTR"},
	{Name: "Proxmox", Host: "10.0.0.2", Label: "PVE"},
}

// === STATE ===
func loadState() int {
	data, err := os.ReadFile(StateFile)
	if err != nil {
		return 0
	}
	idx, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0
	}
	return idx
}

func saveState(idx int) {
	os.WriteFile(StateFile, []byte(strconv.Itoa(idx)), 0644)
}

func cycleState() {
	current := loadState()
	next := (current + 1) % len(targets)
	saveState(next)
}

func loadHistory() []int {
	data, err := os.ReadFile(HistoryFile)
	if err != nil {
		return make([]int, MaxHistory)
	}
	var history []int
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if val, err := strconv.Atoi(line); err == nil {
			history = append(history, val)
		}
	}
	// Pad to MaxHistory
	for len(history) < MaxHistory {
		history = append([]int{0}, history...)
	}
	return history
}

func saveHistory(history []int) {
	lines := make([]string, len(history))
	for i, v := range history {
		lines[i] = strconv.Itoa(v)
	}
	os.WriteFile(HistoryFile, []byte(strings.Join(lines, "\n")), 0644)
}

// === PING ===
func ping(host string) (int, bool) {
	// Get all output and search for the time= pattern in the correct lines
	cmd := exec.Command("ping", "-c", "1", "-W", "2", host)
	output, err := cmd.Output()
	if err != nil {
		return 0, false
	}
	
	outputStr := string(output)
	
	// Split into lines and find the one with "from" and "time="
	lines := strings.Split(outputStr, "\n")
	for _, line := range lines {
		if strings.Contains(line, "from") && strings.Contains(line, "time=") {
			re := regexp.MustCompile(`time[=<](\d+\.?\d*)`)
			matches := re.FindStringSubmatch(line)
			if matches != nil {
ms, _ := strconv.ParseFloat(matches[1], 64)
	return int(ms * 1000), true // Convert to microseconds (e.g. 0.435ms -> 435us)
			}
		}
	}
	
	return 0, false
}

// === SPARKLINE ===
func getBlock(val, max int) string {
	if val == 0 {
		return blocks[0]
	}
	level := (val * len(blocks)) / max
	if level >= len(blocks) {
		level = len(blocks) - 1
	}
	return blocks[level]
}

func getColor(val int) string {
	for _, c := range colors {
		if val <= c.threshold {
			return c.color
		}
	}
	return colors[len(colors)-1].color
}

// === OUTPUT ===
type WaybarOutput struct {
	Text   string `json:"text"`
	Tooltip string `json:"tooltip"`
	Class  string `json:"class"`
}

func main() {
	// Check for cycle command
	if len(os.Args) > 1 && os.Args[1] == "cycle" {
		cycleState()
		fmt.Println("Cycled to next target")
		return
	}

	// Load state
	idx := loadState()
	if idx >= len(targets) {
		idx = 0
	}

	target := targets[idx]

	// Ping
	latency, online := ping(target.Host)

	// Load and update history
	history := loadHistory()
	history = append([]int{latency}, history...)
	if len(history) > MaxHistory {
		history = history[:MaxHistory]
	}

	// Calculate average
	sum := 0
	for _, v := range history[:10] {
		sum += v
	}
	avg := sum / 10

	// Save
	saveState(idx)
	saveHistory(history)

	// Build sparkline
	max := avg * 2
	if max < 10000 {
		max = 10000
	}

	sparkline := ""
	for _, v := range history[:15] {
		color := getColor(v)
		block := getBlock(v, max)
		sparkline += fmt.Sprintf("<span color='%s'>%s</span>", color, block)
	}

	// Status
	var statusIcon, statusClass, statusText, latencyStr string
	if online {
		latencyMs := float64(latency) / 1000.0
		if latencyMs < 0.5 {
			statusIcon = "●"
			statusClass = "good"
			statusText = "Online"
			latencyStr = fmt.Sprintf("%.2fms", latencyMs)
		} else if latencyMs < 2.0 {
			statusIcon = "◐"
			statusClass = "medium"
			statusText = "Online"
			latencyStr = fmt.Sprintf("%.1fms", latencyMs)
		} else if latencyMs < 50.0 {
			statusIcon = "◑"
			statusClass = "slow"
			statusText = "Online"
			latencyStr = fmt.Sprintf("%.0fms", latencyMs)
		} else {
			statusIcon = "○"
			statusClass = "offline"
			statusText = "High Latency"
			latencyStr = fmt.Sprintf("%.0fms", latencyMs)
		}
	} else {
		statusIcon = "○"
		statusClass = "offline"
		statusText = "Offline"
		latencyStr = "-"
	}

	// Format output
	text := fmt.Sprintf("%s<span color='%s'>%s</span> %s",
		statusIcon, getColor(latency), target.Label, sparkline)

	tooltip := fmt.Sprintf("Target: %s\nHost: %s\nStatus: %s\nLatency: %s\nAvg: %.1fms\n\nClick to cycle next target",
		target.Name, target.Host, statusText, latencyStr, float64(avg)/1000)

	output := WaybarOutput{
		Text:   text,
		Tooltip: tooltip,
		Class:   "zugbar-" + statusClass,
	}

	// Print JSON
	fmt.Printf("{\"text\": \"%s\", \"tooltip\": \"%s\", \"class\": \"%s\"}\n",
		escapeHTML(output.Text),
		escapeHTML(output.Tooltip),
		output.Class)
}

func escapeHTML(s string) string {
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "&", "&amp;")
	return s
}