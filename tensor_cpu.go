package main

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"github.com/cybercongress/cyberd/merkle"
	"github.com/spf13/cobra"
	"math"
	"strconv"
	"time"
)

type UserOutLink struct {
	UserID  uint64
	OutLink uint64
}

type Pair struct {
	First  uint64
	Second float64
}

func findIntersection(arr1, arr2 []int) []int {
	intersection := make([]int, 0)

	set := make(map[int]bool)

	// Create a set from the first array
	for _, num := range arr1 {
		set[num] = true // setting the initial value to true
	}

	// Check elements in the second array against the set
	for _, num := range arr2 {
		if set[num] {
			intersection = append(intersection, num)
		}
	}

	return intersection
}

func cosineSimilarity(a, b []int) float64 {
	var intersection = findIntersection(a, b)
	// fmt.Printf("Intersection %d and intersection len %d and A %.2f and B %.2f \n", intersection, len(intersection), math.Pow(float64(len(a)),2), math.Pow(float64(len(b)),2))
	return float64(len(intersection)) / math.Sqrt(float64(len(a)*len(b)))
}

func cosineSimilarity_raw(common, a_len, b_len uint32) float64 {
	// fmt.Printf("Intersection %d and intersection len %d and A %.2f and B %.2f \n", intersection, len(intersection), math.Pow(float64(len(a)),2), math.Pow(float64(len(b)),2))
	return float64(common) / math.Sqrt(float64(a_len*b_len))
}

func matmulSparse(sparseMatrix [][]Pair, vector []float64, columns uint16) []float64 {
	result := make([]float64, columns)

	for i, sparseRow := range sparseMatrix {
		for _, value := range sparseRow {
			result[value.First] += vector[i] * value.Second
			//fmt.Printf("%d %.2f %.2f \n", value.First, vector[i], value.Second)
		}
	}
	return result
}

func weightedMedianColSparse(stake []float64, score [][]Pair, columns uint16, majority float64) []float64 {
	rows := len(stake)
	zero := float64(0)
	useStake := make([]float64, 0)
	for _, s := range stake {
		if s > zero {
			useStake = append(useStake, s)
		}
	}
	inplaceNormalize(useStake)
	stakeSum := sum(useStake)
	stakeIdx := makeRange(0, len(useStake))
	minority := stakeSum - majority
	useScore := make([][]float64, columns)
	for i := range useScore {
		useScore[i] = make([]float64, len(useStake))
	}
	median := make([]float64, columns)
	k := 0
	for r := 0; r < rows; r++ {
		if stake[r] <= zero {
			continue
		}
		for _, val := range score[r] {
			useScore[val.First][k] = val.Second
		}
		k++
	}
	for c := 0; c < int(columns); c++ {
		median[c] = weightedMedian(useStake, useScore[c], stakeIdx, minority, zero, stakeSum)
	}
	return median
}

func inplaceNormalize(x []float64) {
	xSum := sum(x)
	if xSum == 0 {
		return
	}
	for i := range x {
		x[i] = x[i] / xSum
	}
}

func weightedMedian(stake []float64, score []float64, partitionIdx []int, minority float64, partitionLo float64, partitionHi float64) float64 {
	n := len(partitionIdx)
	if n == 0 {
		return 0
	}
	if n == 1 {
		return score[partitionIdx[0]]
	}
	midIdx := n / 2
	pivot := score[partitionIdx[midIdx]]
	loStake := float64(0)
	hiStake := float64(0)
	lower := make([]int, 0)
	upper := make([]int, 0)
	for _, idx := range partitionIdx {
		if score[idx] == pivot {
			continue
		}
		if score[idx] < pivot {
			loStake += stake[idx]
			lower = append(lower, idx)
		} else {
			hiStake += stake[idx]
			upper = append(upper, idx)
		}
	}
	if partitionLo+loStake <= minority && minority < partitionHi-hiStake {
		return pivot
	} else if minority < partitionLo+loStake && len(lower) > 0 {
		return weightedMedian(stake, score, lower, minority, partitionLo, partitionLo+loStake)
	} else if partitionHi-hiStake <= minority && len(upper) > 0 {
		return weightedMedian(stake, score, upper, minority, partitionHi-hiStake, partitionHi)
	}
	return pivot
}

func sum(a []float64) float64 {
	sum := float64(0)
	for _, v := range a {
		sum += v
	}
	return sum
}

func makeRange(min, max int) []int {
	a := make([]int, max-min)
	for i := range a {
		a[i] = min + i
	}
	return a
}

func colClipSparse(sparseMatrix [][]Pair, colThreshold []float64) [][]Pair {
	result := make([][]Pair, len(sparseMatrix))
	for i, sparseRow := range sparseMatrix {
		for _, value := range sparseRow {
			if colThreshold[value.First] < value.Second {
				if 0 < colThreshold[value.First] {
					result[i] = append(result[i], Pair{value.First, colThreshold[value.First]})
				}
			} else {
				result[i] = append(result[i], value)
			}
		}
	}
	return result
}

func rowSumSparse(sparseMatrix [][]Pair) []float64 {
	rows := len(sparseMatrix)
	result := make([]float64, rows)
	for i, sparseRow := range sparseMatrix {
		for _, value := range sparseRow {
			result[i] += value.Second
		}
	}
	return result
}

func vecdiv(x []float64, y []float64) []float64 {
	if len(x) != len(y) {
		panic("Length of slices x and y must be equal")
	}
	n := len(x)
	result := make([]float64, n)
	for i := 0; i < n; i++ {
		if y[i] != 0 {
			result[i] = x[i] / y[i]
		}
	}
	return result
}

func tensor_cpu(inLinksCount []uint32, inLinksOuts []uint64, inLinksUsers []uint64, stakes_new map[AccNumber]uint64) []float64 {

	start := time.Now()
	var pointer uint32 = 0
	var userTotalLinks = make([]uint32, len(stakes_new))
	var usersCrosslinks = make(map[uint64]map[uint64]uint32, len(stakes_new))
	for _, counter := range inLinksCount {
		for i := uint32(pointer); i < (pointer + counter); i++ {
			userTotalLinks[inLinksUsers[i]] += 1
			//println("+ particle_to", particleTo, "inLinksOuts[i]", inLinksOuts[i], "inLinksUsers[i]", inLinksUsers[i])

			for j := i; j < (pointer + counter); j++ {
				if j+1 == (pointer + counter) {
					break
				}
				if inLinksOuts[i] == inLinksOuts[j+1] {
					if usersCrosslinks[inLinksUsers[i]] == nil {
						usersCrosslinks[inLinksUsers[i]] = make(map[uint64]uint32)
					}
					if usersCrosslinks[inLinksUsers[j+1]] == nil {
						usersCrosslinks[inLinksUsers[j+1]] = make(map[uint64]uint32)
					}
					usersCrosslinks[inLinksUsers[i]][inLinksUsers[j+1]] += 1
					usersCrosslinks[inLinksUsers[j+1]][inLinksUsers[i]] += 1
					//println(" -> particle_to", particleTo, "inLinksUsers[i]", inLinksUsers[i], "inLinksUsers[i]", inLinksUsers[j+1])
					//println(" -> particle_to", particleTo, "inLinksUsers[i]", inLinksUsers[j+1], "inLinksUsers[i]", inLinksUsers[i])
				} else {
					break
				}
			}
		}
		pointer += counter
	}
	fmt.Println("crosslinks compute: ", "time", time.Since(start))

	var sum_check uint32 = 0
	for _, userLinks := range userTotalLinks {
		//println("user", i, "user_total_links", userLinks)
		sum_check += userLinks
	}
	println("sum_check", sum_check)
	println("--------")

	var sum_crosslink_check uint32 = 0
	for _, crosslinks := range usersCrosslinks {
		for _, value := range crosslinks {
			//println("A{", user1, "} B{", user2, "} cross", value)
			sum_crosslink_check += value
		}
	}
	println("sum_crosslink_check", sum_crosslink_check)

	//type Pair2 struct {
	//	Key   uint64
	//	Value uint32
	//}
	//type Pair3 struct {
	//	Key   uint64
	//	Value []Pair2
	//}
	//var keys []uint64
	//for k := range usersCrosslinks {
	//	keys = append(keys, k)
	//}
	//sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	//
	//// Create a new sorted map
	//var sortedUsersCrosslinks []Pair3
	//for _, k := range keys {
	//	// Extract and sort the nested map keys
	//	var nestedKeys []uint64
	//	for nk := range usersCrosslinks[k] {
	//		nestedKeys = append(nestedKeys, nk)
	//	}
	//	sort.Slice(nestedKeys, func(i, j int) bool { return nestedKeys[i] < nestedKeys[j] })
	//
	//	// Create a new sorted nested map
	//	var sortedNestedUsersCrosslinks []Pair2
	//	for _, nk := range nestedKeys {
	//		sortedNestedUsersCrosslinks = append(sortedNestedUsersCrosslinks, Pair2{nk, usersCrosslinks[k][nk]})
	//	}
	//
	//	sortedUsersCrosslinks = append(sortedUsersCrosslinks, Pair3{k, sortedNestedUsersCrosslinks})
	//}
	//println("--------")
	//for _, pair := range sortedUsersCrosslinks {
	//	for _, value := range pair.Value {
	//		println("A{", pair.Key, "} B{", value.Key, "} cross", value.Value)
	//	}
	//	//println("user", user, "crosslinks", crosslinks)
	//}
	//var sum_check2 uint32 = 0
	//for _, pair := range sortedUsersCrosslinks {
	//	fmt.Printf("User1: %d\n", pair.Key)
	//	for _, nestedPair := range pair.Value {
	//		fmt.Printf("\tUser2: %d, Crosslinks: %v\n", nestedPair.Key, nestedPair.Value)
	//		sum_check2 += nestedPair.Value
	//	}
	//}
	//println("sum_check2", sum_check2)

	sparseWeightsMatrix := make([][]Pair, len(stakes_new))

	start = time.Now()
	for userID1 := range usersCrosslinks {
		for userID2 := range usersCrosslinks {
			if userID1 != userID2 {
				//println("userID1", userID1, "userID2", userID2, "users_crosslinks[userID1][userID2]", usersCrosslinks[userID1][userID2], "user_total_links[userID1]", userTotalLinks[userID1], "user_total_links[userID2]", userTotalLinks[userID2])
				cosim := cosineSimilarity_raw(usersCrosslinks[userID1][userID2], userTotalLinks[userID1], userTotalLinks[userID2])
				sparseWeightsMatrix[userID1] = append(sparseWeightsMatrix[userID1], Pair{userID2, cosim})
				//fmt.Printf("Cosine similarity between user %d and user %d: %.2f\n", userID1, userID2, cosim)
			}
		}
	}
	fmt.Println("weights compute: ", "time", time.Since(start))

	//fmt.Println("Sparse Matrix: ")
	//for i, row := range sparseWeightsMatrix {
	//	fmt.Println("Row ", i, ":", row)
	//}

	start = time.Now()

	stakes := make([]float64, len(stakes_new))
	for i, stake := range stakes_new {
		stakes[i] = float64(stake)
	}

	//fmt.Println("Stakes: ", stakes)
	columns := uint16(len(stakes))

	// Compute preranks: r_j = SUM(i) w_ij * s_i
	preranks := matmulSparse(sparseWeightsMatrix, stakes, columns)
	//fmt.Println("Preranks: ", preranks)

	// Clip weights at majority consensus
	kappa := float64(0.20)
	//fmt.Println("Kappa: ", kappa)

	consensus := weightedMedianColSparse(stakes, sparseWeightsMatrix, columns, kappa)
	//fmt.Println("Consensus: ", consensus)

	weights := colClipSparse(sparseWeightsMatrix, consensus)

	//fmt.Println("Weights: ", weights)

	//validator_trust := rowSumSparse(weights)
	//fmt.Println("Validator trust: ", validator_trust)

	// =============================
	// == Ranks, Trust, Incentive ==
	// =============================

	// Compute ranks: r_j = SUM(i) w_ij * s_i.
	ranks := matmulSparse(weights, stakes, columns)
	//fmt.Println("Ranks: ", ranks)

	trust := vecdiv(ranks, preranks)
	//fmt.Println("Trust: ", trust)

	fmt.Println("trust compute: ", "time", time.Since(start))

	//inplace_normalize(&mut ranks); // range: I32F32(0, 1)
	//let incentive: Vec<I32F32> = ranks.clone();
	//println!("Incentive: {:?}", &incentive);
	incentive := ranks
	inplaceNormalize(incentive)
	//fmt.Println("Incentive: ", incentive)

	//file3, err := os.Create("trust.csv")
	//if err != nil {
	//	fmt.Println("Error:", err)
	//	panic(err)
	//}
	////defer file.Close()
	//
	//writer := csv.NewWriter(file3)
	//defer writer.Flush()
	//
	//// Write header
	//writer.Write([]string{"id", "trust"})
	//
	//// Write data
	//for id, t := range trust {
	//	writer.Write([]string{strconv.Itoa(id), strconv.FormatFloat(t, 'f', 6, 64)})
	//}

	// =========================
	// == Bonds and Dividends ==
	// =========================

	return trust
}

func RunTensorCPUCmd() *cobra.Command {

	//./cyberrank gen-graph 1000 20000 10000 42
	//./cyberrank run-tensor-cpu 1000 10000 39996042

	cmd := &cobra.Command{
		Use:   "run-tensor-cpu <stakesCount> <cidsCount> <totalLinks>",
		Short: "Run rank calculation on CPU",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {

			stakesCount, _ := strconv.ParseInt(args[0], 10, 64)
			cidsCount, _ := strconv.ParseInt(args[1], 10, 64)
			linksCount, _ := strconv.ParseInt(args[2], 10, 64)

			fmt.Println("Neurons: ", stakesCount)
			fmt.Println("CIDs: ", cidsCount)

			start := time.Now()

			outLinks := make(map[CidNumber]CidLinks)
			inLinks := make(map[CidNumber]CidLinks)
			stakes := make(map[AccNumber]uint64)

			readStakesFromBytesFile(&stakes, "./stakes.data")
			readLinksFromBytesFile(&outLinks, "./outLinks.data")
			readLinksFromBytesFile(&inLinks, "./inLinks.data")
			fmt.Println("Graph open data: ", "time", time.Since(start))

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

			outLinks = nil
			inLinks = nil

			trust := tensor_cpu(inLinksCount, inLinksOuts, inLinksUsers, stakes)

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
