package data_types

type SoftLayer_Billing_Item_Cancellation_Request_Parameters struct {
	Parameters []SoftLayer_Billing_Item_Cancellation_Request `json:"parameters"`
}

type SoftLayer_Billing_Item_Cancellation_Request struct {
	ComplexType string                                             `json:"complexType"`
	AccountId   int                                                `json:"accountId"`
	Id          int                                                `json:"id"`
	TicketId    int                                                `json:"ticketId"`
	Items       []SoftLayer_Billing_Item_Cancellation_Request_Item `json:"items"`
}

type SoftLayer_Billing_Item_Cancellation_Request_Item struct {
	BillingItemId             int  `json:"billingItemId"`
	ImmediateCancellationFlag bool `json:"immediateCancellationFlag"`
}
