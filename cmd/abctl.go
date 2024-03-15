package main

import (
	"airbyte.io/abctl/internal/command/local"
	"fmt"
	"os"
)

const (
	cmdLocal          = "local"
	cmdLocalInstall   = "install"
	cmdLocalUninstall = "uninstall"
)

func main() {
	// TODO: replace with real sub-command support!
	if len(os.Args) != 3 {
		help()
		return
	}
	if os.Args[1] == cmdLocal {
		lc, err := local.New()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		if os.Args[2] == cmdLocalInstall {
			if err := lc.Install(); err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			fmt.Println("success")
			return
		}

		if os.Args[2] == cmdLocalUninstall {
			if err := lc.Uninstall(); err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			fmt.Println("success")
			return
		}
	}
	help()
}

func help() {
	// Calvin S
	fmt.Println("┌─┐┌┐ ┌─┐┌┬┐┬         ┌─┐┬  ┌─┐┬ ┬┌─┐")
	fmt.Println("├─┤├┴┐│   │ │    ───  ├─┤│  ├─┘├─┤├─┤")
	fmt.Println("┴ ┴└─┘└─┘ ┴ ┴─┘       ┴ ┴┴─┘┴  ┴ ┴┴ ┴")
	fmt.Println("────────────────────────────────────")
	fmt.Println("usage: abctl <command> <action>")
	fmt.Println()
	fmt.Println("commands:")
	fmt.Println("  local")
	fmt.Println()
	fmt.Println("  local actions:")
	fmt.Printf("    %s    install Airbyte locally\n", cmdLocalInstall)
	fmt.Printf("    %s  uninstall Airbyte locally\n", cmdLocalUninstall)
}
