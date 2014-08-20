package main

import (
	"fmt"
	"os"

	"github.com/coreos/fleet/job"
)

var (
	cmdLoadUnits = &Command{
		Name:    "load",
		Summary: "Schedule one or more units in the cluster, first submitting them if necessary.",
		Usage:   "[--no-block|--block-attempts=N] UNIT...",
		Description: `Load one or many units in the cluster into systemd, but do not start.

Select units to load by glob matching for units in the current working directory 
or matching the names of previously submitted units.

For units which are not global, load operations are performed synchronously,
which means fleetctl will block until it detects that the unit(s) have
transitioned to a loaded state. This behaviour can be configured with the
respective --block-attempts and --no-block options. Load operations on global
units are always non-blocking.`,
		Run: runLoadUnits,
	}
)

func init() {
	cmdLoadUnits.Flags.BoolVar(&sharedFlags.Sign, "sign", false, "DEPRECATED - this option cannot be used")
	cmdLoadUnits.Flags.IntVar(&sharedFlags.BlockAttempts, "block-attempts", 0, "Wait until the jobs are loaded, performing up to N attempts before giving up. A value of 0 indicates no limit. Does not apply to global units.")
	cmdLoadUnits.Flags.BoolVar(&sharedFlags.NoBlock, "no-block", false, "Do not wait until the jobs have been loaded before exiting. Always the case for global units.")
}

func runLoadUnits(args []string) (exit int) {
	if err := lazyCreateUnits(args); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating units: %v\n", err)
		return 1
	}

	triggered, err := lazyLoadUnits(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading units: %v\n", err)
		return 1
	}

	var loading []string
	for _, u := range triggered {
		if suToGlobal(*u) {
			fmt.Printf("Triggered global unit %s load\n", u.Name)
		} else {
			loading = append(loading, u.Name)
		}
	}

	if !sharedFlags.NoBlock {
		errchan := waitForUnitStates(loading, job.JobStateLoaded, sharedFlags.BlockAttempts, os.Stdout)
		for err := range errchan {
			fmt.Fprintf(os.Stderr, "Error waiting for units: %v\n", err)
			exit = 1
		}
	} else {
		for _, name := range loading {
			fmt.Printf("Triggered unit %s load\n", name)
		}
	}

	return
}
