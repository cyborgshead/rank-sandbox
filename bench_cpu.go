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
	"sort"
	"strconv"
	"time"

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

			fmt.Println("---------------------------------")

			start := time.Now()

			//outLinks := make(map[CidNumber]CidLinks)
			//inLinks := make(map[CidNumber]CidLinks)
			outLinks := make(Links)
			inLinks := make(Links)
			stakes := make([]uint64, stakesCount)

			//readStakesFromBytesFile(&stakes, "./stakes.data")
			//readLinksFromBytesFile(&outLinks, "./outLinks.data")
			//readLinksFromBytesFile(&inLinks, "./inLinks.data")

			outLinks.Put(0,1,1)
			outLinks.Put(0,3,1)
			outLinks.Put(1,4,1)
			outLinks.Put(2,8,2)
			outLinks.Put(3,8,2)
			outLinks.Put(4,9,2)
			outLinks.Put(5,2,3)
			outLinks.Put(6,7,3)
			outLinks.Put(7,9,3)
			outLinks.Put(8,5,0)
			outLinks.Put(9,6,0)

			outLinks.Put(0,5,4)
			outLinks.Put(0,6,4)
			outLinks.Put(1,7,4)
			outLinks.Put(2,5,5)
			outLinks.Put(3,1,5)
			outLinks.Put(4,5,5)
			outLinks.Put(5,6,6)
			outLinks.Put(6,9,6)
			outLinks.Put(7,0,6)
			outLinks.Put(8,0,0)
			outLinks.Put(9,4,0)

			outLinks.Put(0,4,7)
			outLinks.Put(0,9,7)
			outLinks.Put(1,8,7)
			outLinks.Put(2,4,8)
			outLinks.Put(3,8,8)
			outLinks.Put(4,7,8)
			outLinks.Put(5,1,9)
			outLinks.Put(6,3,9)
			outLinks.Put(7,2,9)
			outLinks.Put(8,1,0)
			outLinks.Put(9,3,0)

			inLinks.Put(1,0,1)
			inLinks.Put(3,0,1)
			inLinks.Put(4,1,1)
			inLinks.Put(8,2,2)
			inLinks.Put(8,3,2)
			inLinks.Put(9,4,2)
			inLinks.Put(2,5,3)
			inLinks.Put(7,6,3)
			inLinks.Put(9,7,3)
			inLinks.Put(5,8,0)
			inLinks.Put(6,9,0)

			inLinks.Put(5,0,4)
			inLinks.Put(6,0,4)
			inLinks.Put(7,1,4)
			inLinks.Put(5,2,5)
			inLinks.Put(1,3,5)
			inLinks.Put(5,4,5)
			inLinks.Put(6,5,6)
			inLinks.Put(9,6,6)
			inLinks.Put(0,7,6)
			inLinks.Put(0,8,0)
			inLinks.Put(4,9,0)

			inLinks.Put(4,0,7)
			inLinks.Put(9,0,7)
			inLinks.Put(8,1,7)
			inLinks.Put(4,2,8)
			inLinks.Put(8,3,8)
			inLinks.Put(7,4,8)
			inLinks.Put(1,5,9)
			inLinks.Put(3,6,9)
			inLinks.Put(2,7,9)
			inLinks.Put(1,8,0)
			inLinks.Put(3,9,0)

			fmt.Println("outLinks: ", outLinks)

			//for i := 0; i < int(stakesCount); i++ {
			//	stakes = append(stakes, uint64((i+1)*200))
			//}
			for i, _ := range stakes {
				stakes[i] = uint64((i+1)*200)
			}

			fmt.Println("Graph open data: ", "time", time.Since(start))
			fmt.Println("---------------------------------")

			rank := make([]float64, cidsCount)
			rankUint := make([]uint64, cidsCount)
			ent := make([]float64, cidsCount)
			entUint := make([]uint64, cidsCount)
			light := make([]float64, cidsCount)
			lightUint := make([]uint64, cidsCount)
			karma := make([]float64, cidsCount)
			karmaUint := make([]uint64, cidsCount)
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

			rs := float64(0)
			for _, r := range rank {
				rs += r
			}
			fmt.Printf("RanksSum: %f\n", rs)

			start = time.Now()
			for i, f64 := range rank {
				rankUint[i] = uint64(f64*1e10)
			}
			fmt.Println("Rank converting to uint: ", "time", time.Since(start))
			fmt.Println("Ranks []float64: ", rank)
			fmt.Println("Ranks []uint64: ", rankUint)

			fmt.Println("---------------------------------")

			start = time.Now()

			e := entropy(outLinks, inLinks, stakes, ent)
			fmt.Printf("EntropySum: %f\n", e)

			fmt.Println("Entropy calculation: ", "time", time.Since(start))

			start = time.Now()
			for i,e64 := range ent {
				entUint[i] = uint64(e64*1e10)
			}
			fmt.Println("Entropy converting to uint: ", "time", time.Since(start))
			fmt.Println("Entropy []float64: ", ent)
			fmt.Println("Entropy []uint64: ", entUint)

			fmt.Println("---------------------------------")

			start = time.Now()
			for i, _ := range rank {
				light[i] = rank[i]*ent[i]
			}
			fmt.Println("Light calculation: ", "time", time.Since(start))

			start = time.Now()
			for i, l64 := range light {
				lightUint[i] = uint64(l64*1e10)
			}
			fmt.Println("Light converting to uint: ", "time", time.Since(start))

			fmt.Println("Light []float64: ", light)
			fmt.Println("Light []uint64: ", lightUint)

			fmt.Println("---------------------------------")

			start = time.Now()
			k := karmaCalc(outLinks, inLinks, stakes, light, karma)
			fmt.Println("Karma calculation: ", "time", time.Since(start))

			start = time.Now()
			for i, k64 := range karma {
				karmaUint[i] = uint64(k64*1e10)
			}
			fmt.Println("Karma converting to uint: ", "time", time.Since(start))

			fmt.Println("KarmaSum []float64: ", k)
			fmt.Println("Karma []float64: ", karma)
			fmt.Println("Karma []uint64: ", karmaUint)

			fmt.Println("---------------------------------")

			fmt.Println("Stake []uint64: ", stakes)

			fmt.Println("---------------------------------")

			start = time.Now()
			rankTree := merkle.NewTree(sha256.New(), true)
			for _, r64 := range rankUint {
				rankBytes := make([]byte, 8)
				binary.LittleEndian.PutUint64(rankBytes, r64)
				rankTree.Push(rankBytes)
			}
			rhash := rankTree.RootHash()
			fmt.Println("Rank constructing merkle tree: ", "time", time.Since(start))
			fmt.Printf("Rank merkle root hash: %x\n", rhash)

			fmt.Println("---------------------------------")

			start = time.Now()
			entropyTree := merkle.NewTree(sha256.New(), true)
			for _, e64 := range entUint {
				entropyBytes := make([]byte, 8)
				binary.LittleEndian.PutUint64(entropyBytes, e64)
				entropyTree.Push(entropyBytes)
			}
			ehash := entropyTree.RootHash()
			fmt.Println("Entropy constructing merkle tree: ", "time", time.Since(start))
			fmt.Printf("Entropy merkle root hash: %x\n", ehash)

			fmt.Println("---------------------------------")

			start = time.Now()
			lightTree := merkle.NewTree(sha256.New(), true)
			for _, l64 := range lightUint {
				lightBytes := make([]byte, 8)
				binary.LittleEndian.PutUint64(lightBytes, l64)
				lightTree.Push(lightBytes)
			}
			lhash := lightTree.RootHash()
			fmt.Println("Light constructing merkle tree: ", "time", time.Since(start))
			fmt.Printf("Light merkle root hash: %x\n", lhash)

			fmt.Println("---------------------------------")

			start = time.Now()
			karmaTree := merkle.NewTree(sha256.New(), true)
			for _, k64 := range karmaUint {
				karmaBytes := make([]byte, 8)
				binary.LittleEndian.PutUint64(karmaBytes, k64)
				karmaTree.Push(karmaBytes)
			}
			khash := karmaTree.RootHash()
			fmt.Println("Karma constructing merkle tree: ", "time", time.Since(start))
			fmt.Printf("Karma merkle root hash: %x\n", khash)

			fmt.Println("---------------------------------")
			fmt.Println("Prepare mocked data for CPU-GPU results cross check validation")
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


func karmaCalc(outLinks Links, inLinks Links, stakes []uint64, light []float64, karma []float64) (float64) {
	k := float64(0)

	for from := range outLinks {
		outStake := getOverallOutLinksStake(outLinks, stakes, from)
		inStake := getOverallOutLinksStake(inLinks, stakes, from)
		//fmt.Println("FROM:",from,"OUT/IN ->", outStake,"/",inStake)
		ois := outStake + inStake
		for to := range outLinks[from] {
			users := outLinks[from][to]
			for user := range users {
				w := float64(stakes[user])/float64(ois)
				karma[user] += w*float64(light[from])
				k += w*float64(light[from])
				//fmt.Println("USER:", user,"|",from,"->",to,"| S:", stakes[user], "| TLS:", ois, "| K:", w*float64(light[from]))
			}
		}
		//fmt.Println("--------\n")
	}

	return k
}

func entropy(outLinks Links, inLinks Links, stakes []uint64, ent []float64) (float64) {
	e := float64(0)

	for from := range outLinks {
		outStake := getOverallOutLinksStake(outLinks, stakes, from)
		inStake := getOverallOutLinksStake(inLinks, stakes, from)
		ois := outStake + inStake
		for to := range outLinks[from] {
			linkStake := getOverallLinkStake(outLinks, stakes, from, to)
			w := float64(linkStake) / float64(ois)
			e -= w*math.Log2(w)
			ent[from] -= w*math.Log2(w)
			//fmt.Println("LINK:", from,"->",to,"| LS:", linkStake, "| TLS:", ois, "| W:", w,"| E:", -w*math.Log(w))
		}
		//fmt.Println("--------\n")
	}

	return e
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
				weight := float64(linkStake) / float64(jCidOutStake)
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

