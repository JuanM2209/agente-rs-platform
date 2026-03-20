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

// NewSessionsCmd returns the `sessions` subcommand.
func NewSessionsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "sessions",
		Short: "List your active Nucleus sessions",
		Long: `Fetches your active sessions from the Nucleus API and displays them in a
table.  Use the Session ID shown here with the 'map' command to create a local
TCP port mapping.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSessions()
		},
	}
}

func runSessions() error {
	if !internal.ValidateToken() {
		return fmt.Errorf("not authenticated or token expired — run 'nucleus-helper login' first")
	}

	client, err := internal.NewAPIClientFromConfig()
	if err != nil {
		return err
	}

	sessions, err := client.GetActiveSessions()
	if err != nil {
		return err
	}

	if len(sessions) == 0 {
		fmt.Println("No active sessions found.")
		return nil
	}

	printSessionsTable(sessions)
	return nil
}

func printSessionsTable(sessions []internal.Session) {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Session ID", "Device", "Remote Host", "Port", "TTL", "Status"})
	table.SetAutoWrapText(false)
	table.SetBorder(true)
	table.SetRowLine(false)
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetAlignment(tablewriter.ALIGN_LEFT)

	now := time.Now()
	for _, s := range sessions {
		ttl := s.ExpiresAt.Sub(now)
		ttlStr := formatTTL(ttl)

		status := s.Status
		if ttl <= 0 {
			status = "expired"
		}

		table.Append([]string{
			s.ID,
			s.DeviceName,
			s.RemoteHost,
			strconv.Itoa(s.RemotePort),
			ttlStr,
			status,
		})
	}

	table.Render()
}

// formatTTL converts a duration into a human-readable countdown string.
func formatTTL(d time.Duration) string {
	if d <= 0 {
		return "expired"
	}

	d = d.Round(time.Second)
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60

	if h > 0 {
		return fmt.Sprintf("%dh %02dm %02ds", h, m, s)
	}
	if m > 0 {
		return fmt.Sprintf("%dm %02ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}
