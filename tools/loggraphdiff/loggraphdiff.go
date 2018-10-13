// loggraphdiff is a tool for interpreting changes to the Terraform graph
// based on the simple graph printing format used in the TF_LOG=trace log
// output from Terraform, which looks like this:
//
//     aws_instance.b (destroy) - *terraform.NodeDestroyResourceInstance
//     aws_instance.b (prepare state) - *terraform.NodeApplyableResource
//       provider.aws - *terraform.NodeApplyableProvider
//     aws_instance.b (prepare state) - *terraform.NodeApplyableResource
//       provider.aws - *terraform.NodeApplyableProvider
//     module.child.aws_instance.a (destroy) - *terraform.NodeDestroyResourceInstance
//       module.child.aws_instance.a (prepare state) - *terraform.NodeApplyableResource
//       module.child.output.a_output - *terraform.NodeApplyableOutput
//       provider.aws - *terraform.NodeApplyableProvider
//     module.child.aws_instance.a (prepare state) - *terraform.NodeApplyableResource
//       provider.aws - *terraform.NodeApplyableProvider
//     module.child.output.a_output - *terraform.NodeApplyableOutput
//       module.child.aws_instance.a (prepare state) - *terraform.NodeApplyableResource
//     provider.aws - *terraform.NodeApplyableProvider
//
// It takes the names of two files containing this style of output and
// produces a single graph description in graphviz format that shows the
// differences between the two graphs: nodes and edges which are only in the
// first graph are shown in red, while those only in the second graph are
// shown in green. This color combination is not useful for those who are
// red/green color blind, so the result can be adjusted by replacing the
// keywords "red" and "green" with a combination that the user is able to
// distinguish.

package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
)

type Graph struct {
	nodes map[string]struct{}
	edges map[[2]string]struct{}
}

func main() {
	if len(os.Args) != 3 {
		log.Fatal("usage: loggraphdiff <old-graph-file> <new-graph-file>")
	}

	old, err := readGraph(os.Args[1])
	if err != nil {
		log.Fatalf("failed to read %s: %s", os.Args[1], err)
	}
	new, err := readGraph(os.Args[2])
	if err != nil {
		log.Fatalf("failed to read %s: %s", os.Args[1], err)
	}

	var nodes []string
	for n := range old.nodes {
		nodes = append(nodes, n)
	}
	for n := range new.nodes {
		if _, exists := old.nodes[n]; !exists {
			nodes = append(nodes, n)
		}
	}
	sort.Strings(nodes)

	var edges [][2]string
	for e := range old.edges {
		edges = append(edges, e)
	}
	for e := range new.edges {
		if _, exists := old.edges[e]; !exists {
			edges = append(edges, e)
		}
	}
	sort.Slice(edges, func(i, j int) bool {
		if edges[i][0] != edges[j][0] {
			return edges[i][0] < edges[j][0]
		}
		return edges[i][1] < edges[j][1]
	})

	fmt.Println("digraph G {")
	fmt.Print("  rankdir = \"BT\";\n\n")
	for _, n := range nodes {
		var attrs string
		_, inOld := old.nodes[n]
		_, inNew := new.nodes[n]
		switch {
		case inOld && inNew:
			// no attrs required
		case inOld:
			attrs = " [color=red]"
		case inNew:
			attrs = " [color=green]"
		}
		fmt.Printf("    %q%s;\n", n, attrs)
	}
	fmt.Println("")
	for _, e := range edges {
		var attrs string
		_, inOld := old.edges[e]
		_, inNew := new.edges[e]
		switch {
		case inOld && inNew:
			// no attrs required
		case inOld:
			attrs = " [color=red]"
		case inNew:
			attrs = " [color=green]"
		}
		fmt.Printf("    %q -> %q%s;\n", e[0], e[1], attrs)
	}
	fmt.Println("}")
}

func readGraph(fn string) (Graph, error) {
	ret := Graph{
		nodes: map[string]struct{}{},
		edges: map[[2]string]struct{}{},
	}
	r, err := os.Open(fn)
	if err != nil {
		return ret, err
	}

	sc := bufio.NewScanner(r)
	var latestNode string
	for sc.Scan() {
		l := sc.Text()
		dash := strings.Index(l, " - ")
		if dash == -1 {
			// invalid line, so we'll ignore it
			continue
		}
		name := l[:dash]
		if strings.HasPrefix(name, "  ") {
			// It's an edge
			name = name[2:]
			edge := [2]string{latestNode, name}
			ret.edges[edge] = struct{}{}
		} else {
			// It's a node
			latestNode = name
			ret.nodes[name] = struct{}{}
		}
	}

	return ret, nil
}
