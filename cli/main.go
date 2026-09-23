package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const usage = `AtomicURL CLI

Usage:
  atomicurl [global flags] <command> [command flags]

Commands:
  register  Create an account
  login     Log in and save the session cookie
  logout    Log out and remove the saved session
  create    Create a permanent short link

Global flags:
  --base-url  AtomicURL server URL (default: $ATOMICURL_URL or http://localhost:3000)
  --config    Session file path

Examples:
  atomicurl register --username denzil --email denzil@example.com --password correct-horse
  atomicurl login --username denzil --password correct-horse
  atomicurl create --destination https://example.com --title "Example"
`

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 || args[0] == "help" || args[0] == "--help" || args[0] == "-h" {
		fmt.Print(usage)
		return nil
	}

	configPath, err := defaultConfigPath()
	if err != nil {
		return err
	}
	global := flag.NewFlagSet("atomicurl", flag.ContinueOnError)
	global.SetOutput(os.Stderr)
	baseURL := global.String("base-url", envOrDefault("ATOMICURL_URL", "http://localhost:3000"), "AtomicURL server URL")
	config := global.String("config", configPath, "session file path")
	if err := global.Parse(args); err != nil {
		return err
	}
	remaining := global.Args()
	if len(remaining) == 0 {
		return errors.New("a command is required; run 'atomicurl help' for usage")
	}

	client := NewClient(*baseURL, *config)
	if err := validateClient(client); err != nil {
		return err
	}
	command, commandArgs := remaining[0], remaining[1:]
	ctx := context.Background()

	switch command {
	case "register":
		return runRegister(ctx, client, commandArgs)
	case "login":
		return runLogin(ctx, client, commandArgs)
	case "logout":
		if len(commandArgs) != 0 {
			return errors.New("logout does not accept arguments")
		}
		if err := client.Logout(ctx); err != nil {
			return err
		}
		fmt.Println("logged out")
		return nil
	case "create":
		return runCreate(ctx, client, commandArgs)
	default:
		return fmt.Errorf("unknown command %q; run 'atomicurl help' for usage", command)
	}
}

func runRegister(ctx context.Context, client *Client, args []string) error {
	flags := flag.NewFlagSet("register", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	username := flags.String("username", "", "account username")
	email := flags.String("email", "", "account email")
	password := flags.String("password", "", "account password, or use ATOMICURL_PASSWORD")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return errors.New("register does not accept positional arguments")
	}
	passwordValue, err := passwordFrom(*password)
	if err != nil {
		return err
	}
	if *username == "" || *email == "" {
		return errors.New("register requires --username and --email")
	}
	if err := client.Register(ctx, *username, *email, passwordValue); err != nil {
		return err
	}
	fmt.Println("account created")
	return nil
}

func runLogin(ctx context.Context, client *Client, args []string) error {
	flags := flag.NewFlagSet("login", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	username := flags.String("username", "", "account username")
	password := flags.String("password", "", "account password, or use ATOMICURL_PASSWORD")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return errors.New("login does not accept positional arguments")
	}
	passwordValue, err := passwordFrom(*password)
	if err != nil {
		return err
	}
	if *username == "" {
		return errors.New("login requires --username")
	}
	if err := client.Login(ctx, *username, passwordValue); err != nil {
		return err
	}
	fmt.Println("logged in")
	return nil
}

func runCreate(ctx context.Context, client *Client, args []string) error {
	flags := flag.NewFlagSet("create", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	title := flags.String("title", "", "optional link title")
	destination := flags.String("destination", "", "destination URL")
	code := flags.String("code", "", "optional custom code")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return errors.New("create does not accept positional arguments")
	}
	if *destination == "" {
		return errors.New("create requires --destination")
	}
	link, err := client.CreateLink(ctx, *title, *destination, *code)
	if err != nil {
		return err
	}
	fmt.Printf("created %s\n", client.ShortURL(link))
	return nil
}

func passwordFrom(value string) (string, error) {
	if value != "" {
		return value, nil
	}
	if value := os.Getenv("ATOMICURL_PASSWORD"); value != "" {
		return value, nil
	}
	return "", errors.New("a password is required; use --password or ATOMICURL_PASSWORD")
}

func defaultConfigPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("find config directory: %w", err)
	}
	return filepath.Join(configDir, "atomicurl", "session.json"), nil
}

func envOrDefault(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}
