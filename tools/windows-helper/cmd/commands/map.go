package commands

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/nucleus-portal/windows-helper/internal"
	"github.com/spf13/cobra"
)

// NewMapCmd returns the `map` subcommand.
func NewMapCmd(mapper *internal.Mapper) *cobra.Command {
	var sessionID string
	var localPort int

	cmd := &cobra.Command{
		Use:   "map",
		Short: "Start a local TCP port mapping for a Nucleus session",
		Long: `Binds 127.0.0.1:<local-port> and forwards all TCP connections to the
remote endpoint associated with the given session.

The mapping is automatically removed when the session TTL expires.
Press Ctrl+C to stop the mapping manually.

Example:
  nucleus-helper map --session-id abc123 --local-port 3389`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMap(mapper, sessionID, localPort)
		},
	}

	cmd.Flags().StringVar(&sessionID, "session-id", "", "ID of the Nucleus session to forward")
	cmd.Flags().IntVar(&localPort, "local-port", 0, "Local TCP port to bind (e.g. 3389)")

	_ = cmd.MarkFlagRequired("session-id")
	_ = cmd.MarkFlagRequired("local-port")

	return cmd
}

func runMap(mapper *internal.Mapper, sessionID string, localPort int) error {
	if localPort < 1 || localPort > 65535 {
		return fmt.Errorf("local-port must be between 1 and 65535, got %d", localPort)
	}

	if !internal.ValidateToken() {
		return fmt.Errorf("not authenticated or token expired — run 'nucleus-helper login' first")
	}

	client, err := internal.NewAPIClientFromConfig()
	if err != nil {
		return err
	}

	// Fetch the target session details from the API.
	sessions, err := client.GetActiveSessions()
	if err != nil {
		return fmt.Errorf("fetching sessions: %w", err)
	}

	var target *internal.Session
	for i := range sessions {
		if sessions[i].ID == sessionID {
			target = &sessions[i]
			break
		}
	}

	if target == nil {
		return fmt.Errorf("session %q not found or not active — run 'nucleus-helper sessions' to list active sessions", sessionID)
	}

	if err := mapper.StartMapping(
		target.ID,
		localPort,
		target.RemoteHost,
		target.RemotePort,
		target.ExpiresAt,
	); err != nil {
		return fmt.Errorf("starting mapping: %w", err)
	}

	fmt.Printf("Mapping active: 127.0.0.1:%d  →  %s:%d\n",
		localPort, target.RemoteHost, target.RemotePort)
	fmt.Printf("Session: %s  |  Device: %s\n", target.ID, target.DeviceName)
	fmt.Printf("Expires: %s\n", target.ExpiresAt.Local().Format("2006-01-02 15:04:05"))
	fmt.Println("Press Ctrl+C to stop the mapping.")

	// Block until the user interrupts or the session expires.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	select {
	case <-sigCh:
		fmt.Println("\nStopping mapping…")
		if stopErr := mapper.StopMapping(sessionID); stopErr != nil {
			// Already expired and cleaned up by the TTL watcher — not an error.
			fmt.Println("Mapping already stopped (session may have expired).")
		} else {
			fmt.Println("Mapping stopped.")
		}
	}

	return nil
}
