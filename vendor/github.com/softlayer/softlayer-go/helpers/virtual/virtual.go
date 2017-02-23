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

package virtual

import (
	"time"

	"github.com/softlayer/softlayer-go/datatypes"
	"github.com/softlayer/softlayer-go/helpers/product"
	"github.com/softlayer/softlayer-go/services"
	"github.com/softlayer/softlayer-go/session"
	"github.com/softlayer/softlayer-go/sl"
)

// Upgrade a virtual guest to a specified set of features (e.g. cpu, ram).
// When the upgrade takes place can also be specified (`when`), but
// this is optional. The time set will be 'now' if left as nil.
// The features to upgrade are specified as the options used in
// GetProductPrices().
func UpgradeVirtualGuest(
	sess *session.Session,
	guest *datatypes.Virtual_Guest,
	options map[string]float64,
	when ...time.Time,
) (datatypes.Container_Product_Order_Receipt, error) {

	if guest.PrivateNetworkOnlyFlag == nil || guest.DedicatedAccountHostOnlyFlag == nil {
		service := services.GetVirtualGuestService(sess)
		guestForFlag, err := service.Id(*guest.Id).Mask("privateNetworkOnlyFlag,dedicatedAccountHostOnlyFlag").GetObject()
		if err != nil {
			return datatypes.Container_Product_Order_Receipt{}, err
		}

		guest.PrivateNetworkOnlyFlag = guestForFlag.PrivateNetworkOnlyFlag
		guest.DedicatedAccountHostOnlyFlag = guestForFlag.DedicatedAccountHostOnlyFlag
	}

	pkg, err := product.GetPackageByType(sess, "VIRTUAL_SERVER_INSTANCE")
	if err != nil {
		return datatypes.Container_Product_Order_Receipt{}, err
	}

	productItems, err := product.GetPackageProducts(sess, *pkg.Id)
	if err != nil {
		return datatypes.Container_Product_Order_Receipt{}, err
	}

	prices := product.SelectProductPricesByCategory(productItems, options, !*guest.PrivateNetworkOnlyFlag, !*guest.DedicatedAccountHostOnlyFlag)

	upgradeTime := time.Now().UTC().Format(time.RFC3339)
	if len(when) > 0 {
		upgradeTime = when[0].UTC().Format(time.RFC3339)
	}

	order := datatypes.Container_Product_Order_Virtual_Guest_Upgrade{
		Container_Product_Order_Virtual_Guest: datatypes.Container_Product_Order_Virtual_Guest{
			Container_Product_Order_Hardware_Server: datatypes.Container_Product_Order_Hardware_Server{
				Container_Product_Order: datatypes.Container_Product_Order{
					PackageId: pkg.Id,
					VirtualGuests: []datatypes.Virtual_Guest{
						*guest,
					},
					Prices: prices,
					Properties: []datatypes.Container_Product_Order_Property{
						{
							Name:  sl.String("MAINTENANCE_WINDOW"),
							Value: &upgradeTime,
						},
					},
				},
			},
		},
	}

	orderService := services.GetProductOrderService(sess)
	return orderService.PlaceOrder(&order, sl.Bool(false))
}
