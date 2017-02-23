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

/**
 * AUTOMATICALLY GENERATED CODE - DO NOT MODIFY
 */

package datatypes

// A Catalog is defined as a set of prices for products that SoftLayer offers for sale. These prices are organized into packages which represent the different servers and services that SoftLayer offers.
type Product_Catalog struct {
	Entity

	// A count of brands using this Catalog
	BrandCount *uint `json:"brandCount,omitempty" xmlrpc:"brandCount,omitempty"`

	// Brands using this Catalog
	Brands []Brand `json:"brands,omitempty" xmlrpc:"brands,omitempty"`

	// A count of packages available in this catalog
	PackageCount *uint `json:"packageCount,omitempty" xmlrpc:"packageCount,omitempty"`

	// Packages available in this catalog
	Packages []Product_Package `json:"packages,omitempty" xmlrpc:"packages,omitempty"`

	// A count of prices available in this catalog
	PriceCount *uint `json:"priceCount,omitempty" xmlrpc:"priceCount,omitempty"`

	// Prices available in this catalog
	Prices []Product_Item_Price `json:"prices,omitempty" xmlrpc:"prices,omitempty"`

	// A count of products available in catalog
	ProductCount *uint `json:"productCount,omitempty" xmlrpc:"productCount,omitempty"`

	// Products available in catalog
	Products []Product_Item `json:"products,omitempty" xmlrpc:"products,omitempty"`
}

// The SoftLayer_Product_Catalog_Item_Price type assigns an Item Price to a Catalog. This relation defines the composition of Item Prices in a Catalog.
type Product_Catalog_Item_Price struct {
	Entity

	// Catalog being assigned
	Catalog *Product_Catalog `json:"catalog,omitempty" xmlrpc:"catalog,omitempty"`

	// The id of the Catalog the Item Price is part of.
	CatalogId *int `json:"catalogId,omitempty" xmlrpc:"catalogId,omitempty"`

	// The time the Item Price was defined in the Catalog
	CreateDate *Time `json:"createDate,omitempty" xmlrpc:"createDate,omitempty"`

	// The time the Item Price was changed for the Catalog
	ModifyDate *Time `json:"modifyDate,omitempty" xmlrpc:"modifyDate,omitempty"`

	// Price being assigned
	Price *Product_Item_Price `json:"price,omitempty" xmlrpc:"price,omitempty"`

	// The id of the Item Price that is part of the Catalog.
	PriceId *int `json:"priceId,omitempty" xmlrpc:"priceId,omitempty"`
}

// The SoftLayer_Product_Item data type contains general information relating to a single SoftLayer product.
type Product_Item struct {
	Entity

	// A count of
	ActivePresaleEventCount *uint `json:"activePresaleEventCount,omitempty" xmlrpc:"activePresaleEventCount,omitempty"`

	// no documentation yet
	ActivePresaleEvents []Sales_Presale_Event `json:"activePresaleEvents,omitempty" xmlrpc:"activePresaleEvents,omitempty"`

	// A count of active usage based prices.
	ActiveUsagePriceCount *uint `json:"activeUsagePriceCount,omitempty" xmlrpc:"activeUsagePriceCount,omitempty"`

	// Active usage based prices.
	ActiveUsagePrices []Product_Item_Price `json:"activeUsagePrices,omitempty" xmlrpc:"activeUsagePrices,omitempty"`

	// A count of the attribute values for a product item. These are additional properties that give extra information about the product being sold.
	AttributeCount *uint `json:"attributeCount,omitempty" xmlrpc:"attributeCount,omitempty"`

	// The attribute values for a product item. These are additional properties that give extra information about the product being sold.
	Attributes []Product_Item_Attribute `json:"attributes,omitempty" xmlrpc:"attributes,omitempty"`

	// A count of attributes that govern when an item may no longer be available.
	AvailabilityAttributeCount *uint `json:"availabilityAttributeCount,omitempty" xmlrpc:"availabilityAttributeCount,omitempty"`

	// Attributes that govern when an item may no longer be available.
	AvailabilityAttributes []Product_Item_Attribute `json:"availabilityAttributes,omitempty" xmlrpc:"availabilityAttributes,omitempty"`

	// An item's special billing type, if applicable.
	BillingType *string `json:"billingType,omitempty" xmlrpc:"billingType,omitempty"`

	// An item's included products. Some items have other items included in them that we specifically detail. They are here called Bundled Items. An example is Plesk unlimited. It as a bundled item labeled 'SiteBuilder'. These are the SoftLayer_Product_Item_Bundles objects.
	Bundle []Product_Item_Bundles `json:"bundle,omitempty" xmlrpc:"bundle,omitempty"`

	// A count of an item's included products. Some items have other items included in them that we specifically detail. They are here called Bundled Items. An example is Plesk unlimited. It as a bundled item labeled 'SiteBuilder'. These are the SoftLayer_Product_Item_Bundles objects.
	BundleCount *uint `json:"bundleCount,omitempty" xmlrpc:"bundleCount,omitempty"`

	// Some Product Items have capacity information such as RAM and bandwidth, and others. This provides the numerical representation of the capacity given in the description of this product item.
	Capacity *Float64 `json:"capacity,omitempty" xmlrpc:"capacity,omitempty"`

	// This flag indicates that this product is restricted by a capacity on a related product.
	CapacityRestrictedProductFlag *bool `json:"capacityRestrictedProductFlag,omitempty" xmlrpc:"capacityRestrictedProductFlag,omitempty"`

	// An item's associated item categories.
	Categories []Product_Item_Category `json:"categories,omitempty" xmlrpc:"categories,omitempty"`

	// A count of an item's associated item categories.
	CategoryCount *uint `json:"categoryCount,omitempty" xmlrpc:"categoryCount,omitempty"`

	// A count of some product items have configuration templates which can be used to during provisioning of that product.
	ConfigurationTemplateCount *uint `json:"configurationTemplateCount,omitempty" xmlrpc:"configurationTemplateCount,omitempty"`

	// Some product items have configuration templates which can be used to during provisioning of that product.
	ConfigurationTemplates []Configuration_Template `json:"configurationTemplates,omitempty" xmlrpc:"configurationTemplates,omitempty"`

	// A count of an item's conflicts. For example, McAfee LinuxShield cannot be ordered with Windows. It was not meant for that operating system and as such is a conflict.
	ConflictCount *uint `json:"conflictCount,omitempty" xmlrpc:"conflictCount,omitempty"`

	// An item's conflicts. For example, McAfee LinuxShield cannot be ordered with Windows. It was not meant for that operating system and as such is a conflict.
	Conflicts []Product_Item_Resource_Conflict `json:"conflicts,omitempty" xmlrpc:"conflicts,omitempty"`

	// This flag indicates that this product is restricted by the number of cores on the compute instance. This is deprecated. Use [[SoftLayer_Product_Item/getCapacityRestrictedProductFlag|getCapacityRestrictedProductFlag]]
	CoreRestrictedItemFlag *bool `json:"coreRestrictedItemFlag,omitempty" xmlrpc:"coreRestrictedItemFlag,omitempty"`

	// A product's description
	Description *string `json:"description,omitempty" xmlrpc:"description,omitempty"`

	// Some product items have a downgrade path. This is the first product item in the downgrade path.
	DowngradeItem *Product_Item `json:"downgradeItem,omitempty" xmlrpc:"downgradeItem,omitempty"`

	// A count of some product items have a downgrade path. These are those product items.
	DowngradeItemCount *uint `json:"downgradeItemCount,omitempty" xmlrpc:"downgradeItemCount,omitempty"`

	// Some product items have a downgrade path. These are those product items.
	DowngradeItems []Product_Item `json:"downgradeItems,omitempty" xmlrpc:"downgradeItems,omitempty"`

	// A count of an item's category conflicts. For example, 10 Gbps redundant network functionality cannot be ordered with a secondary GPU and as such is a conflict.
	GlobalCategoryConflictCount *uint `json:"globalCategoryConflictCount,omitempty" xmlrpc:"globalCategoryConflictCount,omitempty"`

	// An item's category conflicts. For example, 10 Gbps redundant network functionality cannot be ordered with a secondary GPU and as such is a conflict.
	GlobalCategoryConflicts []Product_Item_Resource_Conflict `json:"globalCategoryConflicts,omitempty" xmlrpc:"globalCategoryConflicts,omitempty"`

	// The generic hardware component that this item represents.
	HardwareGenericComponentModel *Hardware_Component_Model_Generic `json:"hardwareGenericComponentModel,omitempty" xmlrpc:"hardwareGenericComponentModel,omitempty"`

	// no documentation yet
	HideFromPortalFlag *bool `json:"hideFromPortalFlag,omitempty" xmlrpc:"hideFromPortalFlag,omitempty"`

	// A product's internal identification number
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// An item's inventory status per datacenter.
	Inventory []Product_Package_Inventory `json:"inventory,omitempty" xmlrpc:"inventory,omitempty"`

	// A count of an item's inventory status per datacenter.
	InventoryCount *uint `json:"inventoryCount,omitempty" xmlrpc:"inventoryCount,omitempty"`

	// Flag to indicate the server product is engineered for a multi-server solution. (Deprecated)
	IsEngineeredServerProduct *bool `json:"isEngineeredServerProduct,omitempty" xmlrpc:"isEngineeredServerProduct,omitempty"`

	// An item's primary item category.
	ItemCategory *Product_Item_Category `json:"itemCategory,omitempty" xmlrpc:"itemCategory,omitempty"`

	// A products tax category internal identification number
	ItemTaxCategoryId *int `json:"itemTaxCategoryId,omitempty" xmlrpc:"itemTaxCategoryId,omitempty"`

	// A unique key name for the product.
	KeyName *string `json:"keyName,omitempty" xmlrpc:"keyName,omitempty"`

	// A count of an item's location conflicts. For example, Dual Path network functionality cannot be ordered in WDC and as such is a conflict.
	LocationConflictCount *uint `json:"locationConflictCount,omitempty" xmlrpc:"locationConflictCount,omitempty"`

	// An item's location conflicts. For example, Dual Path network functionality cannot be ordered in WDC and as such is a conflict.
	LocationConflicts []Product_Item_Resource_Conflict `json:"locationConflicts,omitempty" xmlrpc:"locationConflicts,omitempty"`

	// Detailed product description
	LongDescription *string `json:"longDescription,omitempty" xmlrpc:"longDescription,omitempty"`

	// no documentation yet
	ObjectStorageItemFlag *bool `json:"objectStorageItemFlag,omitempty" xmlrpc:"objectStorageItemFlag,omitempty"`

	// A count of a collection of all the SoftLayer_Product_Package(s) in which this item exists.
	PackageCount *uint `json:"packageCount,omitempty" xmlrpc:"packageCount,omitempty"`

	// A collection of all the SoftLayer_Product_Package(s) in which this item exists.
	Packages []Product_Package `json:"packages,omitempty" xmlrpc:"packages,omitempty"`

	// The number of cores that a processor has.
	PhysicalCoreCapacity *string `json:"physicalCoreCapacity,omitempty" xmlrpc:"physicalCoreCapacity,omitempty"`

	// A count of
	PresaleEventCount *uint `json:"presaleEventCount,omitempty" xmlrpc:"presaleEventCount,omitempty"`

	// no documentation yet
	PresaleEvents []Sales_Presale_Event `json:"presaleEvents,omitempty" xmlrpc:"presaleEvents,omitempty"`

	// A count of a product item's prices.
	PriceCount *uint `json:"priceCount,omitempty" xmlrpc:"priceCount,omitempty"`

	// A product item's prices.
	Prices []Product_Item_Price `json:"prices,omitempty" xmlrpc:"prices,omitempty"`

	// A count of if an item must be ordered with another item, it will have a requirement item here.
	RequirementCount *uint `json:"requirementCount,omitempty" xmlrpc:"requirementCount,omitempty"`

	// If an item must be ordered with another item, it will have a requirement item here.
	Requirements []Product_Item_Requirement `json:"requirements,omitempty" xmlrpc:"requirements,omitempty"`

	// The SoftLayer_Software_Description tied to this item. This will only be populated for software items.
	SoftwareDescription *Software_Description `json:"softwareDescription,omitempty" xmlrpc:"softwareDescription,omitempty"`

	// The unique identifier of the SoftLayer_Software_Description tied to this item.
	SoftwareDescriptionId *int `json:"softwareDescriptionId,omitempty" xmlrpc:"softwareDescriptionId,omitempty"`

	// An item's tax category, if applicable.
	TaxCategory *Product_Item_Tax_Category `json:"taxCategory,omitempty" xmlrpc:"taxCategory,omitempty"`

	// A count of third-party policy assignments for this product.
	ThirdPartyPolicyAssignmentCount *uint `json:"thirdPartyPolicyAssignmentCount,omitempty" xmlrpc:"thirdPartyPolicyAssignmentCount,omitempty"`

	// Third-party policy assignments for this product.
	ThirdPartyPolicyAssignments []Product_Item_Policy_Assignment `json:"thirdPartyPolicyAssignments,omitempty" xmlrpc:"thirdPartyPolicyAssignments,omitempty"`

	// The 3rd party vendor for a support subscription item. (Deprecated)
	ThirdPartySupportVendor *string `json:"thirdPartySupportVendor,omitempty" xmlrpc:"thirdPartySupportVendor,omitempty"`

	// The total number of physical processing cores (excluding virtual cores / hyperthreads) for this server.
	TotalPhysicalCoreCapacity *int `json:"totalPhysicalCoreCapacity,omitempty" xmlrpc:"totalPhysicalCoreCapacity,omitempty"`

	// Shows the total number of cores. This is deprecated. Use [[SoftLayer_Product_Item/getCapacity|getCapacity]] for guest_core products and [[SoftLayer_Product_Item/getTotalPhysicalCoreCapacity|getTotalPhysicalCoreCapacity]] for server products
	TotalPhysicalCoreCount *int `json:"totalPhysicalCoreCount,omitempty" xmlrpc:"totalPhysicalCoreCount,omitempty"`

	// The total number of processors for this server.
	TotalProcessorCapacity *int `json:"totalProcessorCapacity,omitempty" xmlrpc:"totalProcessorCapacity,omitempty"`

	// The unit of measurement that a product item is measured in.
	Units *string `json:"units,omitempty" xmlrpc:"units,omitempty"`

	// Some product items have an upgrade path. This is the next product item in the upgrade path.
	UpgradeItem *Product_Item `json:"upgradeItem,omitempty" xmlrpc:"upgradeItem,omitempty"`

	// A count of some product items have an upgrade path. These are those upgrade product items.
	UpgradeItemCount *uint `json:"upgradeItemCount,omitempty" xmlrpc:"upgradeItemCount,omitempty"`

	// A products upgrade item's internal identification number
	UpgradeItemId *int `json:"upgradeItemId,omitempty" xmlrpc:"upgradeItemId,omitempty"`

	// Some product items have an upgrade path. These are those upgrade product items.
	UpgradeItems []Product_Item `json:"upgradeItems,omitempty" xmlrpc:"upgradeItems,omitempty"`
}

// The [[SoftLayer_Product_Item_Attribute]] data type allows us to describe a [[SoftLayer_Product_Item]] by attaching specific attributes, which may dictate how it interacts with other products and services. Most, if not all, of these attributes are geared towards internal usage, so customers should rarely be concerned with them.
type Product_Item_Attribute struct {
	Entity

	// This represents the attribute type of this product attribute.
	AttributeType *Product_Item_Attribute_Type `json:"attributeType,omitempty" xmlrpc:"attributeType,omitempty"`

	// This represents the attribute type's key name of this product attribute.
	AttributeTypeKeyName *string `json:"attributeTypeKeyName,omitempty" xmlrpc:"attributeTypeKeyName,omitempty"`

	// This is the primary key value for the product attribute.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// This represents the product that an attribute is tied to.
	Item *Product_Item `json:"item,omitempty" xmlrpc:"item,omitempty"`

	// This is a foreign key value for the [[SoftLayer_Product_Item_Attribute_Type]].
	ItemAttributeTypeId *int `json:"itemAttributeTypeId,omitempty" xmlrpc:"itemAttributeTypeId,omitempty"`

	// This is a foreign key value for the [[SoftLayer_Product_Item]].
	ItemId *int `json:"itemId,omitempty" xmlrpc:"itemId,omitempty"`

	// This is the value for the attribute.
	Value *string `json:"value,omitempty" xmlrpc:"value,omitempty"`
}

// The [[SoftLayer_Product_Item_Attribute_Type]] data type defines the available type of product attributes that are available. This allows for convenient reference to a [[SoftLayer_Product_Item_Attribute|product attribute]] by a unique key name value.
type Product_Item_Attribute_Type struct {
	Entity

	// This is the unique identifier of the attribute type.
	KeyName *string `json:"keyName,omitempty" xmlrpc:"keyName,omitempty"`

	// This is the user-friendly readable name of the attribute type.
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`
}

// The SoftLayer_Product_Item_Billing_Type data type models special billing types for non-monthly billed items in the SoftLayer product catalog.
type Product_Item_Billing_Type struct {
	Entity

	// A keyword describing a SoftLayer product item billing type.
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`
}

// The SoftLayer_Product_Item_Bundles contains item to price cross references Relates a category, price and item to a bundle.  Match bundle ids to see all items and prices in a particular bundle.
type Product_Item_Bundles struct {
	Entity

	// Item in bundle.
	BundleItem *Product_Item `json:"bundleItem,omitempty" xmlrpc:"bundleItem,omitempty"`

	// Identifier for bundle.
	BundleItemId *int `json:"bundleItemId,omitempty" xmlrpc:"bundleItemId,omitempty"`

	// Category bundle falls in.
	Category *Product_Item_Category `json:"category,omitempty" xmlrpc:"category,omitempty"`

	// Identifier for record.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// Price of item in bundle
	ItemPrice *Product_Item_Price `json:"itemPrice,omitempty" xmlrpc:"itemPrice,omitempty"`

	// Identifier for price.
	ItemPriceId *int `json:"itemPriceId,omitempty" xmlrpc:"itemPriceId,omitempty"`
}

// The SoftLayer_Product_Item_Category data type contains general category information for prices.
type Product_Item_Category struct {
	Entity

	// A count of the billing items associated with an account that share a category code with an item category's category code.
	BillingItemCount *uint `json:"billingItemCount,omitempty" xmlrpc:"billingItemCount,omitempty"`

	// The billing items associated with an account that share a category code with an item category's category code.
	BillingItems []Billing_Item `json:"billingItems,omitempty" xmlrpc:"billingItems,omitempty"`

	// The code used to identify this category.
	CategoryCode *string `json:"categoryCode,omitempty" xmlrpc:"categoryCode,omitempty"`

	// This invoice item's "item category group".
	Group *Product_Item_Category_Group `json:"group,omitempty" xmlrpc:"group,omitempty"`

	// A count of a collection of service offering category groups. Each group contains a collection of items associated with this category.
	GroupCount *uint `json:"groupCount,omitempty" xmlrpc:"groupCount,omitempty"`

	// A collection of service offering category groups. Each group contains a collection of items associated with this category.
	Groups []Product_Package_Item_Category_Group `json:"groups,omitempty" xmlrpc:"groups,omitempty"`

	// identifier for category.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// The friendly, descriptive name of the category as seen on the order forms and on invoices.
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`

	// A count of any unique options associated with an item category.
	OrderOptionCount *uint `json:"orderOptionCount,omitempty" xmlrpc:"orderOptionCount,omitempty"`

	// Any unique options associated with an item category.
	OrderOptions []Product_Item_Category_Order_Option_Type `json:"orderOptions,omitempty" xmlrpc:"orderOptions,omitempty"`

	// A count of a list of configuration available in this category.'
	PackageConfigurationCount *uint `json:"packageConfigurationCount,omitempty" xmlrpc:"packageConfigurationCount,omitempty"`

	// A list of configuration available in this category.'
	PackageConfigurations []Product_Package_Order_Configuration `json:"packageConfigurations,omitempty" xmlrpc:"packageConfigurations,omitempty"`

	// A count of a list of preset configurations this category is used in.'
	PresetConfigurationCount *uint `json:"presetConfigurationCount,omitempty" xmlrpc:"presetConfigurationCount,omitempty"`

	// A list of preset configurations this category is used in.'
	PresetConfigurations []Product_Package_Preset_Configuration `json:"presetConfigurations,omitempty" xmlrpc:"presetConfigurations,omitempty"`

	// Quantity that can be ordered. If 0, it will inherit the quantity from the server quantity ordered. Otherwise it can be specified with the order separately
	QuantityLimit *int `json:"quantityLimit,omitempty" xmlrpc:"quantityLimit,omitempty"`

	// A count of the questions that are associated with an item category.
	QuestionCount *uint `json:"questionCount,omitempty" xmlrpc:"questionCount,omitempty"`

	// A count of the question references that are associated with an item category.
	QuestionReferenceCount *uint `json:"questionReferenceCount,omitempty" xmlrpc:"questionReferenceCount,omitempty"`

	// The question references that are associated with an item category.
	QuestionReferences []Product_Item_Category_Question_Xref `json:"questionReferences,omitempty" xmlrpc:"questionReferences,omitempty"`

	// The questions that are associated with an item category.
	Questions []Product_Item_Category_Question `json:"questions,omitempty" xmlrpc:"questions,omitempty"`
}

// The SoftLayer_Product_Item_Category_Group data type contains general category group information.
type Product_Item_Category_Group struct {
	Entity

	// identifier for category group.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// The friendly, descriptive name of the category group as seen on the order forms and on invoices.
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`
}

// The SoftLayer_Product_Item_Category_Order_Option_Type data type contains options that can be applied to orders for prices.
type Product_Item_Category_Order_Option_Type struct {
	Entity

	// An item category order type's description.
	Description *string `json:"description,omitempty" xmlrpc:"description,omitempty"`

	// An item category order type's unique identifier.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// A simple description for an item category order type.
	Keyname *string `json:"keyname,omitempty" xmlrpc:"keyname,omitempty"`

	// An item category order type's name.
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`

	// The value of the item category type's option.
	Value *string `json:"value,omitempty" xmlrpc:"value,omitempty"`
}

// The SoftLayer_Product_Item_Category_Question data type represents a single question to be answered by an end user.  The question may or may not be required which can be located by looking at the 'required' property on the item category references.  The answerValueExpression property is a regular expression that is used to validate the answer to the question.  The description and valueExample properties can be used to get an idea of the type of answer that should be provided.
type Product_Item_Category_Question struct {
	Entity

	// The type of answer expected.
	AnswerValueExpression *string `json:"answerValueExpression,omitempty" xmlrpc:"answerValueExpression,omitempty"`

	// The description for the question.
	Description *string `json:"description,omitempty" xmlrpc:"description,omitempty"`

	// The type of field that should be used in an HTML form to accept an answer from an end user.
	FieldType *Product_Item_Category_Question_Field_Type `json:"fieldType,omitempty" xmlrpc:"fieldType,omitempty"`

	// The type of field to use.
	FieldTypeId *int `json:"fieldTypeId,omitempty" xmlrpc:"fieldTypeId,omitempty"`

	// identifier for category.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// A count of the link between an item category and an item category question.
	ItemCategoryReferenceCount *uint `json:"itemCategoryReferenceCount,omitempty" xmlrpc:"itemCategoryReferenceCount,omitempty"`

	// The link between an item category and an item category question.
	ItemCategoryReferences []Product_Item_Category_Question_Xref `json:"itemCategoryReferences,omitempty" xmlrpc:"itemCategoryReferences,omitempty"`

	// The keyname for the question.
	KeyName *string `json:"keyName,omitempty" xmlrpc:"keyName,omitempty"`

	// The question for the category.
	Question *string `json:"question,omitempty" xmlrpc:"question,omitempty"`

	// An example and/or explanation of what the answer for the question is expected to look like.
	ValueExample *string `json:"valueExample,omitempty" xmlrpc:"valueExample,omitempty"`
}

// The SoftLayer_Product_Item_Category_Question_Field_Type data type represents the recommended type of field that should be rendered on an HTML form.
type Product_Item_Category_Question_Field_Type struct {
	Entity

	// Identifier for the question type.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// Keyname for the question field type.
	KeyName *string `json:"keyName,omitempty" xmlrpc:"keyName,omitempty"`

	// Short name for the question field type.
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`
}

// The SoftLayer_Product_Item_Category_Question_Xref data type represents a link between an item category and an item category question.  It also contains a 'required' field that designates if the question is required to be answered for the given item category.
type Product_Item_Category_Question_Xref struct {
	Entity

	// Identifier for category question xref record.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// The product item category that this reference points to.
	ItemCategory *Product_Item_Category `json:"itemCategory,omitempty" xmlrpc:"itemCategory,omitempty"`

	// Identifier for item category.
	ItemCategoryId *int `json:"itemCategoryId,omitempty" xmlrpc:"itemCategoryId,omitempty"`

	// Identifier for the question.
	LocationId *int `json:"locationId,omitempty" xmlrpc:"locationId,omitempty"`

	// The item category question that this reference points to.
	Question *Product_Item_Category_Question `json:"question,omitempty" xmlrpc:"question,omitempty"`

	// Identifier for the question.
	QuestionId *int `json:"questionId,omitempty" xmlrpc:"questionId,omitempty"`

	// Flag to indicate whether an answer is required for the question..
	Required *bool `json:"required,omitempty" xmlrpc:"required,omitempty"`
}

// no documentation yet
type Product_Item_Link_ThePlanet struct {
	Entity

	// no documentation yet
	Item *Product_Item `json:"item,omitempty" xmlrpc:"item,omitempty"`

	// no documentation yet
	ServiceProvider *Service_Provider `json:"serviceProvider,omitempty" xmlrpc:"serviceProvider,omitempty"`
}

// Represents the assignment of a policy to a product. The existence of a record means that the associated product is subject to the terms defined in the document content of the policy.
type Product_Item_Policy_Assignment struct {
	Entity

	// no documentation yet
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// The name of the assigned policy.
	PolicyName *string `json:"policyName,omitempty" xmlrpc:"policyName,omitempty"`

	// The [[SoftLayer_Product_Item]] for this policy assignment.
	Product *Product_Item `json:"product,omitempty" xmlrpc:"product,omitempty"`

	// no documentation yet
	ProductId *int `json:"productId,omitempty" xmlrpc:"productId,omitempty"`
}

// The SoftLayer_Product_Item_Price data type contains general information relating to a single SoftLayer product item price. You can find out what packages each price is in as well as which category under which this price is sold. All prices are returned in floating point values measured in US Dollars ($USD).
type Product_Item_Price struct {
	Entity

	// A count of the account that the item price is restricted to.
	AccountRestrictionCount *uint `json:"accountRestrictionCount,omitempty" xmlrpc:"accountRestrictionCount,omitempty"`

	// The account that the item price is restricted to.
	AccountRestrictions []Product_Item_Price_Account_Restriction `json:"accountRestrictions,omitempty" xmlrpc:"accountRestrictions,omitempty"`

	// A count of
	AttributeCount *uint `json:"attributeCount,omitempty" xmlrpc:"attributeCount,omitempty"`

	// no documentation yet
	Attributes []Product_Item_Price_Attribute `json:"attributes,omitempty" xmlrpc:"attributes,omitempty"`

	// Whether the price is for Big Data OS/Journal disks only. (Deprecated)
	BigDataOsJournalDiskFlag *bool `json:"bigDataOsJournalDiskFlag,omitempty" xmlrpc:"bigDataOsJournalDiskFlag,omitempty"`

	// A count of cross reference for bundles
	BundleReferenceCount *uint `json:"bundleReferenceCount,omitempty" xmlrpc:"bundleReferenceCount,omitempty"`

	// cross reference for bundles
	BundleReferences []Product_Item_Bundles `json:"bundleReferences,omitempty" xmlrpc:"bundleReferences,omitempty"`

	// The maximum capacity value for which this price is suitable.
	CapacityRestrictionMaximum *string `json:"capacityRestrictionMaximum,omitempty" xmlrpc:"capacityRestrictionMaximum,omitempty"`

	// The minimum capacity value for which this price is suitable.
	CapacityRestrictionMinimum *string `json:"capacityRestrictionMinimum,omitempty" xmlrpc:"capacityRestrictionMinimum,omitempty"`

	// The type of capacity restriction by which this price must abide.
	CapacityRestrictionType *string `json:"capacityRestrictionType,omitempty" xmlrpc:"capacityRestrictionType,omitempty"`

	// All categories which this item is a member.
	Categories []Product_Item_Category `json:"categories,omitempty" xmlrpc:"categories,omitempty"`

	// A count of all categories which this item is a member.
	CategoryCount *uint `json:"categoryCount,omitempty" xmlrpc:"categoryCount,omitempty"`

	// This flag is used by the [[SoftLayer_Hardware::getUpgradeItems|getUpgradeItems]] method to indicate if a product price is used for the current billing item.
	CurrentPriceFlag *bool `json:"currentPriceFlag,omitempty" xmlrpc:"currentPriceFlag,omitempty"`

	// Whether this price defines a software license for its product item.
	DefinedSoftwareLicenseFlag *bool `json:"definedSoftwareLicenseFlag,omitempty" xmlrpc:"definedSoftwareLicenseFlag,omitempty"`

	// The hourly price for this item, should this item be part of an hourly pricing package.
	HourlyRecurringFee *Float64 `json:"hourlyRecurringFee,omitempty" xmlrpc:"hourlyRecurringFee,omitempty"`

	// The unique identifier of a Product Item Price.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// An item price's inventory status per datacenter.
	Inventory []Product_Package_Inventory `json:"inventory,omitempty" xmlrpc:"inventory,omitempty"`

	// A count of an item price's inventory status per datacenter.
	InventoryCount *uint `json:"inventoryCount,omitempty" xmlrpc:"inventoryCount,omitempty"`

	// The product item a price is tied to.
	Item *Product_Item `json:"item,omitempty" xmlrpc:"item,omitempty"`

	// The unique identifier for a product Item
	ItemId *int `json:"itemId,omitempty" xmlrpc:"itemId,omitempty"`

	// The labor fee for a product item price.
	LaborFee *Float64 `json:"laborFee,omitempty" xmlrpc:"laborFee,omitempty"`

	// The id of the [[SoftLayer_Location_Group_Pricing]] that this price is part of. If set to null, the price is considered a standard price, which can be used with any location when ordering.
	//
	// During order [[SoftLayer_Product_Order/verifyOrder|verification]] and [[SoftLayer_Product_Order/placeOrder|placement]], if a standard price is used, that price may be replaced with a location based price, which does not have this property set to null. The location based price must be part of a [[SoftLayer_Location_Group_Pricing]] that has the location being ordered in order for this to happen.
	LocationGroupId *int `json:"locationGroupId,omitempty" xmlrpc:"locationGroupId,omitempty"`

	// On sale flag.
	OnSaleFlag *bool `json:"onSaleFlag,omitempty" xmlrpc:"onSaleFlag,omitempty"`

	// The one time fee for a product item price.
	OneTimeFee *Float64 `json:"oneTimeFee,omitempty" xmlrpc:"oneTimeFee,omitempty"`

	// A price's total tax amount of the one time fees (oneTimeFee, laborFee, and setupFee). This is only populated after the order is verified via SoftLayer_Product_Order::verifyOrder()
	OneTimeFeeTax *Float64 `json:"oneTimeFeeTax,omitempty" xmlrpc:"oneTimeFeeTax,omitempty"`

	// Order options for the category that this price is associated with.
	OrderOptions []Product_Item_Category_Order_Option_Type `json:"orderOptions,omitempty" xmlrpc:"orderOptions,omitempty"`

	// A count of
	OrderPremiumCount *uint `json:"orderPremiumCount,omitempty" xmlrpc:"orderPremiumCount,omitempty"`

	// no documentation yet
	OrderPremiums []Product_Item_Price_Premium `json:"orderPremiums,omitempty" xmlrpc:"orderPremiums,omitempty"`

	// A count of a price's packages under which this item is sold.
	PackageCount *uint `json:"packageCount,omitempty" xmlrpc:"packageCount,omitempty"`

	// A count of cross reference for packages
	PackageReferenceCount *uint `json:"packageReferenceCount,omitempty" xmlrpc:"packageReferenceCount,omitempty"`

	// cross reference for packages
	PackageReferences []Product_Package_Item_Prices `json:"packageReferences,omitempty" xmlrpc:"packageReferences,omitempty"`

	// A price's packages under which this item is sold.
	Packages []Product_Package `json:"packages,omitempty" xmlrpc:"packages,omitempty"`

	// A count of a list of preset configurations this price is used in.'
	PresetConfigurationCount *uint `json:"presetConfigurationCount,omitempty" xmlrpc:"presetConfigurationCount,omitempty"`

	// A list of preset configurations this price is used in.'
	PresetConfigurations []Product_Package_Preset_Configuration `json:"presetConfigurations,omitempty" xmlrpc:"presetConfigurations,omitempty"`

	// The pricing location group that this price is applicable for. Prices that have a pricing location group will only be available for ordering with the locations specified on the location group.
	PricingLocationGroup *Location_Group_Pricing `json:"pricingLocationGroup,omitempty" xmlrpc:"pricingLocationGroup,omitempty"`

	// A recurring fee is a fee that happens every billing period. This fee is represented as a floating point decimal in US dollars ($USD).
	ProratedRecurringFee *Float64 `json:"proratedRecurringFee,omitempty" xmlrpc:"proratedRecurringFee,omitempty"`

	// A price's tax amount of the recurring fee. This is only populated after the order is verified via SoftLayer_Product_Order::verifyOrder()
	ProratedRecurringFeeTax *Float64 `json:"proratedRecurringFeeTax,omitempty" xmlrpc:"proratedRecurringFeeTax,omitempty"`

	// no documentation yet
	Quantity *int `json:"quantity,omitempty" xmlrpc:"quantity,omitempty"`

	// A recurring fee is a fee that happens every billing period. This fee is represented as a floating point decimal in US dollars ($USD).
	RecurringFee *Float64 `json:"recurringFee,omitempty" xmlrpc:"recurringFee,omitempty"`

	// A price's tax amount of the recurring fee. This is only populated after the order is verified via SoftLayer_Product_Order::verifyOrder()
	RecurringFeeTax *Float64 `json:"recurringFeeTax,omitempty" xmlrpc:"recurringFeeTax,omitempty"`

	// The number of server cores required to order this item. This is deprecated. Use [[SoftLayer_Product_Item_Price/getCapacityRestrictionMinimum|getCapacityRestrictionMinimum]] and [[SoftLayer_Product_Item_Price/getCapacityRestrictionMaximum|getCapacityRestrictionMaximum]]
	RequiredCoreCount *int `json:"requiredCoreCount,omitempty" xmlrpc:"requiredCoreCount,omitempty"`

	// The setup fee associated with a product item price.
	SetupFee *Float64 `json:"setupFee,omitempty" xmlrpc:"setupFee,omitempty"`

	// Used for ordering items on sales orders.
	Sort *int `json:"sort,omitempty" xmlrpc:"sort,omitempty"`

	// The rate for a usage based item
	UsageRate *Float64 `json:"usageRate,omitempty" xmlrpc:"usageRate,omitempty"`
}

// The SoftLayer_Product_Item_Price data type gives more information about the item price restrictions.  An item price may be restricted to one or more accounts. If the item price is restricted to an account, only that account will see the restriction details.
type Product_Item_Price_Account_Restriction struct {
	Entity

	// The account the item price is restricted to.
	Account *Account `json:"account,omitempty" xmlrpc:"account,omitempty"`

	// The account id for the item price account restriction.
	AccountId *int `json:"accountId,omitempty" xmlrpc:"accountId,omitempty"`

	// The unique identifier for the item price account restriction.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// The item price that has the account restriction.
	ItemPrice *Product_Item_Price `json:"itemPrice,omitempty" xmlrpc:"itemPrice,omitempty"`

	// The item price id for the item price account restriction.
	ItemPriceId *int `json:"itemPriceId,omitempty" xmlrpc:"itemPriceId,omitempty"`
}

// no documentation yet
type Product_Item_Price_Attribute struct {
	Entity

	// no documentation yet
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// no documentation yet
	ItemPrice *Product_Item_Price `json:"itemPrice,omitempty" xmlrpc:"itemPrice,omitempty"`

	// no documentation yet
	ItemPriceAttributeType *Product_Item_Price_Attribute_Type `json:"itemPriceAttributeType,omitempty" xmlrpc:"itemPriceAttributeType,omitempty"`

	// no documentation yet
	ItemPriceAttributeTypeId *int `json:"itemPriceAttributeTypeId,omitempty" xmlrpc:"itemPriceAttributeTypeId,omitempty"`

	// no documentation yet
	ItemPriceId *int `json:"itemPriceId,omitempty" xmlrpc:"itemPriceId,omitempty"`

	// no documentation yet
	Value *string `json:"value,omitempty" xmlrpc:"value,omitempty"`
}

// no documentation yet
type Product_Item_Price_Attribute_Type struct {
	Entity

	// no documentation yet
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// no documentation yet
	Keyname *string `json:"keyname,omitempty" xmlrpc:"keyname,omitempty"`
}

// no documentation yet
type Product_Item_Price_Premium struct {
	Entity

	// no documentation yet
	HourlyModifier *Float64 `json:"hourlyModifier,omitempty" xmlrpc:"hourlyModifier,omitempty"`

	// no documentation yet
	ItemPrice *Product_Item_Price `json:"itemPrice,omitempty" xmlrpc:"itemPrice,omitempty"`

	// no documentation yet
	ItemPriceId *int `json:"itemPriceId,omitempty" xmlrpc:"itemPriceId,omitempty"`

	// no documentation yet
	Location *Location `json:"location,omitempty" xmlrpc:"location,omitempty"`

	// no documentation yet
	LocationId *int `json:"locationId,omitempty" xmlrpc:"locationId,omitempty"`

	// no documentation yet
	MonthlyModifier *Float64 `json:"monthlyModifier,omitempty" xmlrpc:"monthlyModifier,omitempty"`

	// no documentation yet
	Package *Product_Package `json:"package,omitempty" xmlrpc:"package,omitempty"`

	// no documentation yet
	PackageId *int `json:"packageId,omitempty" xmlrpc:"packageId,omitempty"`
}

// The SoftLayer_Product_Item_Requirement data type contains information relating to what requirements, if any, exist for an item. The requiredItemId local property is the item id that is required.
type Product_Item_Requirement struct {
	Entity

	// Identifier for this record.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// Item requirement applies to.
	Item *Product_Item `json:"item,omitempty" xmlrpc:"item,omitempty"`

	// This is the id of the item affected by the requirement.
	ItemId *int `json:"itemId,omitempty" xmlrpc:"itemId,omitempty"`

	// This is a custom message to display to the user when this requirement shortfall arises.
	Message *string `json:"message,omitempty" xmlrpc:"message,omitempty"`

	// The product containing the requirement.
	Product *Product_Item `json:"product,omitempty" xmlrpc:"product,omitempty"`

	// This is the id of the item required.
	RequiredItemId *int `json:"requiredItemId,omitempty" xmlrpc:"requiredItemId,omitempty"`
}

// no documentation yet
type Product_Item_Resource_Conflict struct {
	Entity

	// no documentation yet
	Item *Product_Item `json:"item,omitempty" xmlrpc:"item,omitempty"`

	// The unique identifier of the item that contains the conflict.
	ItemId *int `json:"itemId,omitempty" xmlrpc:"itemId,omitempty"`

	// An optional conflict message.
	Message *string `json:"message,omitempty" xmlrpc:"message,omitempty"`

	// no documentation yet
	Package *Product_Package `json:"package,omitempty" xmlrpc:"package,omitempty"`

	// The unique identifier of the service offering that is associated with the conflict.
	PackageId *int `json:"packageId,omitempty" xmlrpc:"packageId,omitempty"`

	// The unique identifier of the conflicting type.
	ResourceTableId *int `json:"resourceTableId,omitempty" xmlrpc:"resourceTableId,omitempty"`
}

// no documentation yet
type Product_Item_Resource_Conflict_Item struct {
	Product_Item_Resource_Conflict

	// A product item that conflicts with another product item.
	Resource *Product_Item `json:"resource,omitempty" xmlrpc:"resource,omitempty"`
}

// no documentation yet
type Product_Item_Resource_Conflict_Item_Category struct {
	Product_Item_Resource_Conflict

	// An item category that conflicts with a product item.
	Resource *Product_Item_Category `json:"resource,omitempty" xmlrpc:"resource,omitempty"`
}

// no documentation yet
type Product_Item_Resource_Conflict_Location struct {
	Product_Item_Resource_Conflict

	// A location that conflicts with a product item.
	Resource *Location `json:"resource,omitempty" xmlrpc:"resource,omitempty"`
}

// The SoftLayer_Product_Item_Tax_Category data type contains the tax categories that are associated with products.
type Product_Item_Tax_Category struct {
	Entity

	// An internal identifier for each tax category.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// A count of
	ItemCount *uint `json:"itemCount,omitempty" xmlrpc:"itemCount,omitempty"`

	// no documentation yet
	Items []Product_Item `json:"items,omitempty" xmlrpc:"items,omitempty"`

	// The name of the tax category.
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`

	// The status of the tax category.
	StatusFlag *int `json:"statusFlag,omitempty" xmlrpc:"statusFlag,omitempty"`
}

// no documentation yet
type Product_Order struct {
	Entity
}

// The SoftLayer_Product_Package data type contains information about packages from which orders can be generated. Packages contain general information regarding what is in them, where they are currently sold, availability, and pricing.
type Product_Package struct {
	Entity

	// The results from this call are similar to [[SoftLayer_Product_Package/getCategories|getCategories]], but these ONLY include account-restricted prices. Not all accounts have restricted pricing.
	AccountRestrictedCategories []Product_Item_Category `json:"accountRestrictedCategories,omitempty" xmlrpc:"accountRestrictedCategories,omitempty"`

	// A count of the results from this call are similar to [[SoftLayer_Product_Package/getCategories|getCategories]], but these ONLY include account-restricted prices. Not all accounts have restricted pricing.
	AccountRestrictedCategoryCount *uint `json:"accountRestrictedCategoryCount,omitempty" xmlrpc:"accountRestrictedCategoryCount,omitempty"`

	// The flag to indicate if there are any restricted prices in a package for the currently-active account.
	AccountRestrictedPricesFlag *bool `json:"accountRestrictedPricesFlag,omitempty" xmlrpc:"accountRestrictedPricesFlag,omitempty"`

	// A count of the available preset configurations for this package.
	ActivePresetCount *uint `json:"activePresetCount,omitempty" xmlrpc:"activePresetCount,omitempty"`

	// The available preset configurations for this package.
	ActivePresets []Product_Package_Preset `json:"activePresets,omitempty" xmlrpc:"activePresets,omitempty"`

	// A count of a collection of valid RAM items available for purchase in this package.
	ActiveRamItemCount *uint `json:"activeRamItemCount,omitempty" xmlrpc:"activeRamItemCount,omitempty"`

	// A collection of valid RAM items available for purchase in this package.
	ActiveRamItems []Product_Item `json:"activeRamItems,omitempty" xmlrpc:"activeRamItems,omitempty"`

	// A count of a collection of valid server items available for purchase in this package.
	ActiveServerItemCount *uint `json:"activeServerItemCount,omitempty" xmlrpc:"activeServerItemCount,omitempty"`

	// A collection of valid server items available for purchase in this package.
	ActiveServerItems []Product_Item `json:"activeServerItems,omitempty" xmlrpc:"activeServerItems,omitempty"`

	// A count of a collection of valid software items available for purchase in this package.
	ActiveSoftwareItemCount *uint `json:"activeSoftwareItemCount,omitempty" xmlrpc:"activeSoftwareItemCount,omitempty"`

	// A collection of valid software items available for purchase in this package.
	ActiveSoftwareItems []Product_Item `json:"activeSoftwareItems,omitempty" xmlrpc:"activeSoftwareItems,omitempty"`

	// A count of a collection of [[SoftLayer_Product_Item_Price]] objects for pay-as-you-go usage.
	ActiveUsagePriceCount *uint `json:"activeUsagePriceCount,omitempty" xmlrpc:"activeUsagePriceCount,omitempty"`

	// A collection of [[SoftLayer_Product_Item_Price]] objects for pay-as-you-go usage.
	ActiveUsagePrices []Product_Item_Price `json:"activeUsagePrices,omitempty" xmlrpc:"activeUsagePrices,omitempty"`

	// This flag indicates that the package is an additional service.
	AdditionalServiceFlag *bool `json:"additionalServiceFlag,omitempty" xmlrpc:"additionalServiceFlag,omitempty"`

	// A count of
	AttributeCount *uint `json:"attributeCount,omitempty" xmlrpc:"attributeCount,omitempty"`

	// no documentation yet
	Attributes []Product_Package_Attribute `json:"attributes,omitempty" xmlrpc:"attributes,omitempty"`

	// A count of a collection of valid locations for this package. (Deprecated - Use [[SoftLayer_Product_Package/getRegions|getRegions]])
	AvailableLocationCount *uint `json:"availableLocationCount,omitempty" xmlrpc:"availableLocationCount,omitempty"`

	// A collection of valid locations for this package. (Deprecated - Use [[SoftLayer_Product_Package/getRegions|getRegions]])
	AvailableLocations []Product_Package_Locations `json:"availableLocations,omitempty" xmlrpc:"availableLocations,omitempty"`

	// The maximum number of available disk storage units associated with the servers in a package.
	AvailableStorageUnits *uint `json:"availableStorageUnits,omitempty" xmlrpc:"availableStorageUnits,omitempty"`

	// This is a collection of categories ([[SoftLayer_Product_Item_Category]]) associated with a package which can be used for ordering. These categories have several objects prepopulated which are useful when determining the available products for purchase. The categories contain groups ([[SoftLayer_Product_Package_Item_Category_Group]]) that organize the products and prices by similar features. For example, operating systems will be grouped by their manufacturer and virtual server disks will be grouped by their disk type (SAN vs. local). Each group will contain prices ([[SoftLayer_Product_Item_Price]]) which you can use determine the cost of each product. Each price has a product ([[SoftLayer_Product_Item]]) which provides the name and other useful information about the server, service or software you may purchase.
	Categories []Product_Item_Category `json:"categories,omitempty" xmlrpc:"categories,omitempty"`

	// The item categories associated with a package, including information detailing which item categories are required as part of a SoftLayer product order.
	Configuration []Product_Package_Order_Configuration `json:"configuration,omitempty" xmlrpc:"configuration,omitempty"`

	// A count of the item categories associated with a package, including information detailing which item categories are required as part of a SoftLayer product order.
	ConfigurationCount *uint `json:"configurationCount,omitempty" xmlrpc:"configurationCount,omitempty"`

	// A count of a collection of valid RAM items available for purchase in this package.
	DefaultRamItemCount *uint `json:"defaultRamItemCount,omitempty" xmlrpc:"defaultRamItemCount,omitempty"`

	// A collection of valid RAM items available for purchase in this package.
	DefaultRamItems []Product_Item `json:"defaultRamItems,omitempty" xmlrpc:"defaultRamItems,omitempty"`

	// A count of the package that represents a multi-server solution. (Deprecated)
	DeploymentCount *uint `json:"deploymentCount,omitempty" xmlrpc:"deploymentCount,omitempty"`

	// The node type for a package in a solution deployment.
	DeploymentNodeType *string `json:"deploymentNodeType,omitempty" xmlrpc:"deploymentNodeType,omitempty"`

	// A count of the packages that are allowed in a multi-server solution. (Deprecated)
	DeploymentPackageCount *uint `json:"deploymentPackageCount,omitempty" xmlrpc:"deploymentPackageCount,omitempty"`

	// The packages that are allowed in a multi-server solution. (Deprecated)
	DeploymentPackages []Product_Package `json:"deploymentPackages,omitempty" xmlrpc:"deploymentPackages,omitempty"`

	// The solution deployment type.
	DeploymentType *string `json:"deploymentType,omitempty" xmlrpc:"deploymentType,omitempty"`

	// The package that represents a multi-server solution. (Deprecated)
	Deployments []Product_Package `json:"deployments,omitempty" xmlrpc:"deployments,omitempty"`

	// A generic description of the processor type and count. This includes HTML, so you may want to strip these tags if you plan to use it.
	Description *string `json:"description,omitempty" xmlrpc:"description,omitempty"`

	// This flag indicates the package does not allow custom disk partitions.
	DisallowCustomDiskPartitions *bool `json:"disallowCustomDiskPartitions,omitempty" xmlrpc:"disallowCustomDiskPartitions,omitempty"`

	// The Softlayer order step is optionally step-based. This returns the first SoftLayer_Product_Package_Order_Step in the step-based order process.
	FirstOrderStep *Product_Package_Order_Step `json:"firstOrderStep,omitempty" xmlrpc:"firstOrderStep,omitempty"`

	// This is only needed for step-based order verification. We use this for the order forms, but it is not required. This step is the first SoftLayer_Product_Package_Step for this package. Use this for for filtering which item categories are returned as a part of SoftLayer_Product_Package_Order_Configuration.
	FirstOrderStepId *int `json:"firstOrderStepId,omitempty" xmlrpc:"firstOrderStepId,omitempty"`

	// Whether the package is a specialized network gateway appliance package.
	GatewayApplianceFlag *bool `json:"gatewayApplianceFlag,omitempty" xmlrpc:"gatewayApplianceFlag,omitempty"`

	// This flag indicates that the package supports GPUs.
	GpuFlag *bool `json:"gpuFlag,omitempty" xmlrpc:"gpuFlag,omitempty"`

	// Determines whether the package contains prices that can be ordered hourly.
	HourlyBillingAvailableFlag *bool `json:"hourlyBillingAvailableFlag,omitempty" xmlrpc:"hourlyBillingAvailableFlag,omitempty"`

	// A package's internal identifier. Everything regarding a SoftLayer_Product_Package is tied back to this id.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// no documentation yet
	IsActive *int `json:"isActive,omitempty" xmlrpc:"isActive,omitempty"`

	// A count of the item-item conflicts associated with a package.
	ItemConflictCount *uint `json:"itemConflictCount,omitempty" xmlrpc:"itemConflictCount,omitempty"`

	// The item-item conflicts associated with a package.
	ItemConflicts []Product_Item_Resource_Conflict `json:"itemConflicts,omitempty" xmlrpc:"itemConflicts,omitempty"`

	// A count of a collection of valid items available for purchase in this package.
	ItemCount *uint `json:"itemCount,omitempty" xmlrpc:"itemCount,omitempty"`

	// A count of the item-location conflicts associated with a package.
	ItemLocationConflictCount *uint `json:"itemLocationConflictCount,omitempty" xmlrpc:"itemLocationConflictCount,omitempty"`

	// The item-location conflicts associated with a package.
	ItemLocationConflicts []Product_Item_Resource_Conflict `json:"itemLocationConflicts,omitempty" xmlrpc:"itemLocationConflicts,omitempty"`

	// A count of a collection of SoftLayer_Product_Item_Prices that are valid for this package.
	ItemPriceCount *uint `json:"itemPriceCount,omitempty" xmlrpc:"itemPriceCount,omitempty"`

	// A count of cross reference for item prices
	ItemPriceReferenceCount *uint `json:"itemPriceReferenceCount,omitempty" xmlrpc:"itemPriceReferenceCount,omitempty"`

	// cross reference for item prices
	ItemPriceReferences []Product_Package_Item_Prices `json:"itemPriceReferences,omitempty" xmlrpc:"itemPriceReferences,omitempty"`

	// A collection of SoftLayer_Product_Item_Prices that are valid for this package.
	ItemPrices []Product_Item_Price `json:"itemPrices,omitempty" xmlrpc:"itemPrices,omitempty"`

	// A collection of valid items available for purchase in this package.
	Items []Product_Item `json:"items,omitempty" xmlrpc:"items,omitempty"`

	// A unique key name for the package.
	KeyName *string `json:"keyName,omitempty" xmlrpc:"keyName,omitempty"`

	// A count of a collection of valid locations for this package. (Deprecated - Use [[SoftLayer_Product_Package/getRegions|getRegions]])
	LocationCount *uint `json:"locationCount,omitempty" xmlrpc:"locationCount,omitempty"`

	// A collection of valid locations for this package. (Deprecated - Use [[SoftLayer_Product_Package/getRegions|getRegions]])
	Locations []Location `json:"locations,omitempty" xmlrpc:"locations,omitempty"`

	// The lowest server [[SoftLayer_Product_Item_Price]] related to this package.
	LowestServerPrice *Product_Item_Price `json:"lowestServerPrice,omitempty" xmlrpc:"lowestServerPrice,omitempty"`

	// The maximum available network speed associated with the package.
	MaximumPortSpeed *uint `json:"maximumPortSpeed,omitempty" xmlrpc:"maximumPortSpeed,omitempty"`

	// The minimum available network speed associated with the package.
	MinimumPortSpeed *uint `json:"minimumPortSpeed,omitempty" xmlrpc:"minimumPortSpeed,omitempty"`

	// This flag indicates that this is a MongoDB engineered package. (Deprecated)
	MongoDbEngineeredFlag *bool `json:"mongoDbEngineeredFlag,omitempty" xmlrpc:"mongoDbEngineeredFlag,omitempty"`

	// The description of the package. For server packages, this is usually a detailed description of processor type and count.
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`

	// A count of the premium price modifiers associated with the [[SoftLayer_Product_Item_Price]] and [[SoftLayer_Location]] objects in a package.
	OrderPremiumCount *uint `json:"orderPremiumCount,omitempty" xmlrpc:"orderPremiumCount,omitempty"`

	// The premium price modifiers associated with the [[SoftLayer_Product_Item_Price]] and [[SoftLayer_Location]] objects in a package.
	OrderPremiums []Product_Item_Price_Premium `json:"orderPremiums,omitempty" xmlrpc:"orderPremiums,omitempty"`

	// This flag indicates the package is pre-configured. (Deprecated)
	PreconfiguredFlag *bool `json:"preconfiguredFlag,omitempty" xmlrpc:"preconfiguredFlag,omitempty"`

	// Whether the package requires the user to define a preset configuration.
	PresetConfigurationRequiredFlag *bool `json:"presetConfigurationRequiredFlag,omitempty" xmlrpc:"presetConfigurationRequiredFlag,omitempty"`

	// Whether the package prevents the user from specifying a Vlan.
	PreventVlanSelectionFlag *bool `json:"preventVlanSelectionFlag,omitempty" xmlrpc:"preventVlanSelectionFlag,omitempty"`

	// This flag indicates the package is for a private hosted cloud deployment. (Deprecated)
	PrivateHostedCloudPackageFlag *bool `json:"privateHostedCloudPackageFlag,omitempty" xmlrpc:"privateHostedCloudPackageFlag,omitempty"`

	// The server role of the private hosted cloud deployment. (Deprecated)
	PrivateHostedCloudPackageType *string `json:"privateHostedCloudPackageType,omitempty" xmlrpc:"privateHostedCloudPackageType,omitempty"`

	// Whether the package only has access to the private network.
	PrivateNetworkOnlyFlag *bool `json:"privateNetworkOnlyFlag,omitempty" xmlrpc:"privateNetworkOnlyFlag,omitempty"`

	// Whether the package is a specialized mass storage QuantaStor package.
	QuantaStorPackageFlag *bool `json:"quantaStorPackageFlag,omitempty" xmlrpc:"quantaStorPackageFlag,omitempty"`

	// This flag indicates the package does not allow different disks with RAID.
	RaidDiskRestrictionFlag *bool `json:"raidDiskRestrictionFlag,omitempty" xmlrpc:"raidDiskRestrictionFlag,omitempty"`

	// This flag determines if the package contains a redundant power supply product.
	RedundantPowerFlag *bool `json:"redundantPowerFlag,omitempty" xmlrpc:"redundantPowerFlag,omitempty"`

	// A count of the regional locations that a package is available in.
	RegionCount *uint `json:"regionCount,omitempty" xmlrpc:"regionCount,omitempty"`

	// The regional locations that a package is available in.
	Regions []Location_Region `json:"regions,omitempty" xmlrpc:"regions,omitempty"`

	// The resource group template that describes a multi-server solution. (Deprecated)
	ResourceGroupTemplate *Resource_Group_Template `json:"resourceGroupTemplate,omitempty" xmlrpc:"resourceGroupTemplate,omitempty"`

	// This currently contains no information but is here for future use.
	SubDescription *string `json:"subDescription,omitempty" xmlrpc:"subDescription,omitempty"`

	// The top level category code for this service offering.
	TopLevelItemCategoryCode *string `json:"topLevelItemCategoryCode,omitempty" xmlrpc:"topLevelItemCategoryCode,omitempty"`

	// The type of service offering. This property can be used to help filter packages.
	Type *Product_Package_Type `json:"type,omitempty" xmlrpc:"type,omitempty"`

	// The server unit size this package will match to.
	UnitSize *int `json:"unitSize,omitempty" xmlrpc:"unitSize,omitempty"`
}

// no documentation yet
type Product_Package_Attribute struct {
	Entity

	// no documentation yet
	AttributeType *Product_Package_Attribute_Type `json:"attributeType,omitempty" xmlrpc:"attributeType,omitempty"`

	// no documentation yet
	Package *Product_Package `json:"package,omitempty" xmlrpc:"package,omitempty"`

	// no documentation yet
	Value *string `json:"value,omitempty" xmlrpc:"value,omitempty"`
}

// no documentation yet
type Product_Package_Attribute_Type struct {
	Entity

	// no documentation yet
	Description *string `json:"description,omitempty" xmlrpc:"description,omitempty"`

	// no documentation yet
	KeyName *string `json:"keyName,omitempty" xmlrpc:"keyName,omitempty"`

	// no documentation yet
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`
}

// SoftLayer keeps near real-time track of the number of items available in it's product catalog inventory. The SoftLayer_Product_Package_Inventory data type models one of these inventory records. SoftLayer tracks inventory per product package and item per datacenter. This type is useful if you need to purchase specific servers in a specific location, and wish to check their availability before ordering.
//
// The data from this type is used primarily on the SoftLayer outlet website.
type Product_Package_Inventory struct {
	Entity

	// The number of units available for purchase in SoftLayer's inventory for a single item in a single datacenter.
	AvailableInventoryCount *int `json:"availableInventoryCount,omitempty" xmlrpc:"availableInventoryCount,omitempty"`

	// The product package item that is associated with an inventory record.
	Item *Product_Item `json:"item,omitempty" xmlrpc:"item,omitempty"`

	// The unique identifier of the product item that an inventory record is associated with.
	ItemId *int `json:"itemId,omitempty" xmlrpc:"itemId,omitempty"`

	// The datacenter that an inventory record is located in.
	Location *Location `json:"location,omitempty" xmlrpc:"location,omitempty"`

	// The unique identifier of the datacenter that an inventory record is located in.
	LocationId *int `json:"locationId,omitempty" xmlrpc:"locationId,omitempty"`

	// The date that an inventory record was last updated.
	ModifyDate *Time `json:"modifyDate,omitempty" xmlrpc:"modifyDate,omitempty"`

	// Whether an inventory record is marked as "overstock". Overstock records appear at the top portion of the SoftLayer outlet website.
	OverstockFlag *int `json:"overstockFlag,omitempty" xmlrpc:"overstockFlag,omitempty"`

	// The product package that is associated with an inventory record.
	Package *Product_Package `json:"package,omitempty" xmlrpc:"package,omitempty"`

	// The unique identifier of the product package that an inventory record is associated with.
	PackageId *int `json:"packageId,omitempty" xmlrpc:"packageId,omitempty"`
}

// This class is used to organize categories for a service offering. A service offering (usually) contains multiple categories (e.g., server, os, disk0, ram). This class allows us to organize the prices into related item category groups.
type Product_Package_Item_Category_Group struct {
	Entity

	// no documentation yet
	Category *Product_Item_Category `json:"category,omitempty" xmlrpc:"category,omitempty"`

	// The item category id associated with this group.
	ItemCategoryId *int `json:"itemCategoryId,omitempty" xmlrpc:"itemCategoryId,omitempty"`

	// no documentation yet
	Package *Product_Package `json:"package,omitempty" xmlrpc:"package,omitempty"`

	// The service offering id associated with this group.
	PackageId *int `json:"packageId,omitempty" xmlrpc:"packageId,omitempty"`

	// A count of
	PriceCount *uint `json:"priceCount,omitempty" xmlrpc:"priceCount,omitempty"`

	// no documentation yet
	Prices []Product_Item_Price `json:"prices,omitempty" xmlrpc:"prices,omitempty"`

	// The sort value for this group.
	Sort *int `json:"sort,omitempty" xmlrpc:"sort,omitempty"`

	// An optional title associated with this group. E.g., for operating systems, this will be the manufacturer.
	Title *string `json:"title,omitempty" xmlrpc:"title,omitempty"`
}

// The SoftLayer_Product_Package_Item_Prices contains price to package cross references Relates a category, price and item to a bundle.  Match bundle ids to see all items and prices in a particular bundle.
type Product_Package_Item_Prices struct {
	Entity

	// The unique identifier for SoftLayer_Product_Package_Item_Price. This is only needed as a reference. The important data is the itemPriceId property.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// The item price to which this object belongs. The item price has details regarding cost for the item it belongs to.
	ItemPrice *Product_Item_Price `json:"itemPrice,omitempty" xmlrpc:"itemPrice,omitempty"`

	// The SoftLayer_Product_Item_Price id. This value is to be used when placing orders. To get more information about this item price, go from the item price to the item description
	ItemPriceId *int `json:"itemPriceId,omitempty" xmlrpc:"itemPriceId,omitempty"`

	// The package to which this object belongs.
	Package *Product_Package `json:"package,omitempty" xmlrpc:"package,omitempty"`

	// The Package ID to which this price reference belongs
	PackageId *int `json:"packageId,omitempty" xmlrpc:"packageId,omitempty"`
}

// This data type is a cross-reference between the SoftLayer_Product_Package and the SoftLayer_Product_Item(s) that belong in the SoftLayer_Product_Package.
type Product_Package_Items struct {
	Entity

	// The unique identifier for this object. It is not used anywhere but in this object.
	Id *string `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// The item to which this object belongs.
	Item *Product_Item `json:"item,omitempty" xmlrpc:"item,omitempty"`

	// The SoftLayer_Product_Item id to which this instance of the object belongs.
	ItemId *int `json:"itemId,omitempty" xmlrpc:"itemId,omitempty"`

	// The package to which this object belongs.
	Package *Product_Package `json:"package,omitempty" xmlrpc:"package,omitempty"`

	// The SoftLayer_Product_Package id to which this instance of the object belongs.
	PackageId *int `json:"packageId,omitempty" xmlrpc:"packageId,omitempty"`
}

// Most packages are available in many locations. This object describes that availability for each package.
type Product_Package_Locations struct {
	Entity

	// This describes the availability of the package tied to this location.
	DeliveryTimeInformation *string `json:"deliveryTimeInformation,omitempty" xmlrpc:"deliveryTimeInformation,omitempty"`

	// A simple flag which describes whether or not this location is available for this package.
	IsAvailable *int `json:"isAvailable,omitempty" xmlrpc:"isAvailable,omitempty"`

	// The location to which this object belongs.
	Location *Location `json:"location,omitempty" xmlrpc:"location,omitempty"`

	// The location id tied to this object.
	LocationId *int `json:"locationId,omitempty" xmlrpc:"locationId,omitempty"`

	// The package to which this object belongs.
	Package *Product_Package `json:"package,omitempty" xmlrpc:"package,omitempty"`

	// The SoftLayer_Product_Package ID tied to this object.
	PackageId *int `json:"packageId,omitempty" xmlrpc:"packageId,omitempty"`
}

// This datatype describes the item categories that are required for each package to be ordered. For instance, for package 2, there will be many required categories. When submitting an order for a server, there must be at most 1 price for each category whose "isRequired" is set. Examples of required categories: - server - ram - bandwidth - disk0
//
// There are others, but these are the main ones. For each required category, a SoftLayer_Product_Item_Price must be chosen that is valid for the package.
//
//
type Product_Package_Order_Configuration struct {
	Entity

	// The error message displayed if the submitted order does not contain this item category, if it is required.
	ErrorMessage *string `json:"errorMessage,omitempty" xmlrpc:"errorMessage,omitempty"`

	// The unique identifier for this object.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// This is a flag which tells SoftLayer_Product_Order::verifyOrder() whether or not this category is required. If this is set, then the order submitted must contain a SoftLayer_Product_Item_Price with this category as part of the order.
	IsRequired *int `json:"isRequired,omitempty" xmlrpc:"isRequired,omitempty"`

	// The item category for this configuration instance.
	ItemCategory *Product_Item_Category `json:"itemCategory,omitempty" xmlrpc:"itemCategory,omitempty"`

	// The SoftLayer_Product_Item_Category.
	ItemCategoryId *int `json:"itemCategoryId,omitempty" xmlrpc:"itemCategoryId,omitempty"`

	// The order step ID for this particular option in the package.
	OrderStepId *int `json:"orderStepId,omitempty" xmlrpc:"orderStepId,omitempty"`

	// The package to which this instance belongs.
	Package *Product_Package `json:"package,omitempty" xmlrpc:"package,omitempty"`

	// The PackageId tied to this instance.
	PackageId *int `json:"packageId,omitempty" xmlrpc:"packageId,omitempty"`

	// This is an integer used to show the order in which each item Category should be displayed. This is merely the suggested order.
	Sort *int `json:"sort,omitempty" xmlrpc:"sort,omitempty"`

	// The step to which this instance belongs.
	Step *Product_Package_Order_Step `json:"step,omitempty" xmlrpc:"step,omitempty"`
}

// Each package has at least 1 step to the ordering process. For server orders, there are many. Each step has certain item categories which are displayed. This type describes the steps for each package.
type Product_Package_Order_Step struct {
	Entity

	// The unique identifier for this object. It is not used anywhere but in this object.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// A count of the next steps in the ordering process for the package tied to this object, including this step.
	InclusivePreviousStepCount *uint `json:"inclusivePreviousStepCount,omitempty" xmlrpc:"inclusivePreviousStepCount,omitempty"`

	// The next steps in the ordering process for the package tied to this object, including this step.
	InclusivePreviousSteps []Product_Package_Order_Step_Next `json:"inclusivePreviousSteps,omitempty" xmlrpc:"inclusivePreviousSteps,omitempty"`

	// A count of the next steps in the ordering process for the package tied to this object.
	NextStepCount *uint `json:"nextStepCount,omitempty" xmlrpc:"nextStepCount,omitempty"`

	// The next steps in the ordering process for the package tied to this object.
	NextSteps []Product_Package_Order_Step_Next `json:"nextSteps,omitempty" xmlrpc:"nextSteps,omitempty"`

	// A count of the item to which this object belongs.
	PreviousStepCount *uint `json:"previousStepCount,omitempty" xmlrpc:"previousStepCount,omitempty"`

	// The item to which this object belongs.
	PreviousSteps []Product_Package_Order_Step_Next `json:"previousSteps,omitempty" xmlrpc:"previousSteps,omitempty"`

	// The number of the step in the order process for this package. These are sequential and only needed for step-based ordering.
	Step *string `json:"step,omitempty" xmlrpc:"step,omitempty"`
}

// This datatype simply describes which steps are next in line for ordering.
type Product_Package_Order_Step_Next struct {
	Entity

	// The unique identifier for this object.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// The unique identifier for SoftLayer_Product_Package_Order_Step for the next step in the process.
	NextOrderStepId *int `json:"nextOrderStepId,omitempty" xmlrpc:"nextOrderStepId,omitempty"`

	// The unique identifier for SoftLayer_Product_Package_Order_Step for the current step.
	OrderStepId *int `json:"orderStepId,omitempty" xmlrpc:"orderStepId,omitempty"`

	// The SoftLayer_Product_Package_Order_Step to which this object belongs.
	Step *Product_Package_Order_Step `json:"step,omitempty" xmlrpc:"step,omitempty"`
}

// Package presets are used to simplify ordering by eliminating the need for price ids when submitting orders.
//
// Orders submitted without prices or a preset id defined will use the “DEFAULT” preset for the package id. The default package presets include the base options required for a package configuration.
//
// Orders submitted with a preset id defined will use the prices included in the package preset. Prices submitted on an order with a preset id will replace the prices included in the package preset for that prices category. If the package preset has a fixed configuration flag <em>(fixedConfigurationFlag)</em> set then the prices included in the preset configuration cannot be replaced by prices submitted on the order. The only exception to the fixed configuration flag would be if a price submitted on the order is an account-restricted price for the same product item.
type Product_Package_Preset struct {
	Entity

	// no documentation yet
	AvailableStorageUnits *uint `json:"availableStorageUnits,omitempty" xmlrpc:"availableStorageUnits,omitempty"`

	// The item categories that are included in this package preset configuration.
	Categories []Product_Item_Category `json:"categories,omitempty" xmlrpc:"categories,omitempty"`

	// A count of the item categories that are included in this package preset configuration.
	CategoryCount *uint `json:"categoryCount,omitempty" xmlrpc:"categoryCount,omitempty"`

	// The preset configuration (category and price).
	Configuration []Product_Package_Preset_Configuration `json:"configuration,omitempty" xmlrpc:"configuration,omitempty"`

	// A count of the preset configuration (category and price).
	ConfigurationCount *uint `json:"configurationCount,omitempty" xmlrpc:"configurationCount,omitempty"`

	// A description of the package preset.
	Description *string `json:"description,omitempty" xmlrpc:"description,omitempty"`

	// A package preset with this flag set will not allow the price's defined in the preset configuration to be overriden during order placement.
	FixedConfigurationFlag *bool `json:"fixedConfigurationFlag,omitempty" xmlrpc:"fixedConfigurationFlag,omitempty"`

	// A preset's internal identifier. Everything regarding a SoftLayer_Product_Package_Preset is tied back to this id.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// The status of the package preset.
	IsActive *string `json:"isActive,omitempty" xmlrpc:"isActive,omitempty"`

	// The key name of the package preset. For the base configuration of a package the preset key name is "DEFAULT".
	KeyName *string `json:"keyName,omitempty" xmlrpc:"keyName,omitempty"`

	// The lowest server prices related to this package preset.
	LowestPresetServerPrice *Product_Item_Price `json:"lowestPresetServerPrice,omitempty" xmlrpc:"lowestPresetServerPrice,omitempty"`

	// The name of the package preset.
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`

	// The package this preset belongs to.
	Package *Product_Package `json:"package,omitempty" xmlrpc:"package,omitempty"`

	// The item categories associated with a package preset, including information detailing which item categories are required as part of a SoftLayer product order.
	PackageConfiguration []Product_Package_Order_Configuration `json:"packageConfiguration,omitempty" xmlrpc:"packageConfiguration,omitempty"`

	// A count of the item categories associated with a package preset, including information detailing which item categories are required as part of a SoftLayer product order.
	PackageConfigurationCount *uint `json:"packageConfigurationCount,omitempty" xmlrpc:"packageConfigurationCount,omitempty"`

	// The package id for the package this preset belongs to.
	PackageId *int `json:"packageId,omitempty" xmlrpc:"packageId,omitempty"`

	// A count of the item prices that are included in this package preset configuration.
	PriceCount *uint `json:"priceCount,omitempty" xmlrpc:"priceCount,omitempty"`

	// The item prices that are included in this package preset configuration.
	Prices []Product_Item_Price `json:"prices,omitempty" xmlrpc:"prices,omitempty"`

	// A count of describes how all disks in this preset will be configured.
	StorageGroupTemplateArrayCount *uint `json:"storageGroupTemplateArrayCount,omitempty" xmlrpc:"storageGroupTemplateArrayCount,omitempty"`

	// Describes how all disks in this preset will be configured.
	StorageGroupTemplateArrays []Configuration_Storage_Group_Template_Group `json:"storageGroupTemplateArrays,omitempty" xmlrpc:"storageGroupTemplateArrays,omitempty"`

	// The starting hourly price for this configuration. Additional options not defined in the preset may increase the cost.
	TotalMinimumHourlyFee *Float64 `json:"totalMinimumHourlyFee,omitempty" xmlrpc:"totalMinimumHourlyFee,omitempty"`

	// The starting monthly price for this configuration. Additional options not defined in the preset may increase the cost.
	TotalMinimumRecurringFee *Float64 `json:"totalMinimumRecurringFee,omitempty" xmlrpc:"totalMinimumRecurringFee,omitempty"`
}

// Package preset attributes contain supplementary information for a package preset.
type Product_Package_Preset_Attribute struct {
	Entity

	// no documentation yet
	AttributeType *Product_Package_Preset_Attribute_Type `json:"attributeType,omitempty" xmlrpc:"attributeType,omitempty"`

	// The internal identifier of the type of attribute that a pacakge preset attribute belongs to.
	AttributeTypeId *int `json:"attributeTypeId,omitempty" xmlrpc:"attributeTypeId,omitempty"`

	// A package preset attribute's internal identifier.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// no documentation yet
	Preset *Product_Package_Preset `json:"preset,omitempty" xmlrpc:"preset,omitempty"`

	// The internal identifier of the package preset an attribute belongs to.
	PresetId *int `json:"presetId,omitempty" xmlrpc:"presetId,omitempty"`

	// A package preset's attribute value.
	Value *string `json:"value,omitempty" xmlrpc:"value,omitempty"`
}

// SoftLayer_Product_Package_Preset_Attribute_Type models the type of attribute that can be assigned to a package preset.
type Product_Package_Preset_Attribute_Type struct {
	Entity

	// A brief description of a package preset attribute type.
	Description *string `json:"description,omitempty" xmlrpc:"description,omitempty"`

	// A package preset attribute type's internal identifier.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// A package preset attribute type's key name. This is typically a shorter version of an attribute type's name.
	KeyName *string `json:"keyName,omitempty" xmlrpc:"keyName,omitempty"`

	// A package preset attribute type's name.
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`
}

// no documentation yet
type Product_Package_Preset_Configuration struct {
	Entity

	// no documentation yet
	Category *Product_Item_Category `json:"category,omitempty" xmlrpc:"category,omitempty"`

	// no documentation yet
	PackagePreset *Product_Package_Preset `json:"packagePreset,omitempty" xmlrpc:"packagePreset,omitempty"`

	// no documentation yet
	Price *Product_Item_Price `json:"price,omitempty" xmlrpc:"price,omitempty"`
}

// The SoftLayer_Product_Package_Server data type contains summarized information for bare metal servers regarding pricing, processor stats, and feature sets.
type Product_Package_Server struct {
	Entity

	// no documentation yet
	Catalog *Product_Catalog `json:"catalog,omitempty" xmlrpc:"catalog,omitempty"`

	// The unique identifier of a [[SoftLayer_Product_Catalog]].
	CatalogId *int `json:"catalogId,omitempty" xmlrpc:"catalogId,omitempty"`

	// Comma-separated list of datacenter names this server is available in
	Datacenters *string `json:"datacenters,omitempty" xmlrpc:"datacenters,omitempty"`

	// The minimum amount of RAM the server is configured with.
	DefaultRamCapacity *Float64 `json:"defaultRamCapacity,omitempty" xmlrpc:"defaultRamCapacity,omitempty"`

	// Flag to indicate if the server configuration supports dual path network routing.
	DualPathNetworkFlag *bool `json:"dualPathNetworkFlag,omitempty" xmlrpc:"dualPathNetworkFlag,omitempty"`

	// Indicates whether or not the server contains a GPU.
	GpuFlag *bool `json:"gpuFlag,omitempty" xmlrpc:"gpuFlag,omitempty"`

	// Flag to determine if a server is available for hourly billing.
	HourlyBillingFlag *bool `json:"hourlyBillingFlag,omitempty" xmlrpc:"hourlyBillingFlag,omitempty"`

	// The unique identifier of a [[SoftLayer_Product_Package_Server]].
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// no documentation yet
	Item *Product_Item `json:"item,omitempty" xmlrpc:"item,omitempty"`

	// The unique identifier of a [[SoftLayer_Product_Item]].
	ItemId *int `json:"itemId,omitempty" xmlrpc:"itemId,omitempty"`

	// no documentation yet
	ItemPrice *Product_Item_Price `json:"itemPrice,omitempty" xmlrpc:"itemPrice,omitempty"`

	// The unique identifier of a [[SoftLayer_Product_Item_Price]].
	ItemPriceId *int `json:"itemPriceId,omitempty" xmlrpc:"itemPriceId,omitempty"`

	// The maximum number of hard drives the server can support.
	MaximumDriveCount *int `json:"maximumDriveCount,omitempty" xmlrpc:"maximumDriveCount,omitempty"`

	// The maximum available network speed for the server.
	MaximumPortSpeed *Float64 `json:"maximumPortSpeed,omitempty" xmlrpc:"maximumPortSpeed,omitempty"`

	// The maximum amount of RAM the server can support.
	MaximumRamCapacity *Float64 `json:"maximumRamCapacity,omitempty" xmlrpc:"maximumRamCapacity,omitempty"`

	// The minimum available network speed for the server.
	MinimumPortSpeed *Float64 `json:"minimumPortSpeed,omitempty" xmlrpc:"minimumPortSpeed,omitempty"`

	// no documentation yet
	NetworkGatewayApplianceRoleFlag *bool `json:"networkGatewayApplianceRoleFlag,omitempty" xmlrpc:"networkGatewayApplianceRoleFlag,omitempty"`

	// Indicates whether or not the server is being sold as part of an outlet package.
	OutletFlag *bool `json:"outletFlag,omitempty" xmlrpc:"outletFlag,omitempty"`

	// no documentation yet
	Package *Product_Package `json:"package,omitempty" xmlrpc:"package,omitempty"`

	// The unique identifier of a [[SoftLayer_Product_Package]].
	PackageId *int `json:"packageId,omitempty" xmlrpc:"packageId,omitempty"`

	// The type of service offering/package.
	PackageType *string `json:"packageType,omitempty" xmlrpc:"packageType,omitempty"`

	// Flag to indicate if the server is an IBM Power server.
	PowerServerFlag *bool `json:"powerServerFlag,omitempty" xmlrpc:"powerServerFlag,omitempty"`

	// no documentation yet
	Preset *Product_Package_Preset `json:"preset,omitempty" xmlrpc:"preset,omitempty"`

	// The unique identifier of a [[SoftLayer_Product_Package_Preset]].
	PresetId *int `json:"presetId,omitempty" xmlrpc:"presetId,omitempty"`

	// Indicates whether or not the server can only be configured with a private network.
	PrivateNetworkOnlyFlag *bool `json:"privateNetworkOnlyFlag,omitempty" xmlrpc:"privateNetworkOnlyFlag,omitempty"`

	// The processor's bus speed.
	ProcessorBusSpeed *string `json:"processorBusSpeed,omitempty" xmlrpc:"processorBusSpeed,omitempty"`

	// The amount of cache the processor has.
	ProcessorCache *string `json:"processorCache,omitempty" xmlrpc:"processorCache,omitempty"`

	// The number of cores in each processor.
	ProcessorCores *int `json:"processorCores,omitempty" xmlrpc:"processorCores,omitempty"`

	// The number of processors the server has.
	ProcessorCount *int `json:"processorCount,omitempty" xmlrpc:"processorCount,omitempty"`

	// The manufacturer of the server's processor.
	ProcessorManufacturer *string `json:"processorManufacturer,omitempty" xmlrpc:"processorManufacturer,omitempty"`

	// The model of the server's processor.
	ProcessorModel *string `json:"processorModel,omitempty" xmlrpc:"processorModel,omitempty"`

	// The name of the server's processor.
	ProcessorName *string `json:"processorName,omitempty" xmlrpc:"processorName,omitempty"`

	// The processor speed.
	ProcessorSpeed *string `json:"processorSpeed,omitempty" xmlrpc:"processorSpeed,omitempty"`

	// The name of the server product.
	ProductName *string `json:"productName,omitempty" xmlrpc:"productName,omitempty"`

	// Indicates whether or not the server has the capability to support a redundant power supply.
	RedundantPowerFlag *bool `json:"redundantPowerFlag,omitempty" xmlrpc:"redundantPowerFlag,omitempty"`

	// Flag to indicate if the server is SAP certified.
	SapCertifiedServerFlag *bool `json:"sapCertifiedServerFlag,omitempty" xmlrpc:"sapCertifiedServerFlag,omitempty"`

	// The hourly starting price for the server. This includes a sum of all the minimum required items, including RAM and hard drives. Not all servers are available hourly.
	StartingHourlyPrice *Float64 `json:"startingHourlyPrice,omitempty" xmlrpc:"startingHourlyPrice,omitempty"`

	// The monthly starting price for the server. This includes a sum of all the minimum required items, including RAM and hard drives.
	StartingMonthlyPrice *Float64 `json:"startingMonthlyPrice,omitempty" xmlrpc:"startingMonthlyPrice,omitempty"`

	// The total number of processor cores available for the server.
	TotalCoreCount *int `json:"totalCoreCount,omitempty" xmlrpc:"totalCoreCount,omitempty"`

	// Flag to indicate if the server configuration supports TXT/TPM.
	TxtTpmFlag *bool `json:"txtTpmFlag,omitempty" xmlrpc:"txtTpmFlag,omitempty"`

	// The size of the server.
	UnitSize *int `json:"unitSize,omitempty" xmlrpc:"unitSize,omitempty"`
}

// The [[SoftLayer_Product_Package_Server_Option]] data type contains various data points associated with package servers that can be used in selection criteria.
type Product_Package_Server_Option struct {
	Entity

	// The unique identifier of a Catalog.
	CatalogId *int `json:"catalogId,omitempty" xmlrpc:"catalogId,omitempty"`

	// A description of the option.
	Description *string `json:"description,omitempty" xmlrpc:"description,omitempty"`

	// The unique identifier of a Package Server Option.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// The type of option.
	Type *string `json:"type,omitempty" xmlrpc:"type,omitempty"`

	// The value of the the option.
	Value *string `json:"value,omitempty" xmlrpc:"value,omitempty"`
}

// The [[SoftLayer_Product_Package_Type]] object indicates the type for a service offering (package). The type can be used to filter packages. For example, if you are looking for the package representing virtual servers, you can filter on the type's key name of '''VIRTUAL_SERVER_INSTANCE'''. For bare metal servers by core or CPU, filter on '''BARE_METAL_CORE''' or '''BARE_METAL_CPU''', respectively.
type Product_Package_Type struct {
	Entity

	// The package type's unique identifier.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// The unique key name of the package type. Use this value when filtering.
	KeyName *string `json:"keyName,omitempty" xmlrpc:"keyName,omitempty"`

	// The name of the package type.
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`

	// A count of all the packages associated with the given package type.
	PackageCount *uint `json:"packageCount,omitempty" xmlrpc:"packageCount,omitempty"`

	// All the packages associated with the given package type.
	Packages []Product_Package `json:"packages,omitempty" xmlrpc:"packages,omitempty"`
}

// The SoftLayer_Product_Upgrade_Request data type contains general information relating to a hardware, virtual server, or service upgrade. It also relates a [[SoftLayer_Billing_Order]] to a [[SoftLayer_Ticket]].
type Product_Upgrade_Request struct {
	Entity

	// The account that an order belongs to
	Account *Account `json:"account,omitempty" xmlrpc:"account,omitempty"`

	// The unique internal id of a SoftLayer account
	AccountId *int `json:"accountId,omitempty" xmlrpc:"accountId,omitempty"`

	// Indicates that the upgrade request has completed or has been cancelled.
	CompletedFlag *bool `json:"completedFlag,omitempty" xmlrpc:"completedFlag,omitempty"`

	// The date an upgrade request was created.
	CreateDate *Time `json:"createDate,omitempty" xmlrpc:"createDate,omitempty"`

	// The unique internal id of the last modified user
	EmployeeId *int `json:"employeeId,omitempty" xmlrpc:"employeeId,omitempty"`

	// The unique internal id of the virtual server that an upgrade will be done
	GuestId *int `json:"guestId,omitempty" xmlrpc:"guestId,omitempty"`

	// The unique internal id of the hardware that an upgrade will be done
	HardwareId *int `json:"hardwareId,omitempty" xmlrpc:"hardwareId,omitempty"`

	// An upgrade request's internal identifier.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// This is the invoice associated with the upgrade request. For hourly servers or services, an invoice will not be available.
	Invoice *Billing_Invoice `json:"invoice,omitempty" xmlrpc:"invoice,omitempty"`

	// The time that system admin starts working on the order item.  This is used for upgrade orders.
	MaintenanceStartTimeUtc *Time `json:"maintenanceStartTimeUtc,omitempty" xmlrpc:"maintenanceStartTimeUtc,omitempty"`

	// The date an upgrade request was last modified.
	ModifyDate *Time `json:"modifyDate,omitempty" xmlrpc:"modifyDate,omitempty"`

	// An order record associated to the upgrade request
	Order *Billing_Order `json:"order,omitempty" xmlrpc:"order,omitempty"`

	// The unique internal id of the order that an upgrade request is related to
	OrderId *int `json:"orderId,omitempty" xmlrpc:"orderId,omitempty"`

	// The total amount of fees
	OrderTotal *Float64 `json:"orderTotal,omitempty" xmlrpc:"orderTotal,omitempty"`

	// The prorated total amount of recurring fees
	ProratedTotal *Float64 `json:"proratedTotal,omitempty" xmlrpc:"proratedTotal,omitempty"`

	// A server object associated with the upgrade request if any.
	Server *Hardware `json:"server,omitempty" xmlrpc:"server,omitempty"`

	// The current status of the upgrade request.
	Status *Product_Upgrade_Request_Status `json:"status,omitempty" xmlrpc:"status,omitempty"`

	// The unique internal id of an upgrade status
	StatusId *int `json:"statusId,omitempty" xmlrpc:"statusId,omitempty"`

	// The ticket that is used to coordinate the upgrade process.
	Ticket *Ticket `json:"ticket,omitempty" xmlrpc:"ticket,omitempty"`

	// The unique internal id of the ticket related to an upgrade request
	TicketId *int `json:"ticketId,omitempty" xmlrpc:"ticketId,omitempty"`

	// The user that placed the order.
	User *User_Customer `json:"user,omitempty" xmlrpc:"user,omitempty"`

	// The unique internal id of the customer who place the order
	UserId *int `json:"userId,omitempty" xmlrpc:"userId,omitempty"`

	// A virtual server object associated with the upgrade request if any.
	VirtualGuest *Virtual_Guest `json:"virtualGuest,omitempty" xmlrpc:"virtualGuest,omitempty"`
}

// The SoftLayer_Product_Upgrade_Request_Status data type contains detailed information relating to an hardware or software upgrade request.
type Product_Upgrade_Request_Status struct {
	Entity

	// The detailed description of an upgrade request status.
	Description *string `json:"description,omitempty" xmlrpc:"description,omitempty"`

	// An internal identifier of an upgrade request status.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// The name of an upgrade request status.
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`

	// The status code of an upgrade request status.
	StatusCode *string `json:"statusCode,omitempty" xmlrpc:"statusCode,omitempty"`
}
