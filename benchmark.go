// +build cuda

package main

import (
	"fmt"
	"github.com/spf13/cobra"
	"math/rand"
	"sort"
	"strconv"
	"time"
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
		Use:   "run-bench <stakesCount> <cidsCount> <tolerance> <cidsCount>",
		Short: "Run cyberrank with different graph and algorithm params",
		Args:  cobra.ExactArgs(4),
		RunE: func(cmd *cobra.Command, args []string) error {

			RandSeed()

			stakesCount, _ := strconv.ParseInt(args[0], 10, 64)
			cidsCount, _ := strconv.ParseInt(args[1], 10, 64)
			dampingFactor, _ := strconv.ParseFloat(args[2], 64)
			tolerance, _ := strconv.ParseFloat(args[3], 64)

			start := time.Now()

			fmt.Println("Agents: ", stakesCount)
			fmt.Println("CIDs: ", cidsCount)
			fmt.Println("Damping: ", dampingFactor)
			fmt.Println("Tolerance: ", tolerance)

			cidShuf := make([]CidNumber, cidsCount)
			for index, _ := range cidShuf {
				cidShuf[index] = CidNumber(index)
			}

			cidSrc := make([]CidNumber, cidsCount)
			for index, _ := range cidSrc {
				cidSrc[index] = CidNumber(index)
			}

			outLinks := make(Links)
			inLinks := make(Links)

			// need to implement other generator for bigger graphs
			for i := 0; i < int(stakesCount); i++ {
				//for _, src := range cidSrc {
				ps := rand.Perm(int(cidsCount))
				for i, j := range ps {
					cidShuf[i] = cidSrc[j]
				}
				for indexSrc, src := range cidSrc {
					if indexSrc % 10000 != 0 { continue }
					if _, exists := outLinks[src]; !exists {
						outLinks[src] = make(CidLinks)
					}
					for indexShuf, dst := range cidShuf {
						//if dst != src {
						//if dst != src && dst % 3 == 0 {
						if indexShuf % 2 == 0 && dst != src {
							outLinks.Put(src, dst, AccNumber(uint64(i)))
							inLinks.Put(dst, src, AccNumber(uint64(i)))
						}
					}
				}
			}

			fmt.Println("Graph generation", "time", time.Since(start))

			start = time.Now()

	 		for i := 0; i < int(cidsCount); i++ {
				if _, ok := outLinks[CidNumber(i)]; !ok {
					if _, ok := inLinks[CidNumber(i)]; !ok {
						fmt.Println("Failed generation, no output/input links on CIDs: ", i)
					}
				}
			}
			fmt.Println("Graph validation", "time", time.Since(start))

			linksCount := uint64(0)
			rank := make([]float64, cidsCount)
			inLinksCount := make([]uint32, cidsCount)
			outLinksCount := make([]uint32, cidsCount)
			inLinksOuts := make([]uint64, 0)
			inLinksUsers := make([]uint64, 0)
			outLinksUsers := make([]uint64, 0)

			stakes := make([]uint64, stakesCount)
			for acc := range stakes {
				stakes[acc] = uint64(rand.Intn(10) + 1)
			}


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
			fmt.Println("Data preparation without workers", "time", time.Since(start))


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

			start = time.Now()
			cRank := (*C.double)(&rank[0])
			C.calculate_rank(
				cStakes, cStakesSize, cCidsSize, cLinksSize,
				cInLinksCount, cOutLinksCount,
				cInLinksOuts, cInLinksUsers, cOutLinksUsers,
				cRank, cDampingFactor, cTolerance,
			)
			fmt.Println("Rank calculation", "time", time.Since(start))

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


