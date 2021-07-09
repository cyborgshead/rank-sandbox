# cyberrank-benchmark

## Install

```
sudo make
```

Check Makefile for internals

## Run

```
./cyberrank gen-graph <stakesCount> <linksPerAgent> <cidsCount> <randSeed>
./cyberrank run-bench-{unit} <stakesCount> <cidsCount> <dampingFactor> <tolerance> <debug-values>
```

## Example
```
## cross validation
./cyberrank gen-graph 10 10 20 1
./cyberrank run-bench-cpu 10 20 0.85 0.001 true
./cyberrank run-bench-gpu 10 20 0.85 0.001 true

## play with gpu
./cyberrank gen-graph 10000 1000 1000000 1
./cyberrank run-bench-gpu 10000 1000000 0.85 0.001 false
```

## Output (GPU)
```
---------------------------------

STEP 0: Graph load
Agents:  10000
CIDs:  1000000
Damping:  0.85
Tolerance:  0.001
Graph open data:  time 15.697448012s
-[GO] Memory: 3919
---------------------------------

STEP 1: Prepare memory
-[GO] Memory: 3972
---------------------------------

STEP 2: Data transformation
Links amount 10000056
Stakes amount 10000
Data preparation time 7.088283781s
-[GO] Memory: 5085
---------------------------------
STEP 2: Rank calculation
[GPU]: Usage Offset: 96.31MB
STEP0: Calculate compressed in links start indexes
-[GPU]: Free: 5845.81MB	Used: 0.00MB
STEP1: Calculate for each cid total stake by out links
-[GPU]: Free: 5745.81MB	Used: 100.00MB
DEV ENTROPY - IN STAKE
-[GPU]: Free: 5737.81MB	Used: 108.00MB
DEV ENTROPY - ENTROPY
-[GPU]: Free: 5729.81MB	Used: 116.00MB
LOCAL WEIGHTS
-[GPU]: Free: 5651.81MB	Used: 194.00MB
STEP2: Calculate compressed in links count
-[GPU]: Free: 5557.81MB	Used: 288.00MB
STEP3: Calculate compressed in links start indexes
-[GPU]: Free: 5549.81MB	Used: 296.00MB
STEP4: Calculate compressed in links
-[GPU]: Free: 5503.81MB	Used: 342.00MB
STEP5: Calculate dangling nodes rank, and default rank
-[GPU]: Free: 5503.81MB	Used: 342.00MB
STEP6: Calculate Rank
-[GPU]: Free: 5487.81MB	Used: 358.00MB
STEP7: Calculate Light
-[GPU]: Free: 5669.81MB	Used: 176.00MB
STEP8: Calculate Karma
-[GPU]: Free: 5667.81MB	Used: 178.00MB
STEP9: Cleaning
-[GPU]: Free: 5845.81MB	Used: 0.00MB
Rank calculation time 3.129636228s
-[GO] Memory: 5085
---------------------------------
STEP 3: Data and stats
Ranks reduction:  time 921.691µs
RanksSum: 0.277508
-------------
Rank converting to uint:  time 2.373032ms
-------------
Rank constructing merkle tree:  time 931.821272ms
Rank merkle root hash: 73dfdf8eea2fb587a65f279f8021991c47b0017bbde2a8744efd562938f0adfb
-------------
Entropy reduction:  time 953.72µs
Entropy: 2003560.926978
-------------
Entropy converting to uint:  time 2.331701ms
-------------
Entropy constructing merkle tree:  time 929.164116ms
Entropy merkle root hash: be4f2620441b9a7a2ca48997fcddba4c17f9635721c41024d4479c55b4c48ccb
-------------
Light converting to uint:  time 2.367135ms
-------------
Light constructing merkle tree:  time 985.600395ms
Light merkle root hash: cb486f66ef6276c10d2fd9778dde76648e3ae98f5540cce35d639c9708a75404
-------------
Karma converting to uint:  time 15.494µs
-------------
Karma constructing merkle tree:  time 10.026613ms
Karma merkle root hash: d3b5ee368611280366023c23e58388fe9d778b49620d902d4675fac3d17da974
-------------
Karma reduction:  time 11.475µs
KarmaSum: 0.282128
-------------
-[GO] Memory: 5935
---------------------------------
```

## Output for debug and cross-validation (CPU)
```
Agents:  10
CIDs:  10
Damping:  0.85
Tolerance:  0.001
---------------------------------
outLinks:  map[0:map[1:map[1:{}] 3:map[1:{}] 4:map[7:{}] 5:map[4:{}] 6:map[4:{}] 9:map[7:{}]] 1:map[4:map[1:{}] 7:map[4:{}] 8:map[7:{}]] 2:map[4:map[8:{}] 5:map[5:{}] 8:map[2:{}]] 3:map[1:map[5:{}] 8:map[2:{} 8:{}]] 4:map[5:map[5:{}] 7:map[8:{}] 9:map[2:{}]] 5:map[1:map[9:{}] 2:map[3:{}] 6:map[6:{}]] 6:map[3:map[9:{}] 7:map[3:{}] 9:map[6:{}]] 7:map[0:map[6:{}] 2:map[9:{}] 9:map[3:{}]] 8:map[0:map[0:{}] 1:map[0:{}] 5:map[0:{}]] 9:map[4:map[0:{}] 6:map[0:{}]]]
Graph open data:  time 313.984µs
---------------------------------
Rank calculation defaultRank 0.015000000000000003
Rank calculation danglingNodesSize 0
Rank calculation defaultRankWithCorrection 0.015000000000000003
Rank calculation time 1.021441ms
RanksSum: 0.961240
Rank converting to uint:  time 244ns
Ranks []float64:  [0.07677196578716734 0.1108431713497047 0.07747281743065203 0.05813936339997833 0.11701145468273072 0.11104249175799072 0.09659783698713889 0.11107465324399031 0.10852580070315508 0.09376091357297728]
Ranks []uint64:  [767719657 1108431713 774728174 581393633 1170114546 1110424917 965978369 1110746532 1085258007 937609135]
---------------------------------
EntropySum: 11.955885
Entropy calculation:  time 66.292µs
Entropy converting to uint:  time 207ns
Entropy []float64:  [2.16364590669563 1.1383030042446554 1.2876878900165245 1.3176681067160705 1.201808097960877 1.285196953662317 1.3519398312641258 1.285196953662317 0.5423584290162798 0.38208020839342965]
Entropy []uint64:  [21636459066 11383030042 12876878900 13176681067 12018080979 12851969536 13519398312 12851969536 5423584290 3820802083]
---------------------------------
Light calculation:  time 224ns
Light converting to uint:  time 200ns
Light []float64:  [0.16610734952438155 0.12617311494737396 0.09976080881091173 0.07660838489692705 0.14062531379188795 0.1427114721344426 0.1305944634368721 0.14275280597827453 0.05885988277709707 0.03582418939712151]
Light []uint64:  [1661073495 1261731149 997608088 766083848 1406253137 1427114721 1305944634 1427528059 588598827 358241893]
---------------------------------
Karma calculation:  time 37.048µs
Karma converting to uint:  time 217ns
KarmaSum []float64:  0.599644162037938
Karma []float64:  [0.009776874052245687 0.024906932098542035 0.028115412772969993 0.04464244202072517 0.062267330246355085 0.056230825545939986 0.07812427353626905 0.09962772839416814 0.08434623831890997 0.11160610505181293]
Karma []uint64:  [97768740 249069320 281154127 446424420 622673302 562308255 781242735 996277283 843462383 1116061050]
---------------------------------
Stake []uint64:  [200 400 600 800 1000 1200 1400 1600 1800 2000]
---------------------------------
Rank constructing merkle tree:  time 31.588µs
Rank merkle root hash: 75b17e8fc1cae28dcc84f22f0fc4bfb00fafcf9da93283d54a2aa00cd9e8fb8e
---------------------------------
Entropy constructing merkle tree:  time 26.469µs
Entropy merkle root hash: 5e672e879b7598281bff6a33d47b27e039e8cf3fd88c48bd30c739af468069c4
---------------------------------
Light constructing merkle tree:  time 35.64µs
Light merkle root hash: b5bcfd81c19005f1fe7708a68f51e9010132f44201055c3683e6c150988c982b
---------------------------------
Karma constructing merkle tree:  time 26.078µs
Karma merkle root hash: 6b7abd33b5ac92e1102fe1ad43862c51a9329d26bc18bb46ee4a71cb224f1d5f
---------------------------------
Prepare mocked data for CPU-GPU results cross check validation
OutLinks, InLinks and Stakes saved in:  time 33.507409ms
```
