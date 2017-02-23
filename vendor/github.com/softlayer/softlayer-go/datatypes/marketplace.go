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

// no documentation yet
type Marketplace_EmailDistribution struct {
	Entity

	// no documentation yet
	Email *string `json:"email,omitempty" xmlrpc:"email,omitempty"`

	// no documentation yet
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`
}

// no documentation yet
type Marketplace_Partner struct {
	Entity

	// no documentation yet
	AccountId *int `json:"accountId,omitempty" xmlrpc:"accountId,omitempty"`

	// no documentation yet
	AttachedFiles []Marketplace_Partner_Attachment `json:"attachedFiles,omitempty" xmlrpc:"attachedFiles,omitempty"`

	// A count of
	AttachmentCount *uint `json:"attachmentCount,omitempty" xmlrpc:"attachmentCount,omitempty"`

	// no documentation yet
	Attachments []Marketplace_Partner_Attachment `json:"attachments,omitempty" xmlrpc:"attachments,omitempty"`

	// no documentation yet
	CompanyDescription *string `json:"companyDescription,omitempty" xmlrpc:"companyDescription,omitempty"`

	// no documentation yet
	CompanyName *string `json:"companyName,omitempty" xmlrpc:"companyName,omitempty"`

	// no documentation yet
	HeadlineDescription *string `json:"headlineDescription,omitempty" xmlrpc:"headlineDescription,omitempty"`

	// no documentation yet
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// no documentation yet
	LinkFreeTrial *string `json:"linkFreeTrial,omitempty" xmlrpc:"linkFreeTrial,omitempty"`

	// no documentation yet
	LinkOrderPage *string `json:"linkOrderPage,omitempty" xmlrpc:"linkOrderPage,omitempty"`

	// no documentation yet
	LinkWebsite *string `json:"linkWebsite,omitempty" xmlrpc:"linkWebsite,omitempty"`

	// no documentation yet
	LogoMedium *Marketplace_Partner_Attachment `json:"logoMedium,omitempty" xmlrpc:"logoMedium,omitempty"`

	// no documentation yet
	LogoMediumTemp *Marketplace_Partner_Attachment `json:"logoMediumTemp,omitempty" xmlrpc:"logoMediumTemp,omitempty"`

	// no documentation yet
	LogoSmall *Marketplace_Partner_Attachment `json:"logoSmall,omitempty" xmlrpc:"logoSmall,omitempty"`

	// no documentation yet
	LogoSmallTemp *Marketplace_Partner_Attachment `json:"logoSmallTemp,omitempty" xmlrpc:"logoSmallTemp,omitempty"`

	// no documentation yet
	MetaDescription *string `json:"metaDescription,omitempty" xmlrpc:"metaDescription,omitempty"`

	// no documentation yet
	MetaKeywords *string `json:"metaKeywords,omitempty" xmlrpc:"metaKeywords,omitempty"`

	// no documentation yet
	ProductBenefits *string `json:"productBenefits,omitempty" xmlrpc:"productBenefits,omitempty"`

	// no documentation yet
	ProductCategoryId *int `json:"productCategoryId,omitempty" xmlrpc:"productCategoryId,omitempty"`

	// no documentation yet
	ProductDescriptionLong *string `json:"productDescriptionLong,omitempty" xmlrpc:"productDescriptionLong,omitempty"`

	// no documentation yet
	ProductDescriptionShort *string `json:"productDescriptionShort,omitempty" xmlrpc:"productDescriptionShort,omitempty"`

	// no documentation yet
	ProductFeatures *string `json:"productFeatures,omitempty" xmlrpc:"productFeatures,omitempty"`

	// no documentation yet
	ProductName *string `json:"productName,omitempty" xmlrpc:"productName,omitempty"`

	// no documentation yet
	ProductTitle *string `json:"productTitle,omitempty" xmlrpc:"productTitle,omitempty"`

	// no documentation yet
	UrlIdentifier *string `json:"urlIdentifier,omitempty" xmlrpc:"urlIdentifier,omitempty"`
}

// no documentation yet
type Marketplace_Partner_Attachment struct {
	Entity

	// no documentation yet
	AttachmentType *Marketplace_Partner_Attachment_Type `json:"attachmentType,omitempty" xmlrpc:"attachmentType,omitempty"`

	// no documentation yet
	AttachmentTypeId *int `json:"attachmentTypeId,omitempty" xmlrpc:"attachmentTypeId,omitempty"`

	// no documentation yet
	BaseName *string `json:"baseName,omitempty" xmlrpc:"baseName,omitempty"`

	// no documentation yet
	DisplayName *string `json:"displayName,omitempty" xmlrpc:"displayName,omitempty"`

	// no documentation yet
	FileName *string `json:"fileName,omitempty" xmlrpc:"fileName,omitempty"`

	// no documentation yet
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// no documentation yet
	MarketplacePartnerId *int `json:"marketplacePartnerId,omitempty" xmlrpc:"marketplacePartnerId,omitempty"`

	// no documentation yet
	SaveAsName *string `json:"saveAsName,omitempty" xmlrpc:"saveAsName,omitempty"`
}

// no documentation yet
type Marketplace_Partner_Attachment_Type struct {
	Entity

	// no documentation yet
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// no documentation yet
	KeyName *string `json:"keyName,omitempty" xmlrpc:"keyName,omitempty"`

	// no documentation yet
	Type *string `json:"type,omitempty" xmlrpc:"type,omitempty"`
}

// no documentation yet
type Marketplace_Partner_File struct {
	Entity

	// no documentation yet
	Attributes *Marketplace_Partner_File_Attributes `json:"attributes,omitempty" xmlrpc:"attributes,omitempty"`

	// no documentation yet
	Contents *[]byte `json:"contents,omitempty" xmlrpc:"contents,omitempty"`
}

// no documentation yet
type Marketplace_Partner_File_Attributes struct {
	Entity

	// no documentation yet
	Bits *int `json:"bits,omitempty" xmlrpc:"bits,omitempty"`

	// no documentation yet
	Channels *int `json:"channels,omitempty" xmlrpc:"channels,omitempty"`

	// no documentation yet
	Height *int `json:"height,omitempty" xmlrpc:"height,omitempty"`

	// no documentation yet
	HtmlAttributes *string `json:"htmlAttributes,omitempty" xmlrpc:"htmlAttributes,omitempty"`

	// no documentation yet
	ImageType *int `json:"imageType,omitempty" xmlrpc:"imageType,omitempty"`

	// no documentation yet
	IsImage *bool `json:"isImage,omitempty" xmlrpc:"isImage,omitempty"`

	// no documentation yet
	MimeType *string `json:"mimeType,omitempty" xmlrpc:"mimeType,omitempty"`

	// no documentation yet
	Width *int `json:"width,omitempty" xmlrpc:"width,omitempty"`
}
