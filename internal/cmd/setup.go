package cmd

import (
	"bufio"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/ankur/hass-cli/internal/config"
)

// SetupCmd runs an interactive setup wizard.
type SetupCmd struct{}

func (s *SetupCmd) Run(globals *Globals) error {
	fmt.Println()
	fmt.Println("╔══════════════════════════════════════╗")
	fmt.Println("║       hass-cli setup wizard          ║")
	fmt.Println("╚══════════════════════════════════════╝")
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)

	// ── Step 1: discover or ask for URL ──────────────────────────────────────
	url, err := resolveURL(reader)
	if err != nil {
		return err
	}

	// ── Step 2: get the token ─────────────────────────────────────────────────
	token, err := resolveToken(reader, url)
	if err != nil {
		return err
	}

	// ── Step 3: verify the connection ─────────────────────────────────────────
	fmt.Print("\nVerifying connection... ")
	if err := testConnection(url, token); err != nil {
		fmt.Println("✗")
		return fmt.Errorf("could not connect to Home Assistant: %w\n\nDouble-check your URL and token, then run `hass setup` again", err)
	}
	fmt.Println("✓")

	// ── Step 4: save ──────────────────────────────────────────────────────────
	cfg := &config.Config{URL: url, Token: token}
	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	path, _ := config.ConfigFilePath()
	fmt.Printf("\nConfig saved to %s\n", path)
	fmt.Println()
	fmt.Println("You're all set! Try:")
	fmt.Println("  hass ls                  — list all entities")
	fmt.Println("  hass ls --domain light   — list lights")
	fmt.Println("  hass on <entity_id>      — turn something on")
	fmt.Println()
	return nil
}

// resolveURL auto-discovers HA or prompts the user.
func resolveURL(reader *bufio.Reader) (string, error) {
	fmt.Println("Step 1 of 3 — Home Assistant URL")
	fmt.Println("─────────────────────────────────")
	fmt.Print("Scanning for Home Assistant on your network... ")

	found := config.DiscoverURL()
	if found != "" {
		fmt.Printf("found!\n\n")
		fmt.Printf("  Detected: %s\n\n", found)
		fmt.Print("Use this address? [Y/n]: ")
		answer, _ := reader.ReadString('\n')
		answer = strings.TrimSpace(strings.ToLower(answer))
		if answer == "" || answer == "y" || answer == "yes" {
			return found, nil
		}
	} else {
		fmt.Println("not found.")
		fmt.Println()
	}

	fmt.Println()
	fmt.Println("Enter your Home Assistant URL.")
	fmt.Println("Examples:")
	fmt.Println("  http://homeassistant.local:8123")
	fmt.Println("  http://192.168.1.100:8123")
	fmt.Println("  https://myhome.duckdns.org")
	fmt.Println()

	for {
		fmt.Print("URL: ")
		line, _ := reader.ReadString('\n')
		url := strings.TrimSpace(line)
		url = strings.TrimRight(url, "/")
		if url != "" {
			return url, nil
		}
		fmt.Println("Please enter a URL.")
	}
}

// resolveToken walks the user through getting and pasting a token.
func resolveToken(reader *bufio.Reader, haURL string) (string, error) {
	fmt.Println()
	fmt.Println("Step 2 of 3 — Long-Lived Access Token")
	fmt.Println("───────────────────────────────────────")
	fmt.Println()
	fmt.Println("You need a Long-Lived Access Token from Home Assistant.")
	fmt.Println()
	fmt.Println("How to create one:")
	fmt.Println("  1. Open Home Assistant in your browser")
	fmt.Println("  2. Click your username (bottom-left) to open your Profile")
	fmt.Println("  3. Scroll to the bottom → 'Long-Lived Access Tokens'")
	fmt.Println("  4. Click 'Create Token', give it a name (e.g. hass-cli)")
	fmt.Println("  5. Copy the token — you only see it once!")
	fmt.Println()

	tokenURL := haURL + "/profile/security"
	fmt.Printf("  Direct link: %s\n", tokenURL)
	fmt.Println()
	fmt.Print("Open this page in your browser now? [Y/n]: ")

	answer, _ := reader.ReadString('\n')
	answer = strings.TrimSpace(strings.ToLower(answer))
	if answer == "" || answer == "y" || answer == "yes" {
		if err := openBrowser(tokenURL); err != nil {
			fmt.Printf("  (Could not open browser automatically: %v)\n", err)
			fmt.Printf("  Please open manually: %s\n", tokenURL)
		} else {
			fmt.Println("  Opened in browser.")
		}
	}

	fmt.Println()
	for {
		fmt.Print("Paste your token here: ")
		line, _ := reader.ReadString('\n')
		token := strings.TrimSpace(line)
		if token != "" {
			return token, nil
		}
		fmt.Println("Token cannot be empty.")
	}
}

// testConnection verifies the URL + token work against the HA API.
func testConnection(haURL, token string) error {
	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequest("GET", haURL+"/api/", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("invalid token (HTTP 401) — make sure you copied the full token")
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %d from %s", resp.StatusCode, haURL)
	}
	return nil
}

// openBrowser opens the given URL in the system default browser.
func openBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		return fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}
	return cmd.Start()
}
