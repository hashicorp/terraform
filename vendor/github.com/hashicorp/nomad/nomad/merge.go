package nomad

import (
	"fmt"

	"github.com/hashicorp/serf/serf"
)

// serfMergeDelegate is used to handle a cluster merge on the gossip
// ring. We check that the peers are nomad servers and abort the merge
// otherwise.
type serfMergeDelegate struct {
}

func (md *serfMergeDelegate) NotifyMerge(members []*serf.Member) error {
	for _, m := range members {
		ok, _ := isNomadServer(*m)
		if !ok {
			return fmt.Errorf("member '%s' is not a server", m.Name)
		}
	}
	return nil
}
