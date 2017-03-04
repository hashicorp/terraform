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
	var mc circonusMetricCluster
	cmc, err := ctxt.client.FetchMetricCluster(cid, "")
	if err != nil {
		return circonusMetricCluster{}, err
	}
	mc.MetricCluster = *cmc

	return mc, nil
}

func (mc *circonusMetricCluster) Create(ctxt *providerContext) error {
	cmc, err := ctxt.client.CreateMetricCluster(&mc.MetricCluster)
	if err != nil {
		return err
	}

	mc.CID = cmc.CID

	return nil
}

func (mc *circonusMetricCluster) Update(ctxt *providerContext) error {
	_, err := ctxt.client.UpdateMetricCluster(&mc.MetricCluster)
	if err != nil {
		return errwrap.Wrapf(fmt.Sprintf("Unable to update stream group %s: {{err}}", mc.CID), err)
	}

	return nil
}

func (mc *circonusMetricCluster) Validate() error {
	if len(mc.Queries) < 1 {
		return fmt.Errorf("there must be at least one stream group query present")
	}

	return nil
}
