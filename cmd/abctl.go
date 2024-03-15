package main

import (
	"airbyte.io/abctl/internal/command/local"
	"fmt"
	"github.com/pterm/pterm"
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
			pterm.Println("install completed successfully")
			return
		}

		if os.Args[2] == cmdLocalUninstall {
			if err := lc.Uninstall(); err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			pterm.Println("uninstall completed successfully")
			return
		}
	}
	help()
}

func help() {
	// Calvin S
	pterm.Println(pterm.LightBlue("┌─┐┌┐ ┌─┐┌┬┐┬") + "         ┌─┐┬  ┌─┐┬ ┬┌─┐")
	pterm.Println(pterm.LightBlue("├─┤├┴┐│   │ │") + "    ───  ├─┤│  ├─┘├─┤├─┤")
	pterm.Println(pterm.LightBlue("┴ ┴└─┘└─┘ ┴ ┴─┘") + "       ┴ ┴┴─┘┴  ┴ ┴┴ ┴")
	pterm.Println("────────────────────────────────────")
	pterm.Println("usage: " + pterm.LightBlue("abctl") + " <command> <action>")
	pterm.Println()
	pterm.Println("commands:")
	pterm.Println(pterm.LightBlue("  local"))
	pterm.Println()
	pterm.Println("  local actions:")
	pterm.Printf(pterm.LightBlue("    %s")+"    install Airbyte locally\n", cmdLocalInstall)
	pterm.Printf(pterm.LightBlue("    %s")+"  uninstall Airbyte locally\n", cmdLocalUninstall)
}
