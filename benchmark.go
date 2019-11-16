// +build cuda

package main

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"github.com/spf13/cobra"
	"math"
	"math/rand"
	"sort"
	"strconv"
	"time"
	"github.com/cybercongress/cyberd/merkle"
)

/*
#cgo CFLAGS: -I/usr/lib/
#cgo LDFLAGS: -L/usr/local/cuda/lib64 -lcbdrank -lcudart
#include "cbdrank.h"
*/
import "C"

type CidNumber uint64
type AccNumber uint64
type Links map[CidNumber]CidLinks
type CidLinks map[CidNumber]map[AccNumber]struct{}

func RandSeed() {
	rand.Seed(time.Now().UnixNano())
}


func RunBenchCmd() *cobra.Command {

	cmd := &cobra.Command{
		Use:   "run-bench <stakesCount> <linksPerAgent> <cidsCount> <dampingFactor> <tolerance>",
		Short: "Run cyberrank with different graph and algorithm params",
		Args:  cobra.ExactArgs(5),
		RunE: func(cmd *cobra.Command, args []string) error {

			// RandSeed()
			rand.Seed(42)

			stakesCount, _ := strconv.ParseInt(args[0], 10, 64)
			linksPerAgent, _ := strconv.ParseInt(args[1], 10 ,64)
			cidsCount, _ := strconv.ParseInt(args[2], 10, 64)
			dampingFactor, _ := strconv.ParseFloat(args[3], 64)
			tolerance, _ := strconv.ParseFloat(args[4], 64)

			fmt.Println("Agents: ", stakesCount)
			fmt.Println("Links per agent: ", linksPerAgent)
			fmt.Println("CIDs: ", cidsCount)
			fmt.Println("Damping: ", dampingFactor)
			fmt.Println("Tolerance: ", tolerance)

			outLinks := make(Links)
			inLinks := make(Links)

			start := time.Now()
			for i := 0; i < int(stakesCount); i++ {
				for i := 0; i < int(linksPerAgent); i++ {
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


			linksCount := uint64(0)
			rank := make([]float64, cidsCount)
			inLinksCount := make([]uint32, cidsCount)
			outLinksCount := make([]uint32, cidsCount)
			inLinksOuts := make([]uint64, 0)
			inLinksUsers := make([]uint64, 0)
			outLinksUsers := make([]uint64, 0)

			start = time.Now()
			stakes := make([]uint64, stakesCount)
			for acc := range stakes {
				stakes[acc] = uint64(rand.Intn(1000000000) + 100000)
			}
			fmt.Println("Stakes generation for agents", "time", time.Since(start))


			start = time.Now()
			for i := int64(0); i < cidsCount; i++ {

				if inLinks, sortedCids, ok := GetSortedInLinks(inLinks, CidNumber(i)); ok {
					for _, cid := range sortedCids {
						inLinksCount[i] += uint32(len(inLinks[cid]))
						for acc := range inLinks[cid] {
							inLinksOuts = append(inLinksOuts, uint64(cid))
							inLinksUsers = append(inLinksUsers, uint64(acc))
						}
					}
					linksCount += uint64(inLinksCount[i])
				}

				if outLinks, ok := outLinks[CidNumber(i)]; ok {
					for _, accs := range outLinks {
						outLinksCount[i] += uint32(len(accs))
						for acc := range accs {
							outLinksUsers = append(outLinksUsers, uint64(acc))
						}
					}
				}
			}
			fmt.Println("Links amount", linksCount)
			fmt.Println("Data preparation", "time", time.Since(start))

			outLinks = nil
			inLinks = nil

			cStakes := (*C.ulong)(&stakes[0])

			cStakesSize := C.ulong(len(stakes))
			cCidsSize := C.ulong(len(inLinksCount))
			cLinksSize := C.ulong(len(inLinksOuts))

			cInLinksCount := (*C.uint)(&inLinksCount[0])
			cOutLinksCount := (*C.uint)(&outLinksCount[0])

			cInLinksOuts := (*C.ulong)(&inLinksOuts[0])
			cInLinksUsers := (*C.ulong)(&inLinksUsers[0])
			cOutLinksUsers := (*C.ulong)(&outLinksUsers[0])

			cDampingFactor := C.double(dampingFactor)
			cTolerance := C.double(tolerance)

			start = time.Now()
			cRank := (*C.double)(&rank[0])
			C.calculate_rank(
				cStakes, cStakesSize, cCidsSize, cLinksSize,
				cInLinksCount, cOutLinksCount,
				cInLinksOuts, cInLinksUsers, cOutLinksUsers,
				cRank, cDampingFactor, cTolerance,
			)
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


