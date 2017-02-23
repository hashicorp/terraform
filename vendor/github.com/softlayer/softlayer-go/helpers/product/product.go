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

package product

import (
	"fmt"

	"github.com/softlayer/softlayer-go/datatypes"
	"github.com/softlayer/softlayer-go/filter"
	"github.com/softlayer/softlayer-go/services"
	"github.com/softlayer/softlayer-go/session"
	"github.com/softlayer/softlayer-go/sl"
	"strings"
)

// CPUCategoryCode Category code for cpus
const CPUCategoryCode = "guest_core"

// MemoryCategoryCode Category code for Memory
const MemoryCategoryCode = "ram"

// NICSpeedCategoryCode Category code for NIC speed
const NICSpeedCategoryCode = "port_speed"

// DedicatedLoadBalancerCategoryCode Category code for Dedicated Load Balancer
const DedicatedLoadBalancerCategoryCode = "dedicated_load_balancer"

// ProxyLoadBalancerCategoryCode Category code for Shared local load balancer (proxy load balancer)
const ProxyLoadBalancerCategoryCode = "proxy_load_balancer"

// GetPackageByType Get the Product_Package which matches the specified
// package type
func GetPackageByType(
	sess *session.Session,
	packageType string,
	mask ...string,
) (datatypes.Product_Package, error) {

	objectMask := "id,name,description,isActive,type[keyName]"
	if len(mask) > 0 {
		objectMask = mask[0]
	}

	service := services.GetProductPackageService(sess)

	// Get package id
	packages, err := service.
		Mask(objectMask).
		Filter(
			filter.Build(
				filter.Path("type.keyName").Eq(packageType),
			),
		).
		Limit(1).
		GetAllObjects()
	if err != nil {
		return datatypes.Product_Package{}, err
	}

	packages = rejectOutletPackages(packages)

	if len(packages) == 0 {
		return datatypes.Product_Package{}, fmt.Errorf("No product packages found for %s", packageType)
	}

	return packages[0], nil
}

// rejectOutletPackages removes packages whose description or name contains the
// string "OUTLET".
func rejectOutletPackages(packages []datatypes.Product_Package) []datatypes.Product_Package {
	selected := []datatypes.Product_Package{}

	for _, pkg := range packages {
		if (pkg.Name == nil || !strings.Contains(*pkg.Name, "OUTLET")) &&
			(pkg.Description == nil || !strings.Contains(*pkg.Description, "OUTLET")) {

			selected = append(selected, pkg)
		}
	}

	return selected
}

// GetPackageProducts Get a list of product items for a specific product
// package ID
func GetPackageProducts(
	sess *session.Session,
	packageId int,
	mask ...string,
) ([]datatypes.Product_Item, error) {

	objectMask := "id,capacity,description,units,keyName,prices[id,categories[id,name,categoryCode]]"
	if len(mask) > 0 {
		objectMask = mask[0]
	}

	service := services.GetProductPackageService(sess)

	// Get product items for package id
	return service.
		Id(packageId).
		Mask(objectMask).
		GetItems()
}

// SelectProductPricesByCategory Get a list of Product_Item_Prices that
// match a specific set of price category code / product item
// capacity combinations.
// These combinations are passed as a map of strings (category code) mapped
// to float64 (capacity)
// For example, these are the options to specify an upgrade to 8 cpus and 32
// GB or memory:
// {"guest_core": 8.0, "ram": 32.0}
// public[0] checks type of network.
// public[1] checks type of cores.

func SelectProductPricesByCategory(
	productItems []datatypes.Product_Item,
	options map[string]float64,
	public ...bool,
) []datatypes.Product_Item_Price {

	forPublicNetwork := true
	if len(public) > 0 {
		forPublicNetwork = public[0]
	}

	// Check type of cores
	forPublicCores := true
	if len(public) > 1 {
		forPublicCores = public[1]
	}

	// Filter product items based on sets of category codes and capacity numbers
	prices := []datatypes.Product_Item_Price{}
	priceCheck := map[string]bool{}
	for _, productItem := range productItems {
		isPrivate := strings.HasPrefix(sl.Get(productItem.Description, "").(string), "Private")
		isPublic := strings.Contains(sl.Get(productItem.Description, "Public").(string), "Public")
		for _, category := range productItem.Prices[0].Categories {
			for categoryCode, capacity := range options {
				if _, ok := priceCheck[categoryCode]; ok {
					continue
				}

				if productItem.Capacity == nil {
					continue
				}

				if *category.CategoryCode != categoryCode {
					continue
				}

				if *productItem.Capacity != datatypes.Float64(capacity) {
					continue
				}

				// Logic taken from softlayer-python @ http://bit.ly/2bN9Gbu
				switch categoryCode {
				case CPUCategoryCode:
					if forPublicCores == isPrivate {
						continue
					}
				case NICSpeedCategoryCode:
					if forPublicNetwork != isPublic {
						continue
					}
				}

				prices = append(prices, productItem.Prices[0])
				priceCheck[categoryCode] = true
			}
		}
	}

	return prices
}
