// +build cuda

package main

import "C"
import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math"
	"runtime"
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
		Use:   "run-bench-gpu <stakesCount> <cidsCount> <dampingFactor> <tolerance> <debug>",
		Short: "Run rank calculation on GPU",
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

			// inLinks.Put(9, 8, 0)
			// inLinks.Put(10, 9, 0)
			// inLinks.Put(7, 8, 0)
			// inLinks.Put(6, 7, 0)
			// inLinks.Put(19, 20, 0)
			// inLinks.Put(3, 4, 0)
			// inLinks.Put(5, 6, 1)
			// inLinks.Put(5, 20, 1)
			// inLinks.Put(5, 4, 1)
			// inLinks.Put(18, 19, 1)
			// inLinks.Put(18, 17, 1)
			// inLinks.Put(11, 10, 1)
			// inLinks.Put(11, 12, 1)
			// inLinks.Put(14, 0, 1)
			// inLinks.Put(2, 3, 2)
			// inLinks.Put(2, 17, 2)
			// inLinks.Put(2, 16, 2)
			// inLinks.Put(13, 16, 2)
			// inLinks.Put(13, 12, 2)
			// inLinks.Put(13, 15, 2)
			// inLinks.Put(1, 15, 2)
			// inLinks.Put(1, 0, 2)
			// inLinks.Put(13, 14, 3)
			// inLinks.Put(1, 14, 3)
			// inLinks.Put(1, 2, 4)
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
			linksCount := uint64(0)
			rank := make([]float64, cidsCount)
			rankUint := make([]uint64, cidsCount)
			ent := make([]float64, cidsCount)
			entUint := make([]uint64, cidsCount)
			light := make([]float64, cidsCount)
			karma := make([]float64, stakesCount)
			karmaUint := make([]uint64, stakesCount)

			inLinksCount := make([]uint32, cidsCount)
			outLinksCount := make([]uint32, cidsCount)
			inLinksOuts := make([]uint64, 0)
			outLinksIns := make([]uint64, 0)
			inLinksUsers := make([]uint64, 0)
			outLinksUsers := make([]uint64, 0)

			runtime.ReadMemStats(mem)
			fmt.Println("-[GO] Memory:", (mem.Alloc-memUsageOffset)/base)

			fmt.Println("---------------------------------\n")
			fmt.Println("STEP 2: Data transformation")

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

				if outLinks, sortedCids, ok := GetSortedInLinks(outLinks, CidNumber(i)); ok {
					for _, cid := range sortedCids {
						outLinksCount[i] += uint32(len(outLinks[cid]))
						for acc := range outLinks[cid] {
							outLinksIns = append(outLinksIns, uint64(cid))
							outLinksUsers = append(outLinksUsers, uint64(acc))
						}
					}
					// linksCount += uint64(inLinksCount[i])
				}

				// if outLinks, ok := outLinks[CidNumber(i)]; ok {
				// 	for _, accs := range outLinks {
				// 		outLinksCount[i] += uint32(len(accs))
				// 		for acc := range accs {
				// 			outLinksUsers = append(outLinksUsers, uint64(acc))
				// 		}
				// 	}
				// }
			}
			fmt.Println("Links amount", linksCount)
			fmt.Println("Stakes amount", len(stakes))
			fmt.Println("Data preparation", "time", time.Since(start))

			runtime.ReadMemStats(mem)
			fmt.Println("-[GO] Memory:", (mem.Alloc-memUsageOffset)/base)

			fmt.Println("---------------------------------")
			fmt.Println("STEP 2: Rank calculation")

			outLinks = nil
			inLinks = nil

			cStakes := (*C.ulong)(&stakes[0])

			cStakesSize := C.ulong(len(stakes))
			cCidsSize := C.ulong(len(inLinksCount))
			cLinksSize := C.ulong(len(inLinksOuts))

			cInLinksCount := (*C.uint)(&inLinksCount[0])
			cOutLinksCount := (*C.uint)(&outLinksCount[0])

			cOutLinksIns := (*C.ulong)(&outLinksIns[0])

			cInLinksOuts := (*C.ulong)(&inLinksOuts[0])
			cInLinksUsers := (*C.ulong)(&inLinksUsers[0])
			cOutLinksUsers := (*C.ulong)(&outLinksUsers[0])

			cDampingFactor := C.double(dampingFactor)
			cTolerance := C.double(tolerance)

			start = time.Now()
			cRank := (*C.double)(&rank[0])
			cEntropy := (*C.double)(&ent[0])
			cLight := (*C.double)(&light[0])
			cKarma := (*C.double)(&karma[0])
			C.calculate_rank(
				cStakes, cStakesSize, cCidsSize, cLinksSize,
				cInLinksCount, cOutLinksCount, cOutLinksIns,
				cInLinksOuts, cInLinksUsers, cOutLinksUsers,
				cRank, cDampingFactor, cTolerance, cEntropy, cLight, cKarma,
			)
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

			fmt.Println("---------------------------------\n")

			return nil
		},
	}

	return cmd
}
