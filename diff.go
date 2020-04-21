package main

import (
	"fmt"

	"github.com/r3labs/diff"
	"github.com/spf13/cobra"
)

func RunDiffCmd() *cobra.Command {

	cmd := &cobra.Command{
		Use:   "run-diff",
		Short: "Run diff of two graphs",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {


			outLinksA := make(map[CidNumber]CidLinks)
			inLinksA := make(map[CidNumber]CidLinks)
			stakesA := make(map[AccNumber]uint64)

			readStakesFromBytesFile(&stakesA, "./diffB/stakes.data")
			readLinksFromBytesFile(&outLinksA, "./diffB/outLinks.data")
			readLinksFromBytesFile(&inLinksA, "./diffB/inLinks.data")

			outLinksB := make(map[CidNumber]CidLinks)
			inLinksB := make(map[CidNumber]CidLinks)
			stakesB := make(map[AccNumber]uint64)

			readStakesFromBytesFile(&stakesB, "./diffC/stakes.data")
			readLinksFromBytesFile(&outLinksB, "./diffC/outLinks.data")
			readLinksFromBytesFile(&inLinksB, "./diffC/inLinks.data")

			changelogStakes, _ := diff.Diff(stakesA, stakesB)
			fmt.Println("DIFF:", "stakes", changelogStakes)

			changelogInLinks, _ := diff.Diff(inLinksA, inLinksB)
			fmt.Println("DIFF:", "inLinks", changelogInLinks)

			changelogOutLinks, _ := diff.Diff(outLinksA, outLinksB)
			fmt.Println("DIFF:", "outLinks", changelogOutLinks)

			return nil
		},
	}

	return cmd
}

