package circonus

import (
	"fmt"

	"github.com/circonus-labs/circonus-gometrics/api"
	"github.com/hashicorp/errwrap"
)

type _StreamGroup struct {
	api.MetricCluster
}

func _NewStreamGroup() _StreamGroup {
	return _StreamGroup{
		MetricCluster: api.MetricCluster{},
	}
}

func _LoadStreamGroup(ctxt *_ProviderContext, cid api.CIDType) (_StreamGroup, error) {
	var sg _StreamGroup
	mc, err := ctxt.client.FetchMetricCluster(cid, "")
	if err != nil {
		return _StreamGroup{}, err
	}
	sg.MetricCluster = *mc

	return sg, nil
}

func (sg *_StreamGroup) Create(ctxt *_ProviderContext) error {
	mc, err := ctxt.client.CreateMetricCluster(&sg.MetricCluster)
	if err != nil {
		return err
	}

	sg.CID = mc.CID

	return nil
}

func (sg *_StreamGroup) Update(ctxt *_ProviderContext) error {
	_, err := ctxt.client.UpdateMetricCluster(&sg.MetricCluster)
	if err != nil {
		return errwrap.Wrapf(fmt.Sprintf("Unable to update stream group %s: {{err}}", sg.CID), err)
	}

	return nil
}

func (sg *_StreamGroup) Validate() error {
	if len(sg.Queries) < 1 {
		return fmt.Errorf("there must be at least one stream group query present")
	}

	return nil
}
