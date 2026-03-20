package commands

import (
	"fmt"

	"github.com/nucleus-portal/windows-helper/internal"
	"github.com/spf13/cobra"
)

// NewUnmapCmd returns the `unmap` subcommand.
func NewUnmapCmd(mapper *internal.Mapper) *cobra.Command {
	var sessionID string

	cmd := &cobra.Command{
		Use:   "unmap",
		Short: "Stop the local TCP mapping for a session",
		Long: `Stops the local TCP port mapping for the specified session ID and closes
the bound local port.

Example:
  nucleus-helper unmap --session-id abc123`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUnmap(mapper, sessionID)
		},
	}

	cmd.Flags().StringVar(&sessionID, "session-id", "", "ID of the session whose mapping should be stopped")
	_ = cmd.MarkFlagRequired("session-id")

	return cmd
}

func runUnmap(mapper *internal.Mapper, sessionID string) error {
	if err := mapper.StopMapping(sessionID); err != nil {
		return fmt.Errorf("stopping mapping for session %q: %w", sessionID, err)
	}

	fmt.Printf("Mapping for session %q stopped.\n", sessionID)
	return nil
}
