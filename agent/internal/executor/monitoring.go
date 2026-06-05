package executor

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
)

type Alert struct {
	Level     string    `json:"level"`   // critical, warning, info
	Category  string    `json:"category"`
	Message   string    `json:"message"`
	Value     string    `json:"value"`
	Threshold string    `json:"threshold"`
	Time      time.Time `json:"time"`
}

type HealthReport struct {
	Healthy bool    `json:"healthy"`
	Alerts  []Alert `json:"alerts"`
	Score   int     `json:"score"` // 0-100
}

// RunHealthCheck performs a comprehensive system health check.
func RunHealthCheck() (*HealthReport, error) {
	report := &HealthReport{Healthy: true, Score: 100}
	var alerts []Alert

	// Disk space check
	if usage, err := disk.Usage("/"); err == nil {
		pct := usage.UsedPercent
		if pct >= 95 {
			alerts = append(alerts, Alert{
				Level: "critical", Category: "disk",
				Message:   "Festplatte fast voll",
				Value:     fmt.Sprintf("%.1f%%", pct),
				Threshold: "95%", Time: time.Now(),
			})
			report.Score -= 30
		} else if pct >= 85 {
			alerts = append(alerts, Alert{
				Level: "warning", Category: "disk",
				Message:   "Festplatte wird knapp",
				Value:     fmt.Sprintf("%.1f%%", pct),
				Threshold: "85%", Time: time.Now(),
			})
			report.Score -= 10
		}
	}

	// RAM check
	if vm, err := mem.VirtualMemory(); err == nil {
		pct := vm.UsedPercent
		if pct >= 95 {
			alerts = append(alerts, Alert{
				Level: "critical", Category: "memory",
				Message:   "RAM kritisch hoch",
				Value:     fmt.Sprintf("%.1f%%", pct),
				Threshold: "95%", Time: time.Now(),
			})
			report.Score -= 20
		} else if pct >= 85 {
			alerts = append(alerts, Alert{
				Level: "warning", Category: "memory",
				Message:   "RAM-Auslastung hoch",
				Value:     fmt.Sprintf("%.1f%%", pct),
				Threshold: "85%", Time: time.Now(),
			})
			report.Score -= 5
		}
	}

	// Load average check
	if data, err := os.ReadFile("/proc/loadavg"); err == nil {
		parts := strings.Fields(string(data))
		if len(parts) >= 1 {
			if load, err := strconv.ParseFloat(parts[0], 64); err == nil {
				// Get CPU count
				cpuCount := 1
				if cpuData, err := os.ReadFile("/proc/cpuinfo"); err == nil {
					cpuCount = strings.Count(string(cpuData), "processor\t:")
					if cpuCount == 0 {
						cpuCount = 1
					}
				}
				threshold := float64(cpuCount) * 2.0
				if load >= threshold {
					alerts = append(alerts, Alert{
						Level: "warning", Category: "load",
						Message:   "Systemlast sehr hoch",
						Value:     fmt.Sprintf("%.2f", load),
						Threshold: fmt.Sprintf("%.2f", threshold),
						Time:      time.Now(),
					})
					report.Score -= 10
				}
			}
		}
	}

	// SSL certificate expiry checks
	sslAlerts := checkSSLExpiry()
	alerts = append(alerts, sslAlerts...)
	for _, a := range sslAlerts {
		if a.Level == "critical" {
			report.Score -= 20
		} else {
			report.Score -= 5
		}
	}

	// Critical service checks
	criticalServices := []string{"nginx", "postgresql", "mysql", "mariadb"}
	for _, svc := range criticalServices {
		out, _ := runCmdOutput("systemctl", "is-active", svc)
		status := strings.TrimSpace(out)
		if status == "activating" || status == "active" {
			continue
		}
		// Check if service is even installed
		if _, err := runCmdOutput("systemctl", "status", svc); err != nil {
			continue // not installed, skip
		}
		if status == "failed" {
			alerts = append(alerts, Alert{
				Level: "critical", Category: "service",
				Message:   fmt.Sprintf("Dienst %s ausgefallen", svc),
				Value:     status, Threshold: "active",
				Time: time.Now(),
			})
			report.Score -= 25
		}
	}

	// Fail2ban: check if it's blocking too many IPs (possible attack)
	if out, err := runCmdOutput("fail2ban-client", "status"); err == nil {
		if strings.Contains(out, "Jail list:") {
			// Check for high ban count in sshd jail
			if out2, err := runCmdOutput("fail2ban-client", "status", "sshd"); err == nil {
				if strings.Contains(out2, "Currently banned:") {
					for _, line := range strings.Split(out2, "\n") {
						if strings.Contains(line, "Currently banned:") {
							parts := strings.Split(line, ":")
							if len(parts) >= 2 {
								if count, err := strconv.Atoi(strings.TrimSpace(parts[1])); err == nil && count >= 20 {
									alerts = append(alerts, Alert{
										Level: "warning", Category: "security",
										Message:   "Viele SSH-Brute-Force-Angriffe erkannt",
										Value:     fmt.Sprintf("%d gebannte IPs", count),
										Threshold: "20",
										Time:      time.Now(),
									})
									report.Score -= 5
								}
							}
						}
					}
				}
			}
		}
	}

	if report.Score < 0 {
		report.Score = 0
	}
	report.Healthy = len(alerts) == 0 || !hasCritical(alerts)
	report.Alerts = alerts
	return report, nil
}

func hasCritical(alerts []Alert) bool {
	for _, a := range alerts {
		if a.Level == "critical" {
			return true
		}
	}
	return false
}

func checkSSLExpiry() []Alert {
	var alerts []Alert
	letsEncryptDir := "/etc/letsencrypt/live"

	entries, err := os.ReadDir(letsEncryptDir)
	if err != nil {
		return alerts
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		domain := entry.Name()
		certFile := fmt.Sprintf("%s/%s/fullchain.pem", letsEncryptDir, domain)

		out, err := runCmdOutput("openssl", "x509", "-enddate", "-noout", "-in", certFile)
		if err != nil {
			continue
		}

		// Parse: notAfter=Jan  1 00:00:00 2025 GMT
		dateStr := strings.TrimPrefix(strings.TrimSpace(out), "notAfter=")
		expiry, err := time.Parse("Jan  2 15:04:05 2006 MST", dateStr)
		if err != nil {
			expiry, err = time.Parse("Jan _2 15:04:05 2006 MST", dateStr)
			if err != nil {
				continue
			}
		}

		daysLeft := int(time.Until(expiry).Hours() / 24)

		if daysLeft <= 0 {
			alerts = append(alerts, Alert{
				Level: "critical", Category: "ssl",
				Message:   fmt.Sprintf("SSL-Zertifikat für %s ist ABGELAUFEN", domain),
				Value:     fmt.Sprintf("%d Tage", daysLeft),
				Threshold: "0 Tage",
				Time:      time.Now(),
			})
		} else if daysLeft <= 7 {
			alerts = append(alerts, Alert{
				Level: "critical", Category: "ssl",
				Message:   fmt.Sprintf("SSL-Zertifikat für %s läuft in %d Tagen ab", domain, daysLeft),
				Value:     fmt.Sprintf("%d Tage", daysLeft),
				Threshold: "7 Tage",
				Time:      time.Now(),
			})
		} else if daysLeft <= 30 {
			alerts = append(alerts, Alert{
				Level: "warning", Category: "ssl",
				Message:   fmt.Sprintf("SSL-Zertifikat für %s läuft in %d Tagen ab", domain, daysLeft),
				Value:     fmt.Sprintf("%d Tage", daysLeft),
				Threshold: "30 Tage",
				Time:      time.Now(),
			})
		}
	}
	return alerts
}
