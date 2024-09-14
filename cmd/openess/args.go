package main

import (
	"fmt"
	"openess/internal/log"
	"os"
	"strings"
)

type Args struct {
	LogLevel    int
	DeviceAddr  *string
	ConfPath    string
	Interactive bool
}

func helpMessage() string {
	var builder strings.Builder
	fmt.Fprintf(&builder, "Usage: %s [OPTIONS] \n", os.Args[0])
	fmt.Fprintln(&builder, "Options:")
	fmt.Fprintln(&builder, "\t-l, --log\t\t logging level: off, warn, info, debug (default off)")
	fmt.Fprintln(&builder, "\t-d, --device\t\t datalogger IP address (overrides address from config)")
	fmt.Fprintln(&builder, "\t-c, --config\t path to the config file (default 'data/config.json')")
	fmt.Fprintln(&builder, "\t-b, --background\t run in background, otherwise starts interactive shell")

	return builder.String()
}

func ParseArgs(args []string) Args {
	parsed := Args{
		LogLevel:    log.LOG_INFO,
		DeviceAddr:  nil,
		ConfPath:    "data/config.json",
		Interactive: true,
	}

	var key string

	for _, arg := range args[1:] {
		switch arg {
		case "-b", "--background":
			parsed.Interactive = false
			continue
		case "-h", "--help":
			fmt.Fprintf(os.Stderr, helpMessage())
			os.Exit(0)
		}

		if key == "" {
			key = arg
			continue
		}

		switch key {
		case "-l", "--log":
			switch arg {
			case "off":
				parsed.LogLevel = log.LOG_OFF
			case "warn":
				parsed.LogLevel = log.LOG_ERROR
			case "info":
				parsed.LogLevel = log.LOG_INFO
			case "debug":
				parsed.LogLevel = log.LOG_DEBUG
			default:
				fmt.Fprintln(os.Stderr, "invalid log level")
				os.Exit(1)
			}
		case "-d", "--device":
		    addr := arg
			parsed.DeviceAddr = &addr
		case "-c", "--config":
			parsed.ConfPath = arg
		default:
			fmt.Fprintf(os.Stderr, "invalid argument: '%s'\n", key)
			fmt.Fprintf(os.Stderr, helpMessage())
			os.Exit(1)
		}

		key = ""
	}

	return parsed
}
