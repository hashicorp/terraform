package circonus

import (
	"fmt"

	"github.com/circonus-labs/circonus-gometrics/api"
	"github.com/hashicorp/errwrap"
)

type circonusMetricCluster struct {
	api.MetricCluster
}

func newMetricCluster() circonusMetricCluster {
	return circonusMetricCluster{
		MetricCluster: api.MetricCluster{},
	}
}

func loadMetricCluster(ctxt *providerContext, cid api.CIDType) (circonusMetricCluster, error) {
	var sg circonusMetricCluster
	mc, err := ctxt.client.FetchMetricCluster(cid, "")
	if err != nil {
		return circonusMetricCluster{}, err
	}
	sg.MetricCluster = *mc

	return sg, nil
}

func (sg *circonusMetricCluster) Create(ctxt *providerContext) error {
	mc, err := ctxt.client.CreateMetricCluster(&sg.MetricCluster)
	if err != nil {
		return err
	}

	sg.CID = mc.CID

	return nil
}

func (sg *circonusMetricCluster) Update(ctxt *providerContext) error {
	_, err := ctxt.client.UpdateMetricCluster(&sg.MetricCluster)
	if err != nil {
		return errwrap.Wrapf(fmt.Sprintf("Unable to update stream group %s: {{err}}", sg.CID), err)
	}

	return nil
}

func (sg *circonusMetricCluster) Validate() error {
	if len(sg.Queries) < 1 {
		return fmt.Errorf("there must be at least one stream group query present")
	}

	return nil
}
