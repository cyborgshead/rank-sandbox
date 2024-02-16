package main

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"github.com/cybercongress/cyberd/merkle"
	"github.com/spf13/cobra"
	"math"
	"math/rand"
	"strconv"
	"time"
)

func RunTensorGenCPUCmd() *cobra.Command {

	//./cyberrank gen-graph 1000 20000 10000 42
	//./cyberrank run-tensor-cpu 1000 10000 39996042

	cmd := &cobra.Command{
		Use:   "run-tensor-gen-cpu <stakesCount> <linksPerAgent> <cidsCount> <randSeed>",
		Short: "Run tensor calculation on CPU with with generarated graph ",
		Args:  cobra.ExactArgs(4),
		RunE: func(cmd *cobra.Command, args []string) error {

			stakesCount, _ := strconv.ParseInt(args[0], 10, 64)
			linksPerAgent, _ := strconv.ParseInt(args[1], 10, 64)
			cidsCount, _ := strconv.ParseInt(args[2], 10, 64)
			randSeed, _ := strconv.ParseInt(args[3], 10, 64)

			if randSeed == 0 {
				randSeed = time.Now().UnixNano()
			}
			rand.Seed(randSeed)

			fmt.Println("Agents: ", stakesCount)
			fmt.Println("Links per agent: ", linksPerAgent)
			fmt.Println("CIDs: ", cidsCount)
			fmt.Println("RandSeed: ", randSeed)

			outLinks := make(Links)
			inLinks := make(Links)

			start := time.Now()
			linksCount := 0
			for i := 0; i < int(stakesCount); i++ {
				for j := 0; j < int(linksPerAgent); j++ {
					dst := int64(0)
					if i%2 == 0 {
						dst = rand.Int63n(cidsCount / 100)
					} else {
						dst = rand.Int63n(cidsCount)
					}

					src := int64(0)
					if i%2 == 0 {
						src = rand.Int63n(cidsCount / 100)
					} else {
						src = rand.Int63n(cidsCount)
					}
					//src := rand.Int63n(cidsCount)
					//dst := rand.Int63n(cidsCount)
					//println("src", src, "dst", dst, "neuron", i)
					if src != dst {
						outLinks.Put(CidNumber(src), CidNumber(dst), AccNumber(uint64(i)))
						inLinks.Put(CidNumber(dst), CidNumber(src), AccNumber(uint64(i)))
						linksCount += 1
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
					//println("src", i, "dst", dst, "neuron", agent)
					outLinks.Put(CidNumber(i), CidNumber(dst), AccNumber(agent))
					inLinks.Put(CidNumber(dst), CidNumber(i), AccNumber(agent))
					linksCount += 1
					fixed++
				}
			}
			fmt.Println("Added links: ", fixed)
			fmt.Println("Total links: ", linksCount)
			fmt.Println("Graph check and filling", "time", time.Since(start))

			start = time.Now()
			stakes := make([]uint64, stakesCount)
			for acc := range stakes {
				stakes[acc] = uint64(rand.Intn(1000000000) + 100000)
			}
			fmt.Println("Stakes generation for agents", "time", time.Since(start))
			println("len stakes debug", len(stakes))

			printMapSizeInGB(outLinks)
			printMapSizeInGB(inLinks)

			//for outLinksKey, outLinksValue := range outLinks {
			//	fmt.Println("OutLinks", outLinksKey, outLinksValue)
			//}
			//for inLinksKey, inLinksValue := range inLinks {
			//	fmt.Println("inLinks", inLinksKey, inLinksValue)
			//}

			//start := time.Now()
			//
			//outLinks := make(map[CidNumber]CidLinks)
			//inLinks := make(map[CidNumber]CidLinks)
			//stakes := make(map[AccNumber]uint64)
			//
			//readStakesFromBytesFile(&stakes, "./stakes.data")
			//readLinksFromBytesFile(&outLinks, "./outLinks.data")
			//readLinksFromBytesFile(&inLinks, "./inLinks.data")
			//fmt.Println("Graph open data: ", "time", time.Since(start))

			//linksCount = 0
			//rank := make([]float64, cidsCount)
			inLinksCount := make([]uint32, cidsCount)
			outLinksCount := make([]uint32, cidsCount)
			inLinksOuts := make([]uint64, linksCount)
			inLinksUsers := make([]uint64, linksCount)
			outLinksUsers := make([]uint64, linksCount)

			start = time.Now()
			var pointer1 uint32 = 0
			var pointer2 uint32 = 0
			for i := int64(0); i < cidsCount; i++ {

				if inLinks, sortedCids, ok := GetSortedInLinks(inLinks, CidNumber(i)); ok {
					for _, cid := range sortedCids {
						inLinksCount[i] += uint32(len(inLinks[cid]))
						for acc := range inLinks[cid] {
							//inLinksOuts = append(inLinksOuts, uint64(cid))
							//inLinksUsers = append(inLinksUsers, uint64(acc))
							inLinksOuts[pointer1] = uint64(cid)
							inLinksUsers[pointer1] = uint64(acc)
							pointer1++
						}
					}
					//linksCount += uint64(inLinksCount[i])
				}

				if outLinks, ok := outLinks[CidNumber(i)]; ok {
					for _, accs := range outLinks {
						outLinksCount[i] += uint32(len(accs))
						for acc := range accs {
							//outLinksUsers = append(outLinksUsers, uint64(acc))
							outLinksUsers[pointer2] = uint64(acc)
							pointer2++
						}
					}
				}
			}
			fmt.Println("Links amount", linksCount)
			fmt.Println("Data preparation", "time", time.Since(start))

			//fmt.Println("outLinksCount", outLinksCount)
			//fmt.Println("outLinksUsers", outLinksUsers)

			//fmt.Println("inLinksOuts", inLinksOuts)
			//fmt.Println("inLinksCount", inLinksCount)
			//fmt.Println("inLinksUsers", inLinksUsers)

			//outLinks = nil
			//inLinks = nil

			stakesNew := make(map[AccNumber]uint64)
			for i, stake := range stakes {
				stakesNew[AccNumber(i)] = stake
			}

			trust := tensor_cpu(inLinksCount, inLinksOuts, inLinksUsers, stakesNew)

			start = time.Now()
			merkleTree := merkle.NewTree(sha256.New(), true)
			for _, f64 := range trust {
				rankBytes := make([]byte, 8)
				binary.LittleEndian.PutUint64(rankBytes, math.Float64bits(f64))
				merkleTree.Push(rankBytes)
			}
			hash := merkleTree.RootHash()
			fmt.Println("Trust constructing merkle tree: ", "time", time.Since(start))
			fmt.Printf("Trust merkle root hash: %x\n", hash)

			return nil
		},
	}

	return cmd

}
