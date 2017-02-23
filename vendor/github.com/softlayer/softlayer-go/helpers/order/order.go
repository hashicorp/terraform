/**
 * Copyright 2016 IBM Corp.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package order

import (
	"github.com/softlayer/softlayer-go/datatypes"
	"github.com/softlayer/softlayer-go/services"
	"github.com/softlayer/softlayer-go/session"
)

// CheckBillingOrderStatus returns true if the status of the billing order for
// the provided product order receipt is in the list of provided statuses.
// Returns false otherwise, along with the billing order item used to check the statuses,
// and any error encountered.
func CheckBillingOrderStatus(sess *session.Session, receipt *datatypes.Container_Product_Order_Receipt, statuses []string) (bool, *datatypes.Billing_Order_Item, error) {
	service := services.GetBillingOrderItemService(sess)

	item, err := service.
		Id(*receipt.PlacedOrder.Items[0].Id).
		Mask("mask[id,billingItem[id,provisionTransaction[id,transactionStatus[name]]]]").
		GetObject()

	if err != nil {
		return false, nil, err
	}

	currentStatus := *item.BillingItem.ProvisionTransaction.TransactionStatus.Name
	for _, status := range statuses {
		if currentStatus == status {
			return true, &item, nil
		}
	}

	return false, &item, nil
}

// CheckBillingOrderComplete returns true if the status of the billing order for
// the provided product order receipt is "COMPLETE". Returns false otherwise,
// along with the billing order item used to check the statuses, and any error encountered.
func CheckBillingOrderComplete(sess *session.Session, receipt *datatypes.Container_Product_Order_Receipt) (bool, *datatypes.Billing_Order_Item, error) {
	return CheckBillingOrderStatus(sess, receipt, []string{"COMPLETE"})
}
