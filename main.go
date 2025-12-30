package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/browserutils/kooky"
	_ "github.com/browserutils/kooky/browser/chrome"
)

// --- CONFIGURATION ---
const WaybarSignal = "8" // RTMIN+8

var (
	ConfigDir  = filepath.Join(os.Getenv("HOME"), ".config", "waybar-basecamp")
	DataFile   = filepath.Join(ConfigDir, "data.json")
	OutputFile = "/tmp/waybar_basecamp.json"
)

type Config struct {
	AccountID   string `json:"account_id"`
	ProfileName string `json:"profile_name"`
}

type WaybarOutput struct {
	Text    string `json:"text"`
	Alt     string `json:"alt"`
	Tooltip string `json:"tooltip"`
	Class   string `json:"class"`
}

// --- HELPER TYPE FOR KOOKY FILTER ---
// This allows us to pass a function as a kooky.Filter
type cookieFilter func(*kooky.Cookie) bool

func (f cookieFilter) Filter(c *kooky.Cookie) bool {
	return f(c)
}

func main() {
	setupCmd := flag.NewFlagSet("setup", flag.ExitOnError)
	checkCmd := flag.NewFlagSet("check", flag.ExitOnError)

	if len(os.Args) < 2 {
		fmt.Println("Usage: waybar-basecamp [setup|check]")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "setup":
		setupCmd.Parse(os.Args[2:])
		runSetup()
	case "check":
		checkCmd.Parse(os.Args[2:])
		if err := runCheck(); err != nil {
			fmt.Printf("Error: %v\n", err)
			writeWaybar(nil)
		}
	default:
		fmt.Println("Expected 'setup' or 'check' subcommands")
		os.Exit(1)
	}
}

// --- SETUP LOGIC ---

func runSetup() {
	fmt.Println("--- Basecamp Go Setup ---")
	fmt.Println("Scanning for browser cookie databases...")

	stores := kooky.FindAllCookieStores(context.TODO())
	if len(stores) == 0 {
		log.Fatal("No cookie stores found on this system.")
	}

	fmt.Println("\n[STEP 1] Found these Cookie Files:")
	for _, store := range stores {
		if strings.Contains(store.Browser(), "hrome") || strings.Contains(store.Browser(), "hromium") {
			fmt.Printf("  - %s (%s)\n", store.FilePath(), store.Browser())
		}
	}

	fmt.Println("\nLook at the paths above. Identify the unique folder name for your profile.")
	fmt.Println("Examples: 'Profile 1', 'Default', 'Work'")
	fmt.Print("Enter unique Profile identifier: ")
	
	profileName := readInput()
	if profileName == "" {
		profileName = "Default"
	}

	fmt.Println("\n[STEP 2] Enter Account ID")
	fmt.Println("  (Found in URL: https://3.basecamp.com/YOUR_ID/...)")
	fmt.Print("Enter ID: ")
	accountID := readInput()
	if accountID == "" {
		log.Fatal("Account ID is required.")
	}

	cfg := Config{AccountID: accountID, ProfileName: profileName}
	if err := saveConfig(cfg); err != nil {
		log.Fatalf("Failed to save config: %v", err)
	}

	fmt.Printf("\nSuccess! Saved configuration.\n")
}

// --- CHECK LOGIC ---

func runCheck() error {
	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("config error: %v", err)
	}

	jar, err := getCookieJar(cfg.ProfileName)
	if err != nil {
		return err
	}

	client := &http.Client{Jar: jar}
	url := fmt.Sprintf("https://3.basecamp.com/%s/my/readings.json", cfg.AccountID)
	
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if len(bodyBytes) == 0 {
		return writeWaybar(&[]interface{}{})
	}

	// Unmarshal into generic interface
	var rawData interface{}
	if err := json.Unmarshal(bodyBytes, &rawData); err != nil {
		return fmt.Errorf("API response was not JSON (Status: %s)", resp.Status)
	}

	switch data := rawData.(type) {
	// Case 1: Root is an Object (Most likely, based on your finding)
	case map[string]interface{}:
		// Check for the specific "unreads" property
		if unreads, ok := data["unreads"].([]interface{}); ok {
			// Found it! Count the length of this array.
			return writeWaybar(&unreads)
		}
		
		// Fallback: Check for "readings" (sometimes used in older APIs)
		if readings, ok := data["readings"].([]interface{}); ok {
			return writeWaybar(&readings)
		}

		// Check if it's an error response
		if errVal, ok := data["error"]; ok {
			return fmt.Errorf("API Error: %v", errVal)
		}

		return fmt.Errorf("JSON object returned, but no 'unreads' property found: %v", data)

	// Case 2: Root is an Array (Less likely now, but good to keep as fallback)
	case []interface{}:
		return writeWaybar(&data)

	default:
		return fmt.Errorf("unknown JSON structure")
	}
}

func getCookieJar(profileName string) (http.CookieJar, error) {
	ctx := context.TODO()
	stores := kooky.FindAllCookieStores(ctx)

	var selectedStore kooky.CookieStore

	for _, store := range stores {
		path := store.FilePath()
		if !strings.Contains(path, profileName) { continue }
		if !strings.HasSuffix(path, "Cookies") { continue }
		if _, err := os.Stat(path); os.IsNotExist(err) { continue }

		selectedStore = store
		break
	}

	if selectedStore == nil {
		return nil, fmt.Errorf("no cookie store found matching profile '%s'", profileName)
	}

	// FIX: Use our custom cookieFilter type to satisfy the interface
	filter := cookieFilter(func(c *kooky.Cookie) bool {
		return strings.Contains(c.Domain, "basecamp.com") || 
			   strings.Contains(c.Domain, "37signals.com")
	})

	jar, err := selectedStore.SubJar(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to create cookie jar: %v", err)
	}

	return jar, nil
}

// --- HELPERS ---

func writeWaybar(readings *[]interface{}) error {
	count := 0
	if readings != nil {
		count = len(*readings)
	}

	out := WaybarOutput{
		Text:    "",
		Alt:     fmt.Sprintf("%d", count),
		Tooltip: fmt.Sprintf("%d Unread Notifications", count),
		Class:   "empty",
	}

	if count > 0 {
		out.Text = fmt.Sprintf("%d", count)
		out.Class = "unread"
	} else if readings == nil {
		out.Text = "err"
		out.Alt = "error"
		out.Tooltip = "Check Failed"
		out.Class = "error"
	}

	file, _ := json.Marshal(out)
	_ = os.WriteFile(OutputFile, file, 0644)

	exec.Command("pkill", "-RTMIN+"+WaybarSignal, "waybar").Run()
	return nil
}

func loadConfig() (*Config, error) {
	file, err := os.ReadFile(DataFile)
	if err != nil {
		return nil, err
	}
	var cfg Config
	err = json.Unmarshal(file, &cfg)
	return &cfg, err
}

func saveConfig(cfg Config) error {
	if _, err := os.Stat(ConfigDir); os.IsNotExist(err) {
		os.MkdirAll(ConfigDir, 0755)
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(DataFile, data, 0644)
}

func readInput() string {
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		return strings.TrimSpace(scanner.Text())
	}
	return ""
}
