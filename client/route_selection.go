package main

import  (
	"math/rand"
)

// GetNodesInRoute randomly selects up to 3 nodes from the list based on their load.
// Nodes with lower load (i.e. higher weight calculated as 1/(float64(Load)+epsilon)) are more likely to be picked.
func GetNodesInRoute(nodes []RelayNode) []RelayNode {
	const picks = 3
	const epsilon = 1e-6
	chosenNodes := []RelayNode{}

	candidateNodes := make([]RelayNode, len(nodes))
	copy(candidateNodes, nodes)

	for i := 0; i < picks && len(candidateNodes) > 0; i++ {
		totalWeight := 0.0
		weights := make([]float64, len(candidateNodes))

		for j, node := range candidateNodes {
			weight := 1.0 / (float64(node.Load) + epsilon)
			weights[j] = weight
			totalWeight += weight
		}

		r := rand.Float64() * totalWeight

		sum := 0.0
		selectedIndex := 0
		for j, w := range weights {
			sum += w
			if r <= sum {
				selectedIndex = j
				break
			}
		}

		chosenNodes = append(chosenNodes, candidateNodes[selectedIndex])

		candidateNodes = append(candidateNodes[:selectedIndex], candidateNodes[selectedIndex+1:]...)
	}

	return chosenNodes
}