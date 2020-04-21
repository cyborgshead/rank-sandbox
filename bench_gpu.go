// +build cuda

package main

import "C"
import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/cybercongress/cyberd/merkle"
	"github.com/spf13/cobra"
)

/*
#cgo CFLAGS: -I/usr/lib/
#cgo LDFLAGS: -L/usr/local/cuda/lib64 -lcbdrank -lcudart
#include "cbdrank.h"
*/
import "C"

func RunBenchGPUCmd() *cobra.Command {

	cmd := &cobra.Command{
		Use:   "run-bench-gpu <stakesCount> <cidsCount> <dampingFactor> <tolerance>",
		Short: "Run rank calculation on GPU",
		Args:  cobra.ExactArgs(4),
		RunE: func(cmd *cobra.Command, args []string) error {

			stakesCount, _ := strconv.ParseInt(args[0], 10, 64)
			cidsCount, _ := strconv.ParseInt(args[1], 10, 64)
			dampingFactor, _ := strconv.ParseFloat(args[1], 64)
			tolerance, _ := strconv.ParseFloat(args[2], 64)

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

			linksCount := uint64(0)
			rank := make([]float64, cidsCount)
			inLinksCount := make([]uint32, cidsCount)
			outLinksCount := make([]uint32, cidsCount)
			inLinksOuts := make([]uint64, 0)
			inLinksUsers := make([]uint64, 0)
			outLinksUsers := make([]uint64, 0)

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
			fmt.Println("Stakes amount", len(stakes))
			fmt.Println("Data preparation", "time", time.Since(start))

			outLinks = nil
			inLinks = nil

			cStakes := (*C.ulong)(&stakes)

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