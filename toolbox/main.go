package main

import (
	"fmt"
	"os"

	"github.com/charmbracelet/log"
	"toolbox/cmd/secrets"
)

func init() {
	log.SetReportTimestamp(false)
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	cmd := os.Args[1]
	// Remove the subcommand from args so subcommands see their own flags
	os.Args = append([]string{os.Args[0]}, os.Args[2:]...)

	var err error
	switch cmd {
	case "secrets":
		err = secrets.Run()
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", cmd)
		printUsage()
		os.Exit(1)
	}

	if err != nil {
		log.Fatal(err)
	}
}

func printUsage() {
	fmt.Println(`Usage: toolbox <command> [options]

Commands:
  secrets     Manage secrets in Vault via SSH tunnel

Run 'toolbox <command> --help' for more information on a command.`)
}
