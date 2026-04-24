package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

const (
	StateFile   = "/tmp/zugbar_state"
	HistoryFile = "/tmp/zugbar_history"
	CountersFile = "/tmp/zugbar_counters"
	MaxHistory = 30
)

type Target struct {
	Name     string
	Host    string
	Iface   string
	Label   string
	IsLocal bool
}

var targets = []Target{
	{Name: "BadBitch", Host: "", Iface: "enp0s31f6", Label: "WIRED", IsLocal: true},
	{Name: "EdgeRouter", Host: "10.0.0.1", Iface: "eth0", Label: "WAN", IsLocal: false},
	{Name: "TOD", Host: "10.0.0.15", Iface: "vmbr0", Label: "TOD", IsLocal: false},
}

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

func getInterfaceStats(host, iface string, isLocal bool) (rx, tx int64, online bool) {
	if isLocal {
		cmd := exec.Command("cat", "/proc/net/dev")
		out, err := cmd.Output()
		if err != nil {
			return 0, 0, false
		}
		re := regexp.MustCompile(fmt.Sprintf(`%s:\s+(\d+)\s+\d+\s+\d+\s+\d+\s+\d+\s+\d+\s+\d+\s+\d+\s+(\d+)`, iface))
		matches := re.FindStringSubmatch(string(out))
		if matches != nil {
			rx, _ = strconv.ParseInt(matches[1], 10, 64)
			tx, _ = strconv.ParseInt(matches[2], 10, 64)
			return rx, tx, true
		}
		return 0, 0, false
	}

	cmd := exec.Command("ssh", "-o", "ConnectTimeout=2", "-o", "StrictHostKeyChecking=no", fmt.Sprintf("root@%s", host), "cat", "/proc/net/dev")
	out, err := cmd.Output()
	if err != nil {
		return 0, 0, false
	}

	re := regexp.MustCompile(fmt.Sprintf(`%s:\s+(\d+)\s+\d+\s+\d+\s+\d+\s+\d+\s+\d+\s+\d+\s+\d+\s+(\d+)`, iface))
	matches := re.FindStringSubmatch(string(out))
	if matches != nil {
		rx, _ = strconv.ParseInt(matches[1], 10, 64)
		tx, _ = strconv.ParseInt(matches[2], 10, 64)
		return rx, tx, true
	}
	return 0, 0, false
}

var prevStats = make(map[string]struct{ rx, tx int64 })

func saveCounters() {
	var lines []string
	for k, v := range prevStats {
		lines = append(lines, fmt.Sprintf("%s:%d:%d", k, v.rx, v.tx))
	}
	if len(lines) > 0 {
		os.WriteFile(CountersFile, []byte(strings.Join(lines, "\n")), 0644)
	}
}

func loadCounters() {
	data, err := os.ReadFile(CountersFile)
	if err != nil {
		return
	}
	for _, line := range strings.Split(string(data), "\n") {
		parts := strings.Split(line, ":")
		if len(parts) == 3 {
			rx, _ := strconv.ParseInt(parts[1], 10, 64)
			tx, _ := strconv.ParseInt(parts[2], 10, 64)
			prevStats[parts[0]] = struct{ rx, tx int64 }{rx, tx}
		}
	}
}

func getBandwidth(host, iface string, isLocal bool) (rxbps, txbps int64, online bool) {
	rx, tx, ok := getInterfaceStats(host, iface, isLocal)
	if !ok {
		return 0, 0, false
	}

	key := host + iface
	if host == "" {
		key = "local" + iface
	}
	if prev, exists := prevStats[key]; exists {
		rxbps = rx - prev.rx
		txbps = tx - prev.tx
		if rxbps < 0 {
			rxbps = 0
		}
		if txbps < 0 {
			txbps = 0
		}
	}
	prevStats[key] = struct{ rx, tx int64 }{rx, tx}

	return rxbps, txbps, true
}

func formatBytes(b int64) string {
	if b < 1024 {
		return fmt.Sprintf("%dB", b)
	}
	if b < 1024*1024 {
		return fmt.Sprintf("%.0fKB", float64(b)/1024)
	}
	if b < 1024*1024*1024 {
		return fmt.Sprintf("%.1fMB", float64(b)/(1024*1024))
	}
	return fmt.Sprintf("%.1fGB", float64(b)/(1024*1024*1024))
}

func getStatusClass(rxbps, txbps int64) string {
	total := rxbps + txbps
	if total < 1024 {
		return "idle"
	}
	if total < 1024*1024 {
		return "light"
	}
	if total < 10*1024*1024 {
		return "medium"
	}
	if total < 100*1024*1024 {
		return "heavy"
	}
	return "extreme"
}

func getLevel(rxbps, txbps int64) string {
	if rxbps == 0 && txbps == 0 {
		return strings.Repeat("-", 20)
	}
	total := rxbps + txbps
	level := int((total * 20) / (100 * 1024 * 1024))
	if level > 20 {
		level = 20
	}
	bar := ""
	for i := 0; i < 20; i++ {
		if i < level {
			bar += "#"
		} else {
			bar += "-"
		}
	}
	return bar
}

type WaybarOutput struct {
	Text   string `json:"text"`
	Tooltip string `json:"tooltip"`
	Class  string `json:"class"`
}

func main() {
	loadCounters()

	if len(os.Args) > 1 && os.Args[1] == "cycle" {
		cycleState()
		fmt.Println("Cycled to next target")
		return
	}

	// Handle "set N" command for keyboard shortcuts
	if len(os.Args) > 1 && os.Args[1] == "set" {
		if len(os.Args) > 2 {
			targetNum := os.Args[2]
			idx, err := strconv.Atoi(targetNum)
			if err == nil {
				saveState(idx)
				// Also update counters for new target to avoid stale data
				target := targets[idx]
				rx, tx, _ := getInterfaceStats(target.Host, target.Iface, target.IsLocal)
				key := target.Host + target.Iface
				if target.IsLocal {
					key = "local" + target.Iface
				}
				prevStats[key] = struct{ rx, tx int64 }{rx, tx}
				saveCounters()
				fmt.Println("Set to target", idx)
			}
		}
		return
	}

	idx := loadState()
	if idx >= len(targets) {
		idx = 0
	}

	target := targets[idx]

	rxbps, txbps, online := getBandwidth(target.Host, target.Iface, target.IsLocal)

	saveCounters()
	saveState(idx)

	var statusIcon, statusClass, statusText string
	if online {
		statusIcon = ">"
		statusClass = getStatusClass(rxbps, txbps)
		statusText = "Up"
	} else {
		statusIcon = "X"
		statusClass = "offline"
		statusText = "Offline"
	}

	rxStr := formatBytes(rxbps)
	txStr := formatBytes(txbps)
	level := getLevel(rxbps, txbps)
	text := fmt.Sprintf("%s %s %s DL:%s UL:%s", statusIcon, target.Label, level, rxStr, txStr)

	tooltip := fmt.Sprintf("Target: %s\\nInterface: %s\\nStatus: %s\\nDownload: %s\\nUpload: %s\\n\\nClick to cycle next target",
		target.Name, target.Iface, statusText, rxStr, txStr)

	output := WaybarOutput{
		Text:   text,
		Tooltip: tooltip,
		Class:   "zugbar-" + statusClass,
	}

	buf := new(strings.Builder)
	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(false)
	enc.Encode(output)
	fmt.Print(buf.String())
}