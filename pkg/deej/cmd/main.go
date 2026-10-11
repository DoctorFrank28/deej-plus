package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/DoctorFrank28/deej-plus/pkg/deej"
)

var (
	gitCommit  string
	versionTag string
	buildType  string

	verbose bool
	openUI  bool
)

func init() {
	flag.BoolVar(&verbose, "verbose", false, "show verbose logs (useful for debugging serial)")
	flag.BoolVar(&verbose, "v", false, "shorthand for --verbose")
	flag.BoolVar(&openUI, "open", false, "open the deej window once started (or show the running instance's)")
	flag.Parse()
}

func main() {

	// started from a shortcut, the startup folder or the registry, the working directory can be
	// anywhere: config.yaml and the logs live next to the executable
	if _, err := os.Stat("config.yaml"); err != nil {
		if exe, err := os.Executable(); err == nil {
			os.Chdir(filepath.Dir(exe))
		}
	}

	// first we need a logger
	logger, err := deej.NewLogger(buildType)
	if err != nil {
		panic(fmt.Sprintf("Failed to create logger: %v", err))
	}

	named := logger.Named("main")
	named.Debug("Created logger")

	named.Infow("Version info",
		"gitCommit", gitCommit,
		"versionTag", versionTag,
		"buildType", buildType)

	// provide a fair warning if the user's running in verbose mode
	if verbose {
		named.Debug("Verbose flag provided, all log messages will be shown")
	}

	// create the deej instance
	d, err := deej.NewDeej(logger, verbose)
	if err != nil {
		named.Fatalw("Failed to create deej object", "error", err)
	}

	// if injected by build process, set version info to show up in the tray
	if buildType != "" && (versionTag != "" || gitCommit != "") {
		identifier := gitCommit
		if versionTag != "" {
			identifier = versionTag
		}

		versionString := identifier
		if buildType != "release" {
			versionString = fmt.Sprintf("%s (%s)", identifier, buildType)
		}

		d.SetVersion(versionString)
	}

	if openUI {
		d.OpenUIOnStart()
	}

	// onwards, to glory
	if err = d.Initialize(); err != nil {
		named.Fatalw("Failed to initialize deej", "error", err)
	}
}
