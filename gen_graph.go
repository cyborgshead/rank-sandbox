package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io/ioutil"
	"math/rand"
	"strconv"
	"time"
	"github.com/spf13/cobra"
)

type CidNumber uint64
type AccNumber uint64
type Links map[CidNumber]CidLinks
type CidLinks map[CidNumber]map[AccNumber]struct{}

func RunGenGraphCmd() *cobra.Command {

	cmd := &cobra.Command{
		Use:   "gen-graph <stakesCount> <linksPerAgent> <cidsCount> <randSeed>",
		Short: "Generates graph with provided params and random seed",
		Args:  cobra.ExactArgs(4),
		RunE: func(cmd *cobra.Command, args []string) error {

			stakesCount, _ := strconv.ParseInt(args[0], 10, 64)
			linksPerAgent, _ := strconv.ParseInt(args[1], 10, 64)
			cidsCount, _ := strconv.ParseInt(args[2], 10, 64)
			randSeed, _ := strconv.ParseInt(args[3], 10, 64)

			if (randSeed == 0) { randSeed = time.Now().UnixNano() }
			rand.Seed(randSeed)

			fmt.Println("Agents: ", stakesCount)
			fmt.Println("Links per agent: ", linksPerAgent)
			fmt.Println("CIDs: ", cidsCount)
			fmt.Println("RandSeed: ", randSeed)

			outLinks := make(Links)
			inLinks := make(Links)

			start := time.Now()
			for i := 0; i < int(stakesCount); i++ {
				for j := 0; j < int(linksPerAgent); j++ {
					src := rand.Int63n(cidsCount)
					dst := rand.Int63n(cidsCount)
					if src != dst {
						outLinks.Put(CidNumber(src), CidNumber(dst), AccNumber(uint64(i)))
						inLinks.Put(CidNumber(dst), CidNumber(src), AccNumber(uint64(i)))
					}
				}
			}
			fmt.Println("Graph generation", "time", time.Since(start))

			start = time.Now()
			fixed := int(0)
			for i := 0; i < int(cidsCount); i++ {
				if _, ok := outLinks[CidNumber(i)]; !ok {
					dst := rand.Int63n(cidsCount)
					agent := rand.Int63n(stakesCount)
					outLinks.Put(CidNumber(i), CidNumber(dst), AccNumber(agent))
					inLinks.Put(CidNumber(dst), CidNumber(i), AccNumber(agent))
					fixed++
				}
			}
			fmt.Println("Added links: ", fixed)
			fmt.Println("Graph check and filling", "time", time.Since(start))

			start = time.Now()
			stakes := make([]uint64, stakesCount)
			for i, _ := range stakes {
				stakes[i] = uint64(rand.Intn(1000000000) + 100000)
			}
			fmt.Println("Stakes generation for agents", "time", time.Since(start))

			start = time.Now()
			saveStakesToBytesFile(&stakes, "./stakes.data")
			saveLinksToBytesFile(&outLinks, "./outLinks.data")
			saveLinksToBytesFile(&inLinks, "./inLinks.data")
			fmt.Println("OutLinks, InLinks and Stakes saved in: ", "time", time.Since(start))

			return nil
		},
	}

	return cmd
}

func (links Links) Put(from CidNumber, to CidNumber, acc AccNumber) {
	cidLinks := links[from]
	if cidLinks == nil {
		cidLinks = make(CidLinks)
	}
	users := cidLinks[to]
	if users == nil {
		users = make(map[AccNumber]struct{})
	}
	users[acc] = struct{}{}
	cidLinks[to] = users
	links[from] = cidLinks
}

func saveLinksToBytesFile(links *Links, fileName string) {
	var network bytes.Buffer
	enc := gob.NewEncoder(&network)
	err := enc.Encode(links)
	if err != nil {
		fmt.Printf("encode error:", err)
	}
	err = ioutil.WriteFile(fileName, network.Bytes(), 0644)
	if err != nil {
		fmt.Printf("error on write links to file  err: %v", err)
	}

}

func saveStakesToBytesFile(stakes *[]uint64, fileName string) {
	var network bytes.Buffer
	enc := gob.NewEncoder(&network)
	err := enc.Encode(stakes)
	if err != nil {
		fmt.Printf("encode error:", err)
	}
	err = ioutil.WriteFile(fileName, network.Bytes(), 0644)
	if err != nil {
		fmt.Printf("error on write stakes to file  err: %v", err)
	}

}
