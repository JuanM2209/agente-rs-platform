// nucleus-helper — Windows CLI for local TCP port mapping of Nucleus sessions.
//
// Architecture note:
// This CLI is the MVP implementation. The internal/mapper.go TCP forwarding
// engine is designed to be consumed by a future Windows systray GUI application.
// No GUI dependencies are introduced here.
//
// The Mapper is created once at startup in main() and passed down into each
// subcommand that needs it.  This means 'map', 'unmap', and 'status' share a
// single in-process registry of active mappings.  A future tray app can replace
// this wiring by constructing a Mapper once and passing it to the tray icon and
// its menu actions instead of Cobra commands.
package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/nucleus-portal/windows-helper/cmd/commands"
	"github.com/nucleus-portal/windows-helper/internal"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func main() {
	// Initialise structured logging.  Pretty-print to stderr so that stdout
	// remains clean for table output that callers might parse.
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	// Single shared Mapper — all subcommands that touch mappings receive a
	// pointer to this instance.
	mapper := internal.NewMapper()

	// Ensure all mappings are torn down cleanly on any exit signal.
	registerShutdownHook(mapper)

	rootCmd := buildRootCmd(mapper)

	if err := rootCmd.Execute(); err != nil {
		// Cobra already prints the error; we just need a non-zero exit.
		os.Exit(1)
	}
}

// buildRootCmd assembles the full Cobra command tree.
func buildRootCmd(mapper *internal.Mapper) *cobra.Command {
	var verbose bool

	root := &cobra.Command{
		Use:   "nucleus-helper",
		Short: "Local TCP port mapper for Nucleus remote-access sessions",
		Long: `nucleus-helper manages local TCP port forwarding for Nucleus sessions.

It authenticates with the Nucleus API, retrieves your active sessions, and
forwards a chosen local port to the remote endpoint associated with each session.

Run 'nucleus-helper --help' on any subcommand for detailed usage.`,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			if verbose {
				zerolog.SetGlobalLevel(zerolog.DebugLevel)
			}
		},
	}

	root.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose (debug) logging")

	root.AddCommand(commands.NewLoginCmd())
	root.AddCommand(commands.NewSessionsCmd())
	root.AddCommand(commands.NewMapCmd(mapper))
	root.AddCommand(commands.NewUnmapCmd(mapper))
	root.AddCommand(commands.NewStatusCmd(mapper))

	return root
}

// registerShutdownHook listens for OS signals and cleanly tears down all active
// mappings before the process exits.
func registerShutdownHook(mapper *internal.Mapper) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		sig := <-sigCh
		log.Info().Str("signal", sig.String()).Msg("shutting down — stopping all mappings")
		mapper.StopAll()
		os.Exit(0)
	}()
}
