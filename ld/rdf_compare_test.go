package ld_test

import (
	"log"
	"sort"
	"strings"

	. "github.com/piprate/json-gold/ld"
)

// Perm calls f with each permutation of a.
func Perm(a []string, f func([]string) bool) {
	perm(a, f, 0)
}

// Permute the values at index i to len(a)-1.
func perm(a []string, f func([]string) bool, i int) bool {
	if i > len(a) {
		return f(a)
	}
	if perm(a, f, i+1) {
		// stop
		return true
	}
	for j := i + 1; j < len(a); j++ {
		a[i], a[j] = a[j], a[i]
		if perm(a, f, i+1) {
			// stop
			return true
		}
		a[i], a[j] = a[j], a[i]
	}
	return false
}

// allBlankNodes collects all blank node IDs from every position (subject,
// object, and graph name) across all graphs in a dataset.
func allBlankNodes(ds *RDFDataset) []string {
	set := make(map[string]struct{})
	for graphName, quads := range ds.Graphs {
		if strings.HasPrefix(graphName, "_:") {
			set[graphName] = struct{}{}
		}
		for _, q := range quads {
			if IsBlankNode(q.Subject) {
				set[q.Subject.GetValue()] = struct{}{}
			}
			if IsBlankNode(q.Object) {
				set[q.Object.GetValue()] = struct{}{}
			}
			if q.Graph != nil && IsBlankNode(q.Graph) {
				set[q.Graph.GetValue()] = struct{}{}
			}
		}
	}
	return GetKeys(set)
}

// remapDataset returns a new RDFDataset with all blank node IDs replaced
// according to the mapping actualBlanks[i] -> mappedBlanks[i].  Graph names
// that are blank nodes are remapped along with nodes in subject/object position.
func remapDataset(ds *RDFDataset, actualBlanks, mappedBlanks []string) *RDFDataset {
	nodeMap := make(map[string]string, len(actualBlanks))
	for i, b := range actualBlanks {
		nodeMap[b] = mappedBlanks[i]
	}

	remapNode := func(n Node) Node {
		if IsBlankNode(n) {
			if mapped, ok := nodeMap[n.GetValue()]; ok {
				return NewBlankNode(mapped)
			}
		}
		return n
	}

	newDS := &RDFDataset{Graphs: make(map[string][]*Quad, len(ds.Graphs))}
	for graphName, quads := range ds.Graphs {
		newGraphName := graphName
		if mapped, ok := nodeMap[graphName]; ok {
			newGraphName = mapped
		}
		newQuads := make([]*Quad, 0, len(quads))
		for _, q := range quads {
			subj := remapNode(q.Subject)
			obj := remapNode(q.Object)
			graph := ""
			if q.Graph != nil {
				graph = remapNode(q.Graph).GetValue()
			}
			newQuads = append(newQuads, NewQuad(subj, q.Predicate, obj, graph))
		}
		newDS.Graphs[newGraphName] = newQuads
	}
	return newDS
}

func sortNQuads(input string) string {
	temp := strings.Split(input, "\n")
	if temp[len(temp)-1] == "" {
		temp = temp[:len(temp)-1]
	}
	sort.Strings(temp)
	temp = append(temp, "")
	return strings.Join(temp, "\n")
}

// Isomorphic returns true if two given sets of n-quads are isomorphic.
// This is a lazy implementation and it should only be used for testing.
// We build all possible permutations of blank node IDs and try them one
// by one. Minimal optimisations are applied.
func Isomorphic(expectedStr, actualStr string) bool {
	expected := sortNQuads(expectedStr)
	actual := sortNQuads(actualStr)

	// if quads are identical, exit early
	if DeepCompare(expected, actual, true) {
		return true
	}

	serializer := &NQuadRDFSerializer{}

	expectedDS, err := serializer.Parse(expectedStr)
	if err != nil {
		log.Printf("Error when parsing expected quads: %s\n", err.Error())
		return false
	}
	actualDS, err := serializer.Parse(actualStr)
	if err != nil {
		log.Printf("Error when parsing actual quads: %s\n", err.Error())
		return false
	}

	if len(expectedDS.Graphs) != len(actualDS.Graphs) {
		log.Println("Number of graphs doesn't match")
		return false
	}

	// Collect blank nodes from all positions (subject, object, graph name)
	// across the entire dataset so that blank-node graph names are handled too.
	expectedBlanks := allBlankNodes(expectedDS)
	actualBlanks := allBlankNodes(actualDS)
	if len(expectedBlanks) != len(actualBlanks) {
		log.Println("Number of blank nodes doesn't match")
		return false
	}

	expectedObj, _ := serializer.Serialize(expectedDS)
	expectedSorted := sortNQuads(expectedObj.(string))

	isomorphic := false
	Perm(expectedBlanks, func(perm []string) bool {
		permutedDS := remapDataset(actualDS, actualBlanks, perm)
		permutedObj, _ := serializer.Serialize(permutedDS)
		permutedSorted := sortNQuads(permutedObj.(string))

		if DeepCompare(expectedSorted, permutedSorted, true) {
			isomorphic = true
			return true
		}
		return false
	})
	return isomorphic
}
