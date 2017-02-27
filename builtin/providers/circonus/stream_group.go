package circonus

import (
	"fmt"

	"github.com/circonus-labs/circonus-gometrics/api"
	"github.com/hashicorp/errwrap"
)

type circonusStreamGroup struct {
	api.MetricCluster
}

func newStreamGroup() circonusStreamGroup {
	return circonusStreamGroup{
		MetricCluster: api.MetricCluster{},
	}
}

func loadStreamGroup(ctxt *providerContext, cid api.CIDType) (circonusStreamGroup, error) {
	var sg circonusStreamGroup
	mc, err := ctxt.client.FetchMetricCluster(cid, "")
	if err != nil {
		return circonusStreamGroup{}, err
	}
	sg.MetricCluster = *mc

	return sg, nil
}

func (sg *circonusStreamGroup) Create(ctxt *providerContext) error {
	mc, err := ctxt.client.CreateMetricCluster(&sg.MetricCluster)
	if err != nil {
		return err
	}

	sg.CID = mc.CID

	return nil
}

func (sg *circonusStreamGroup) Update(ctxt *providerContext) error {
	_, err := ctxt.client.UpdateMetricCluster(&sg.MetricCluster)
	if err != nil {
		return errwrap.Wrapf(fmt.Sprintf("Unable to update stream group %s: {{err}}", sg.CID), err)
	}

	return nil
}

func (sg *circonusStreamGroup) Validate() error {
	if len(sg.Queries) < 1 {
		return fmt.Errorf("there must be at least one stream group query present")
	}

	return nil
}
