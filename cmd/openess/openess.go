package main

import (
	"os"
)

func main() {
	args := ParseArgs(os.Args)

	if args.Interactive {
		InteractiveMain(args)
	} else {
		BackgroundMain(args)
	}
}
