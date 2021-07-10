package main

import "C"
import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"io/ioutil"
	"math"
	"runtime"
	"sort"
	"strconv"
	"time"

	"github.com/cybercongress/cyberd/merkle"
	"github.com/spf13/cobra"
)

func RunBenchCPUCmd() *cobra.Command {

	cmd := &cobra.Command{
		Use:   "run-bench-cpu <stakesCount> <cidsCount> <dampingFactor> <tolerance> <debug>",
		Short: "Run rank calculation on CPU",
		Args:  cobra.ExactArgs(5),
		RunE: func(cmd *cobra.Command, args []string) error {

			mem := &runtime.MemStats{}
			memUsageOffset := mem.Alloc
			base := uint64(1048576)

			stakesCount, _ := strconv.ParseInt(args[0], 10, 64)
			cidsCount, _ := strconv.ParseInt(args[1], 10, 64)
			// stakesCount := int64(6)
			// cidsCount := int64(21)
			dampingFactor, _ := strconv.ParseFloat(args[2], 64)
			tolerance, _ := strconv.ParseFloat(args[3], 64)
			debug, _ := strconv.ParseBool(args[4])

			fmt.Println("---------------------------------\n")
			fmt.Println("STEP 0: Graph load")
			fmt.Println("Agents: ", stakesCount)
			fmt.Println("CIDs: ", cidsCount)
			fmt.Println("Damping: ", dampingFactor)
			fmt.Println("Tolerance: ", tolerance)

			start := time.Now()

			outLinks := make(map[CidNumber]CidLinks)
			inLinks := make(map[CidNumber]CidLinks)
			// outLinks := make(Links)
			// inLinks := make(Links)
			stakes := make([]uint64, stakesCount)

			readStakesFromBytesFile(&stakes, "./stakes.data")
			readLinksFromBytesFile(&outLinks, "./outLinks.data")
			readLinksFromBytesFile(&inLinks, "./inLinks.data")
			stakes[0] = 0
			stakes[8] = 0

			// outLinks.Put(8, 9, 0)
			// outLinks.Put(9, 10, 0)
			// outLinks.Put(8, 7, 0)
			// outLinks.Put(7, 6, 0)
			// outLinks.Put(20, 19, 0)
			// outLinks.Put(4, 3, 0)
			// outLinks.Put(6, 5, 1)
			// outLinks.Put(20, 5, 1)
			// outLinks.Put(4, 5, 1)
			// outLinks.Put(19, 18, 1)
			// outLinks.Put(17, 18, 1)
			// outLinks.Put(10, 11, 1)
			// outLinks.Put(12, 11, 1)
			// outLinks.Put(0, 14, 1)
			// outLinks.Put(3, 2, 2)
			// outLinks.Put(17, 2, 2)
			// outLinks.Put(16, 2, 2)
			// outLinks.Put(16, 13, 2)
			// outLinks.Put(12, 13, 2)
			// outLinks.Put(15, 13, 2)
			// outLinks.Put(15, 1, 2)
			// outLinks.Put(0, 1, 2)
			// outLinks.Put(14, 13, 3)
			// outLinks.Put(14, 1, 3)
			// outLinks.Put(2, 1, 4)
			// outLinks.Put(11, 18, 5)

			// inLinks.Put(9,  8, 0)
			// inLinks.Put(10, 9,  0)
			// inLinks.Put(7,  8, 0)
			// inLinks.Put(6,  7, 0)
			// inLinks.Put(19, 20,  0)
			// inLinks.Put(3,  4, 0)
			// inLinks.Put(5,  6, 1)
			// inLinks.Put(5,  20, 1)
			// inLinks.Put(5,  4, 1)
			// inLinks.Put(18, 19,  1)
			// inLinks.Put(18, 17,  1)
			// inLinks.Put(11, 10,  1)
			// inLinks.Put(11, 12,  1)
			// inLinks.Put(14, 0,  1)
			// inLinks.Put(2,  3, 2)
			// inLinks.Put(2,  17, 2)
			// inLinks.Put(2,  16, 2)
			// inLinks.Put(13, 16,  2)
			// inLinks.Put(13, 12,  2)
			// inLinks.Put(13, 15,  2)
			// inLinks.Put(1,  15, 2)
			// inLinks.Put(1,  0, 2)
			// inLinks.Put(13, 14,  3)
			// inLinks.Put(1,  14, 3)
			// inLinks.Put(1,  2, 4)
			// inLinks.Put(18, 11, 5)

			// stakes[0] = uint64(4)
			// stakes[1] = uint64(6)
			// stakes[2] = uint64(8)
			// stakes[3] = uint64(12)
			// stakes[4] = uint64(16)
			// stakes[5] = uint64(9)

			fmt.Println("Graph open data: ", "time", time.Since(start))
			runtime.ReadMemStats(mem)
			fmt.Println("-[GO] Memory:", (mem.Alloc-memUsageOffset)/base)

			fmt.Println("---------------------------------\n")
			fmt.Println("STEP 1: Prepare memory")
			rank := make([]float64, cidsCount)
			rankUint := make([]uint64, cidsCount)
			ent := make([]float64, cidsCount)
			entUint := make([]uint64, cidsCount)
			karma := make([]float64, stakesCount)
			karmaUint := make([]uint64, stakesCount)
			defaultRank := (1.0 - dampingFactor) / float64(cidsCount)
			danglingNodesSize := uint64(0)

			runtime.ReadMemStats(mem)
			fmt.Println("-[GO] Memory:", (mem.Alloc-memUsageOffset)/base)

			for i := range rank {
				rank[i] = defaultRank
				if len(inLinks[CidNumber(i)]) == 0 {
					danglingNodesSize++
				}
			}

			innerProductOverSize := defaultRank * (float64(danglingNodesSize) / float64(cidsCount))
			defaultRankWithCorrection := float64(dampingFactor*innerProductOverSize) + defaultRank

			fmt.Println("Default rank", defaultRank)
			fmt.Println("Dangling nodes", danglingNodesSize)
			fmt.Println("Default rank with correction", defaultRankWithCorrection)

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

			entropy(outLinks, inLinks, stakes, ent, cidsCount, dampingFactor)
			karmas(outLinks, inLinks, stakes, rank, ent, karma)

			fmt.Println("Processing", "duration", time.Since(start).String())
			runtime.ReadMemStats(mem)
			fmt.Println("-[GO] Memory:", (mem.Alloc-memUsageOffset)/base)

			fmt.Println("---------------------------------\n")
			fmt.Println("STEP 3: Data and stats")

			start = time.Now()
			rankTreeFloat := merkle.NewTree(sha256.New(), true)
			for _, f64 := range rank {
				rankBytes := make([]byte, 8)
				binary.LittleEndian.PutUint64(rankBytes, math.Float64bits(f64))
				rankTreeFloat.Push(rankBytes)
			}
			rankFloatHash := rankTreeFloat.RootHash()
			fmt.Println("[FLOAT] Rank Tree build", "duration", time.Since(start))
			fmt.Printf("[FLOAT] Rank Tree hash: %x\n", rankFloatHash)

			rankSum := float64(0)
			for _, r := range rank {
				rankSum += r
			}
			fmt.Printf("Ranks sum: %f\n", rankSum)

			start = time.Now()
			for i, f64 := range rank {
				rankUint[i] = uint64(f64 * 1e15)
			}
			fmt.Println("Rank to integer", "duration", time.Since(start).String())

			start = time.Now()
			rankTreeUint := merkle.NewTree(sha256.New(), true)
			for _, r64 := range rankUint {
				rankBytes := make([]byte, 8)
				binary.LittleEndian.PutUint64(rankBytes, r64)
				rankTreeUint.Push(rankBytes)
			}
			rankUintHash := rankTreeUint.RootHash()
			fmt.Println("[UINT] Rank Tree build", "duration", time.Since(start))
			fmt.Printf("[UINT] Rank Tree hash: %x\n", rankUintHash)

			if debug {
				fmt.Println("---------------------------------\n")
				fmt.Println("[FLOAT] Ranks: ", rank)
				fmt.Println("---------------------------------\n")
				fmt.Println("[UINT] Ranks: ", rankUint)
			}

			fmt.Println("---------------------------------\n")

			start = time.Now()
			entropyTreeFloat := merkle.NewTree(sha256.New(), true)
			for _, f64 := range ent {
				entropyBytes := make([]byte, 8)
				binary.LittleEndian.PutUint64(entropyBytes, math.Float64bits(f64))
				entropyTreeFloat.Push(entropyBytes)
			}
			entropyFloatHash := entropyTreeFloat.RootHash()
			fmt.Println("[FLOAT] Entropy tree", "duration", time.Since(start))
			fmt.Printf("[FLOAT] Entropy hash: %x\n", entropyFloatHash)

			entropySum := float64(0)
			for _, r := range ent {
				entropySum += r
			}
			fmt.Printf("Entropy sum: %f\n", entropySum)

			start = time.Now()
			for i, f64 := range ent {
				entUint[i] = uint64(f64 * 1e15)
			}
			fmt.Println("Entropy to integer", "duration", time.Since(start).String())

			start = time.Now()
			entropyTreeUint := merkle.NewTree(sha256.New(), true)
			for _, f64 := range entUint {
				entropyBytes := make([]byte, 8)
				binary.LittleEndian.PutUint64(entropyBytes, f64)
				entropyTreeUint.Push(entropyBytes)
			}
			entropyUintHash := entropyTreeUint.RootHash()
			fmt.Println("[UINT] Entropy tree", "duration", time.Since(start))
			fmt.Printf("[UINT] Entropy hash: %x\n", entropyUintHash)

			if debug {
				fmt.Println("---------------------------------\n")
				fmt.Println("[FLOAT] Entropy: ", ent)
				fmt.Println("---------------------------------\n")
				fmt.Println("[UINT] Entropy: ", entUint)
			}

			fmt.Println("---------------------------------\n")

			start = time.Now()
			karmaTreeFloat := merkle.NewTree(sha256.New(), true)
			for _, f64 := range karma {
				karmaBytes := make([]byte, 8)
				binary.LittleEndian.PutUint64(karmaBytes, math.Float64bits(f64))
				karmaTreeFloat.Push(karmaBytes)
			}
			karmaFloatHash := karmaTreeFloat.RootHash()
			fmt.Println("[FLOAT] Karma tree", "duration", time.Since(start))
			fmt.Printf("[FLOAT] Karma hash: %x\n", karmaFloatHash)

			karmaSum := float64(0)
			for _, r := range karma {
				karmaSum += r
			}
			fmt.Printf("Karma sum: %f\n", karmaSum)

			start = time.Now()
			for i, f64 := range karma {
				karmaUint[i] = uint64(f64 * 1e15)
			}
			fmt.Println("Karma to integer", "duration", time.Since(start).String())

			start = time.Now()
			karmaTreeUint := merkle.NewTree(sha256.New(), true)
			for _, f64 := range karmaUint {
				karmaBytes := make([]byte, 8)
				binary.LittleEndian.PutUint64(karmaBytes, f64)
				karmaTreeUint.Push(karmaBytes)
			}
			karmaUintHash := karmaTreeUint.RootHash()
			fmt.Println("[UINT] Karma tree", "duration", time.Since(start))
			fmt.Printf("[UINT] Karma hash: %x\n", karmaUintHash)

			if debug {
				fmt.Println("---------------------------------\n")
				fmt.Println("[FLOAT] Karma: ", karma)
				fmt.Println("---------------------------------\n")
				fmt.Println("[UINT] Karma: ", karmaUint)
			}

			fmt.Println("CPU------------------------------\n")

			return nil
		},
	}

	return cmd
}

func karmas(outLinks Links, inLinks Links, stakes []uint64, rank []float64, entropy []float64, karma []float64) {
	for from := range outLinks {
		stake := getOverallOutLinksStake(outLinks, stakes, from)
		for to := range outLinks[from] {
			users := outLinks[from][to]
			for user := range users {
				w := float64(stakes[user]) / float64(stake)
				if math.IsNaN(w) {
					w = float64(0)
				}
				karma[user] += w * float64(rank[from]*entropy[from])
			}
		}
	}
}

func entropy(outLinks Links, inLinks Links, stakes []uint64, entropy []float64, cidsCount int64, dampingFactor float64) {
	swd := make([]float64, cidsCount)
	sumswd := make([]float64, cidsCount)
	for i, _ := range swd {
		swd[i] = dampingFactor*float64(
			getOverallOutLinksStake(inLinks, stakes, CidNumber(i))) + (1-dampingFactor)*float64(
			getOverallOutLinksStake(outLinks, stakes, CidNumber(i)))
	}

	for i, _ := range sumswd {
		for to := range inLinks[CidNumber(i)] {
			sumswd[i] += dampingFactor * swd[to]
		}
		for to := range outLinks[CidNumber(i)] {
			sumswd[i] += (1 - dampingFactor) * swd[to]
		}
	}

	for i, _ := range entropy {
		if swd[i] == 0 {
			continue
		}
		for to := range inLinks[CidNumber(i)] {
			if sumswd[to] == 0 {
				continue
			}
			entropy[i] += math.Abs(-swd[i] / sumswd[to] * math.Log2(swd[i]/sumswd[to]))
		}
		for to := range outLinks[CidNumber(i)] {
			if sumswd[to] == 0 {
				continue
			}
			entropy[i] += math.Abs(-swd[i] / sumswd[to] * math.Log2(swd[i]/sumswd[to]))
		}
	}
}

func step(inLinks Links, outLinks Links, stakes []uint64, defaultRankWithCorrection float64, dampingFactor float64, prevrank []float64) []float64 {

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
				if linkStake == 0 || jCidOutStake == 0 {
					continue
				}
				weight := float64(linkStake) / float64(jCidOutStake)
				// if math.IsNaN(weight) {
				// 	weight = float64(0)
				// }
				ksum = prevrank[j]*weight + ksum //force no-fma here by explicit conversion
			}
			rank[cid] = ksum*dampingFactor + defaultRankWithCorrection //force no-fma here by explicit conversion
		}
	}

	return rank
}

func getOverallLinkStake(outLinks Links, stakes []uint64, from CidNumber, to CidNumber) uint64 {

	stake := uint64(0)
	users := outLinks[from][to]
	for user := range users {
		stake += stakes[user]
	}
	return stake
}

func getOverallOutLinksStake(outLinks Links, stakes []uint64, from CidNumber) uint64 {

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

func readStakesFromBytesFile(stakes *[]uint64, fileName string) {
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
