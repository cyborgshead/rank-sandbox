package main

import "C"
import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"io/ioutil"
	"math"
	"sort"
	"strconv"
	"time"
	"crypto/sha256"

	"github.com/cybercongress/cyberd/merkle"
	"github.com/spf13/cobra"
)

func RunBenchCPUCmd() *cobra.Command {

	cmd := &cobra.Command{
		Use:   "run-bench-cpu <stakesCount> <cidsCount> <dampingFactor> <tolerance>",
		Short: "Run rank calculation on CPU",
		Args:  cobra.ExactArgs(4),
		RunE: func(cmd *cobra.Command, args []string) error {

			stakesCount, _ := strconv.ParseInt(args[0], 10, 64)
			cidsCount, _ := strconv.ParseInt(args[1], 10, 64)
			dampingFactor, _ := strconv.ParseFloat(args[2], 64)
			tolerance, _ := strconv.ParseFloat(args[3], 64)

			fmt.Println("Agents: ", stakesCount)
			fmt.Println("CIDs: ", cidsCount)
			fmt.Println("Damping: ", dampingFactor)
			fmt.Println("Tolerance: ", tolerance)

			start := time.Now()

			outLinks := make(map[CidNumber]CidLinks)
			inLinks := make(map[CidNumber]CidLinks)
			stakes := make(map[AccNumber]uint64)

			readStakesFromBytesFile(&stakes, "./stakes.data")
			readLinksFromBytesFile(&outLinks, "./outLinks.data")
			readLinksFromBytesFile(&inLinks, "./inLinks.data")
			fmt.Println("Graph open data: ", "time", time.Since(start))

			rank := make([]float64, cidsCount)
			defaultRank := (1.0 - dampingFactor) / float64(cidsCount)
			danglingNodesSize := uint64(0)

			for i := range rank {
				rank[i] = defaultRank
				if len(inLinks[CidNumber(i)]) == 0 {
					danglingNodesSize++
				}
			}

			innerProductOverSize := defaultRank * (float64(danglingNodesSize) / float64(cidsCount))
			defaultRankWithCorrection := float64(dampingFactor*innerProductOverSize) + defaultRank

			fmt.Println("Rank calculation", "defaultRank", defaultRank)
			fmt.Println("Rank calculation", "danglingNodesSize", danglingNodesSize)
			fmt.Println("Rank calculation", "defaultRankWithCorrection", defaultRankWithCorrection)

			change := tolerance + 1

			start = time.Now()
			steps := 0
			prevrank := make([]float64, 0)
			prevrank = append(prevrank, rank...)
			for change > tolerance {
				rank = step(inLinks, outLinks, stakes, defaultRankWithCorrection, dampingFactor, prevrank)
				change = calculateChange(prevrank, rank)
				prevrank = rank
				steps++
			}
			fmt.Println("Rank calculation", "time", time.Since(start))


			start = time.Now()
			merkleTree := merkle.NewTree(sha256.New(), true)
			for _, f64 := range rank {
				rankBytes := make([]byte, 8)
				binary.LittleEndian.PutUint64(rankBytes, math.Float64bits(f64))
				merkleTree.Push(rankBytes)
			}
			hash := merkleTree.RootHash()
			fmt.Println("Rank constructing merkle tree: ", "time", time.Since(start))
			fmt.Printf("Rank merkle root hash: %x\n", hash)

			return nil
		},
	}

	return cmd
}

func step(inLinks Links, outLinks Links, stakes map[AccNumber]uint64, defaultRankWithCorrection float64, dampingFactor float64, prevrank []float64) []float64 {

	rank := append(make([]float64, 0, len(prevrank)), prevrank...)

	for cid := range inLinks {
		_, sortedCids, ok := GetSortedInLinks(inLinks, cid)

		if !ok {
			continue
		} else {
			ksum := float64(0)
			for _, j := range sortedCids {
				linkStake := getOverallLinkStake(outLinks, stakes, j, cid)
				jCidOutStake := getOverallOutLinksStake(outLinks, stakes, j)
				weight := float64(linkStake) / float64(jCidOutStake)
				ksum = prevrank[j]*weight + ksum //force no-fma here by explicit conversion
			}
			rank[cid] = ksum*dampingFactor + defaultRankWithCorrection //force no-fma here by explicit conversion
		}
	}

	return rank
}

func getOverallLinkStake(outLinks Links, stakes map[AccNumber]uint64, from CidNumber, to CidNumber) uint64 {

	stake := uint64(0)
	users := outLinks[from][to]
	for user := range users {
		stake += stakes[user]
	}
	return stake
}

func getOverallOutLinksStake(outLinks Links, stakes map[AccNumber]uint64, from CidNumber) uint64 {

	stake := uint64(0)
	for to := range outLinks[from] {
		stake += getOverallLinkStake(outLinks, stakes, from, to)
	}
	return stake
}

func calculateChange(prevrank, rank []float64) float64 {

	maxDiff := 0.0
	diff := 0.0
	for i, pForI := range prevrank {
		if pForI > rank[i] {
			diff = pForI - rank[i]
		} else {
			diff = rank[i] - pForI
		}
		if diff > maxDiff {
			maxDiff = diff
		}
	}

	return maxDiff
}

func readLinksFromBytesFile(links *map[CidNumber]CidLinks, fileName string) {
	var network bytes.Buffer

	data, err := ioutil.ReadFile(fileName)
	if err != nil {
		fmt.Printf("error on read links from file  err: %v", err)
	}
	n, err := network.Write(data)
	if err != nil {
		fmt.Printf("error on read links from file n = %v err: %v", n, err)
	}

	dec := gob.NewDecoder(&network)
	err = dec.Decode(links)
	if err != nil {
		fmt.Printf("Decode error:", err)
	}

}

func readStakesFromBytesFile(stakes *map[AccNumber]uint64, fileName string) {
	var network bytes.Buffer

	data, err := ioutil.ReadFile(fileName)
	if err != nil {
		fmt.Printf("error on read stakes from file  err: %v", err)
	}
	n, err := network.Write(data)
	if err != nil {
		fmt.Printf("error on read stakes from file n = %v err: %v", n, err)
	}

	dec := gob.NewDecoder(&network)
	err = dec.Decode(stakes)
	if err != nil {
		fmt.Printf("Decode error:", err)
	}

}

func GetSortedInLinks(inLinks Links, cid CidNumber) (CidLinks, []CidNumber, bool) {
	links := inLinks[cid]

	if len(links) == 0 {
		return nil, nil, false
	}

	numbers := make([]CidNumber, 0, len(links))
	for num := range links {
		numbers = append(numbers, num)
	}

	sort.Slice(numbers, func(i, j int) bool { return numbers[i] < numbers[j] })

	return links, numbers, true
}

