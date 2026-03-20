package commands

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/nucleus-portal/windows-helper/internal"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// loginRequest is the payload sent to the auth endpoint.
type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// loginResponse is the expected JSON envelope from the auth endpoint.
type loginResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
}

// NewLoginCmd returns the `login` subcommand.
func NewLoginCmd() *cobra.Command {
	var apiURL string
	var email string

	cmd := &cobra.Command{
		Use:   "login",
		Short: "Authenticate with the Nucleus API and save credentials locally",
		Long: `Authenticates against the Nucleus API using your email and password.
The resulting JWT token is saved to ~/.nucleus/config.json for use by all
other commands.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLogin(apiURL, email)
		},
	}

	cmd.Flags().StringVar(&apiURL, "api-url", "", "Base URL of the Nucleus API (e.g. https://api.nucleus.example.com)")
	cmd.Flags().StringVar(&email, "email", "", "Email address to authenticate with")

	_ = cmd.MarkFlagRequired("api-url")
	_ = cmd.MarkFlagRequired("email")

	return cmd
}

func runLogin(apiURL, email string) error {
	// Normalise the base URL — strip any trailing slash.
	apiURL = strings.TrimRight(apiURL, "/")

	fmt.Printf("Logging in as %s to %s\n", email, apiURL)
	password, err := readPassword("Password: ")
	if err != nil {
		return fmt.Errorf("reading password: %w", err)
	}

	token, err := authenticate(apiURL, email, password)
	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	if err := internal.SaveToken(token, apiURL); err != nil {
		return fmt.Errorf("saving credentials: %w", err)
	}

	fmt.Println("Login successful. Credentials saved to ~/.nucleus/config.json")
	return nil
}

// authenticate calls POST /api/v1/auth/login and returns the JWT token.
func authenticate(apiURL, email, password string) (string, error) {
	payload, err := json.Marshal(loginRequest{Email: email, Password: password})
	if err != nil {
		return "", fmt.Errorf("encoding request: %w", err)
	}

	url := apiURL + "/api/v1/auth/login"
	resp, err := http.Post(url, "application/json", strings.NewReader(string(payload))) //nolint:gosec // URL is provided by the user
	if err != nil {
		return "", fmt.Errorf("contacting API at %q: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return "", fmt.Errorf("invalid email or password")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("server returned HTTP %d", resp.StatusCode)
	}

	var loginResp loginResponse
	if err := json.NewDecoder(resp.Body).Decode(&loginResp); err != nil {
		return "", fmt.Errorf("decoding server response: %w", err)
	}
	if loginResp.Token == "" {
		return "", fmt.Errorf("server returned an empty token")
	}

	return loginResp.Token, nil
}

// readPassword reads a password from the terminal without echoing it.
// Falls back to a plain stdin read if the terminal mode cannot be changed
// (e.g., when stdin is not a real TTY).
func readPassword(prompt string) (string, error) {
	fmt.Print(prompt)

	fd := int(os.Stdin.Fd())

	// Try the secure, no-echo path first.
	if term.IsTerminal(fd) {
		b, err := term.ReadPassword(fd)
		fmt.Println() // newline after the hidden input
		if err != nil {
			return "", err
		}
		return string(b), nil
	}

	// Fallback for non-interactive environments (CI, piped input).
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		return scanner.Text(), nil
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return "", fmt.Errorf("no password provided")
}
