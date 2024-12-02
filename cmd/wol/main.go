package main

import (
	"fmt"
	"log"
	"os"

	ytqueuer "github.com/chadeldridge/yt-queuer/application"
	"github.com/chadeldridge/yt-queuer/cmd"
)

var (
	appName = "wol"
	help    = fmt.Sprintf(`Usage: %s [options] <srcIfaceName> <dstMAC> <port>
wol requires root privilege to bind to the inteface for broadcast.

Options:
  -h, --help     Show this help message
  -v, --version  Show the version number

Examples:
  wol -v
  wol eth0 7C:1C:4E:00:C3:5D 9
`, appName)
)

func main() {
	/*
		mac := "7C:1C:4E:00:C3:5D"
		iface := "eth0"
		port := 9
	*/

	logger := log.New(os.Stdout, "wol: ", log.LstdFlags)
	iface, mac, port := parseArgs(logger, os.Args[1:])

	err := ytqueuer.Wake(iface, mac, port)
	if err != nil {
		log.Fatal(err)
	}
}

func parseArgs(logger *log.Logger, args []string) (string, string, string) {
	ret := make([]string, 3)

	for _, arg := range args {
		switch arg {
		case "-h", "--help":
			cmd.Help(logger, help) // Exit 1
		case "-v", "--version":
			cmd.Version(logger, appName) // Exit 0
		default:
			ret = append(ret, arg)
		}
	}

	if len(args) < 3 {
		cmd.Help(logger, help) // Exit 1
	}

	return ret[0], ret[1], ret[2]
}
