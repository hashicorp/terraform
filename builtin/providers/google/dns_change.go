package google

import (
	"google.golang.org/api/dns/v1"

	"github.com/hashicorp/terraform/helper/resource"
)

type DnsChangeWaiter struct {
	Service     *dns.Service
	Change      *dns.Change
	Project     string
	ManagedZone string
}

func (w *DnsChangeWaiter) RefreshFunc() resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		var chg *dns.Change
		var err error

		chg, err = w.Service.Changes.Get(
			w.Project, w.ManagedZone, w.Change.Id).Do()

		if err != nil {
			return nil, "", err
		}

		return chg, chg.Status, nil
	}
}

func (w *DnsChangeWaiter) Conf() *resource.StateChangeConf {
	return &resource.StateChangeConf{
		Pending: []string{"pending"},
		Target:  "done",
		Refresh: w.RefreshFunc(),
	}
}
