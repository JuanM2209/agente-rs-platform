package commands

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/nucleus-portal/windows-helper/internal"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

// NewStatusCmd returns the `status` subcommand.
func NewStatusCmd(mapper *internal.Mapper) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show active local TCP mappings and their TTL countdown",
		Long: `Displays a table of all TCP port mappings currently managed by this
process, including each mapping's local port, remote endpoint, bytes forwarded,
and remaining TTL.

Note: mappings are held in-process.  If you restart nucleus-helper, previously
created mappings will not be listed here.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStatus(mapper)
		},
	}
}

func runStatus(mapper *internal.Mapper) error {
	mappings := mapper.ListMappings()

	// Tool-level auth summary.
	authed := internal.ValidateToken()
	if authed {
		_, apiURL, _ := internal.LoadToken()
		fmt.Printf("Authenticated  API: %s\n", apiURL)
	} else {
		fmt.Println("Not authenticated (run 'nucleus-helper login')")
	}

	fmt.Println()

	if len(mappings) == 0 {
		fmt.Println("No active mappings.")
		return nil
	}

	printMappingsTable(mappings)
	return nil
}

func printMappingsTable(mappings []internal.Mapping) {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{
		"Session ID", "Local Port", "Remote Host", "Remote Port",
		"Bytes Fwd", "Started", "TTL", "Status", "Latency", "Last Check",
	})
	table.SetAutoWrapText(false)
	table.SetBorder(true)
	table.SetRowLine(false)
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetAlignment(tablewriter.ALIGN_LEFT)

	now := time.Now()
	for _, m := range mappings {
		ttl := m.ExpiresAt.Sub(now)
		ttlStr := formatTTL(ttl)

		status := "active"
		if ttl <= 0 {
			status = "expired"
		}

		started := m.StartedAt.Local().Format("15:04:05")

		table.Append([]string{
			m.SessionID,
			strconv.Itoa(m.LocalPort),
			m.RemoteHost,
			strconv.Itoa(m.RemotePort),
			formatBytes(m.BytesFwd),
			started,
			ttlStr,
			status,
			formatLatency(m.LatencyMS),
			formatLastCheck(m.LastCheckedAt),
		})
	}

	table.Render()
}

// formatBytes returns a human-readable byte count.
func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

func formatLatency(latencyMS int) string {
	if latencyMS <= 0 {
		return "pending"
	}
	return fmt.Sprintf("%d ms", latencyMS)
}

func formatLastCheck(ts time.Time) string {
	if ts.IsZero() {
		return "pending"
	}
	return ts.Local().Format("15:04:05")
}
