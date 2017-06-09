package softlayer

type SoftLayer_Billing_Item_Service interface {
	Service

	CancelService(billingId int) (bool, error)
}
