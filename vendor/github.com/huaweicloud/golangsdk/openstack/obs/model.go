// Copyright 2019 Huawei Technologies Co.,Ltd.
// Licensed under the Apache License, Version 2.0 (the "License"); you may not use
// this file except in compliance with the License.  You may obtain a copy of the
// License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software distributed
// under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR
// CONDITIONS OF ANY KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations under the License.

package obs

import (
	"encoding/xml"
	"io"
	"net/http"
	"time"
)

type BaseModel struct {
	StatusCode      int                 `xml:"-"`
	RequestId       string              `xml:"RequestId"`
	ResponseHeaders map[string][]string `xml:"-"`
}

type Bucket struct {
	XMLName      xml.Name  `xml:"Bucket"`
	Name         string    `xml:"Name"`
	CreationDate time.Time `xml:"CreationDate"`
	Location     string    `xml:"Location"`
}

type Owner struct {
	XMLName     xml.Name `xml:"Owner"`
	ID          string   `xml:"ID"`
	DisplayName string   `xml:"DisplayName,omitempty"`
}

type Initiator struct {
	XMLName     xml.Name `xml:"Initiator"`
	ID          string   `xml:"ID"`
	DisplayName string   `xml:"DisplayName,omitempty"`
}

type ListBucketsInput struct {
	QueryLocation bool
}

type ListBucketsOutput struct {
	BaseModel
	XMLName xml.Name `xml:"ListAllMyBucketsResult"`
	Owner   Owner    `xml:"Owner"`
	Buckets []Bucket `xml:"Buckets>Bucket"`
}

type bucketLocationObs struct {
	XMLName  xml.Name `xml:"Location"`
	Location string   `xml:",chardata"`
}

type BucketLocation struct {
	XMLName  xml.Name `xml:"CreateBucketConfiguration"`
	Location string   `xml:"LocationConstraint,omitempty"`
}

type CreateBucketInput struct {
	BucketLocation
	Bucket                      string           `xml:"-"`
	ACL                         AclType          `xml:"-"`
	StorageClass                StorageClassType `xml:"-"`
	GrantReadId                 string           `xml:"-"`
	GrantWriteId                string           `xml:"-"`
	GrantReadAcpId              string           `xml:"-"`
	GrantWriteAcpId             string           `xml:"-"`
	GrantFullControlId          string           `xml:"-"`
	GrantReadDeliveredId        string           `xml:"-"`
	GrantFullControlDeliveredId string           `xml:"-"`
	Epid                        string           `xml:"-"`
}

type BucketStoragePolicy struct {
	XMLName      xml.Name         `xml:"StoragePolicy"`
	StorageClass StorageClassType `xml:"DefaultStorageClass"`
}

type SetBucketStoragePolicyInput struct {
	Bucket string `xml:"-"`
	BucketStoragePolicy
}

type getBucketStoragePolicyOutputS3 struct {
	BaseModel
	BucketStoragePolicy
}

type GetBucketStoragePolicyOutput struct {
	BaseModel
	StorageClass string
}

type bucketStoragePolicyObs struct {
	XMLName      xml.Name `xml:"StorageClass"`
	StorageClass string   `xml:",chardata"`
}
type getBucketStoragePolicyOutputObs struct {
	BaseModel
	bucketStoragePolicyObs
}

type ListObjsInput struct {
	Prefix        string
	MaxKeys       int
	Delimiter     string
	Origin        string
	RequestHeader string
}

type ListObjectsInput struct {
	ListObjsInput
	Bucket string
	Marker string
}

type Content struct {
	XMLName      xml.Name         `xml:"Contents"`
	Owner        Owner            `xml:"Owner"`
	ETag         string           `xml:"ETag"`
	Key          string           `xml:"Key"`
	LastModified time.Time        `xml:"LastModified"`
	Size         int64            `xml:"Size"`
	StorageClass StorageClassType `xml:"StorageClass"`
}

type ListObjectsOutput struct {
	BaseModel
	XMLName        xml.Name  `xml:"ListBucketResult"`
	Delimiter      string    `xml:"Delimiter"`
	IsTruncated    bool      `xml:"IsTruncated"`
	Marker         string    `xml:"Marker"`
	NextMarker     string    `xml:"NextMarker"`
	MaxKeys        int       `xml:"MaxKeys"`
	Name           string    `xml:"Name"`
	Prefix         string    `xml:"Prefix"`
	Contents       []Content `xml:"Contents"`
	CommonPrefixes []string  `xml:"CommonPrefixes>Prefix"`
	Location       string    `xml:"-"`
}

type ListVersionsInput struct {
	ListObjsInput
	Bucket          string
	KeyMarker       string
	VersionIdMarker string
}

type Version struct {
	DeleteMarker
	XMLName xml.Name `xml:"Version"`
	ETag    string   `xml:"ETag"`
	Size    int64    `xml:"Size"`
}

type DeleteMarker struct {
	XMLName      xml.Name         `xml:"DeleteMarker"`
	Key          string           `xml:"Key"`
	VersionId    string           `xml:"VersionId"`
	IsLatest     bool             `xml:"IsLatest"`
	LastModified time.Time        `xml:"LastModified"`
	Owner        Owner            `xml:"Owner"`
	StorageClass StorageClassType `xml:"StorageClass"`
}

type ListVersionsOutput struct {
	BaseModel
	XMLName             xml.Name       `xml:"ListVersionsResult"`
	Delimiter           string         `xml:"Delimiter"`
	IsTruncated         bool           `xml:"IsTruncated"`
	KeyMarker           string         `xml:"KeyMarker"`
	NextKeyMarker       string         `xml:"NextKeyMarker"`
	VersionIdMarker     string         `xml:"VersionIdMarker"`
	NextVersionIdMarker string         `xml:"NextVersionIdMarker"`
	MaxKeys             int            `xml:"MaxKeys"`
	Name                string         `xml:"Name"`
	Prefix              string         `xml:"Prefix"`
	Versions            []Version      `xml:"Version"`
	DeleteMarkers       []DeleteMarker `xml:"DeleteMarker"`
	CommonPrefixes      []string       `xml:"CommonPrefixes>Prefix"`
	Location            string         `xml:"-"`
}

type ListMultipartUploadsInput struct {
	Bucket         string
	Prefix         string
	MaxUploads     int
	Delimiter      string
	KeyMarker      string
	UploadIdMarker string
}

type Upload struct {
	XMLName      xml.Name         `xml:"Upload"`
	Key          string           `xml:"Key"`
	UploadId     string           `xml:"UploadId"`
	Initiated    time.Time        `xml:"Initiated"`
	StorageClass StorageClassType `xml:"StorageClass"`
	Owner        Owner            `xml:"Owner"`
	Initiator    Initiator        `xml:"Initiator"`
}

type ListMultipartUploadsOutput struct {
	BaseModel
	XMLName            xml.Name `xml:"ListMultipartUploadsResult"`
	Bucket             string   `xml:"Bucket"`
	KeyMarker          string   `xml:"KeyMarker"`
	NextKeyMarker      string   `xml:"NextKeyMarker"`
	UploadIdMarker     string   `xml:"UploadIdMarker"`
	NextUploadIdMarker string   `xml:"NextUploadIdMarker"`
	Delimiter          string   `xml:"Delimiter"`
	IsTruncated        bool     `xml:"IsTruncated"`
	MaxUploads         int      `xml:"MaxUploads"`
	Prefix             string   `xml:"Prefix"`
	Uploads            []Upload `xml:"Upload"`
	CommonPrefixes     []string `xml:"CommonPrefixes>Prefix"`
}

type BucketQuota struct {
	XMLName xml.Name `xml:"Quota"`
	Quota   int64    `xml:"StorageQuota"`
}

type SetBucketQuotaInput struct {
	Bucket string `xml:"-"`
	BucketQuota
}

type GetBucketQuotaOutput struct {
	BaseModel
	BucketQuota
}

type GetBucketStorageInfoOutput struct {
	BaseModel
	XMLName      xml.Name `xml:"GetBucketStorageInfoResult"`
	Size         int64    `xml:"Size"`
	ObjectNumber int      `xml:"ObjectNumber"`
}

type getBucketLocationOutputS3 struct {
	BaseModel
	BucketLocation
}
type getBucketLocationOutputObs struct {
	BaseModel
	bucketLocationObs
}
type GetBucketLocationOutput struct {
	BaseModel
	Location string `xml:"-"`
}

type Grantee struct {
	XMLName     xml.Name     `xml:"Grantee"`
	Type        GranteeType  `xml:"type,attr"`
	ID          string       `xml:"ID,omitempty"`
	DisplayName string       `xml:"DisplayName,omitempty"`
	URI         GroupUriType `xml:"URI,omitempty"`
}

type granteeObs struct {
	XMLName     xml.Name    `xml:"Grantee"`
	Type        GranteeType `xml:"type,attr"`
	ID          string      `xml:"ID,omitempty"`
	DisplayName string      `xml:"DisplayName,omitempty"`
	Canned      string      `xml:"Canned,omitempty"`
}

type Grant struct {
	XMLName    xml.Name       `xml:"Grant"`
	Grantee    Grantee        `xml:"Grantee"`
	Permission PermissionType `xml:"Permission"`
	Delivered  bool           `xml:"Delivered"`
}
type grantObs struct {
	XMLName    xml.Name       `xml:"Grant"`
	Grantee    granteeObs     `xml:"Grantee"`
	Permission PermissionType `xml:"Permission"`
	Delivered  bool           `xml:"Delivered"`
}

type AccessControlPolicy struct {
	XMLName   xml.Name `xml:"AccessControlPolicy"`
	Owner     Owner    `xml:"Owner"`
	Grants    []Grant  `xml:"AccessControlList>Grant"`
	Delivered string   `xml:"Delivered,omitempty"`
}

type accessControlPolicyObs struct {
	XMLName xml.Name   `xml:"AccessControlPolicy"`
	Owner   Owner      `xml:"Owner"`
	Grants  []grantObs `xml:"AccessControlList>Grant"`
}

type GetBucketAclOutput struct {
	BaseModel
	AccessControlPolicy
}

type getBucketAclOutputObs struct {
	BaseModel
	accessControlPolicyObs
}

type SetBucketAclInput struct {
	Bucket string  `xml:"-"`
	ACL    AclType `xml:"-"`
	AccessControlPolicy
}

type SetBucketPolicyInput struct {
	Bucket string
	Policy string
}

type GetBucketPolicyOutput struct {
	BaseModel
	Policy string `json:"body"`
}

type CorsRule struct {
	XMLName       xml.Name `xml:"CORSRule"`
	ID            string   `xml:"ID,omitempty"`
	AllowedOrigin []string `xml:"AllowedOrigin"`
	AllowedMethod []string `xml:"AllowedMethod"`
	AllowedHeader []string `xml:"AllowedHeader,omitempty"`
	MaxAgeSeconds int      `xml:"MaxAgeSeconds"`
	ExposeHeader  []string `xml:"ExposeHeader,omitempty"`
}

type BucketCors struct {
	XMLName   xml.Name   `xml:"CORSConfiguration"`
	CorsRules []CorsRule `xml:"CORSRule"`
}

type SetBucketCorsInput struct {
	Bucket string `xml:"-"`
	BucketCors
}

type GetBucketCorsOutput struct {
	BaseModel
	BucketCors
}

type BucketVersioningConfiguration struct {
	XMLName xml.Name             `xml:"VersioningConfiguration"`
	Status  VersioningStatusType `xml:"Status"`
}

type SetBucketVersioningInput struct {
	Bucket string `xml:"-"`
	BucketVersioningConfiguration
}

type GetBucketVersioningOutput struct {
	BaseModel
	BucketVersioningConfiguration
}

type IndexDocument struct {
	Suffix string `xml:"Suffix"`
}

type ErrorDocument struct {
	Key string `xml:"Key,omitempty"`
}

type Condition struct {
	XMLName                     xml.Name `xml:"Condition"`
	KeyPrefixEquals             string   `xml:"KeyPrefixEquals,omitempty"`
	HttpErrorCodeReturnedEquals string   `xml:"HttpErrorCodeReturnedEquals,omitempty"`
}

type Redirect struct {
	XMLName              xml.Name     `xml:"Redirect"`
	Protocol             ProtocolType `xml:"Protocol,omitempty"`
	HostName             string       `xml:"HostName,omitempty"`
	ReplaceKeyPrefixWith string       `xml:"ReplaceKeyPrefixWith,omitempty"`
	ReplaceKeyWith       string       `xml:"ReplaceKeyWith,omitempty"`
	HttpRedirectCode     string       `xml:"HttpRedirectCode,omitempty"`
}

type RoutingRule struct {
	XMLName   xml.Name  `xml:"RoutingRule"`
	Condition Condition `xml:"Condition,omitempty"`
	Redirect  Redirect  `xml:"Redirect"`
}

type RedirectAllRequestsTo struct {
	XMLName  xml.Name     `xml:"RedirectAllRequestsTo"`
	Protocol ProtocolType `xml:"Protocol,omitempty"`
	HostName string       `xml:"HostName"`
}

type BucketWebsiteConfiguration struct {
	XMLName               xml.Name              `xml:"WebsiteConfiguration"`
	RedirectAllRequestsTo RedirectAllRequestsTo `xml:"RedirectAllRequestsTo,omitempty"`
	IndexDocument         IndexDocument         `xml:"IndexDocument,omitempty"`
	ErrorDocument         ErrorDocument         `xml:"ErrorDocument,omitempty"`
	RoutingRules          []RoutingRule         `xml:"RoutingRules>RoutingRule,omitempty"`
}

type SetBucketWebsiteConfigurationInput struct {
	Bucket string `xml:"-"`
	BucketWebsiteConfiguration
}

type GetBucketWebsiteConfigurationOutput struct {
	BaseModel
	BucketWebsiteConfiguration
}

type GetBucketMetadataInput struct {
	Bucket        string
	Origin        string
	RequestHeader string
}

type SetObjectMetadataInput struct {
	Bucket                  string
	Key                     string
	VersionId               string
	MetadataDirective       MetadataDirectiveType
	CacheControl            string
	ContentDisposition      string
	ContentEncoding         string
	ContentLanguage         string
	ContentType             string
	Expires                 string
	WebsiteRedirectLocation string
	StorageClass            StorageClassType
	Metadata                map[string]string
}

type SetObjectMetadataOutput struct {
	BaseModel
	MetadataDirective       MetadataDirectiveType
	CacheControl            string
	ContentDisposition      string
	ContentEncoding         string
	ContentLanguage         string
	ContentType             string
	Expires                 string
	WebsiteRedirectLocation string
	StorageClass            StorageClassType
	Metadata                map[string]string
}

type GetBucketMetadataOutput struct {
	BaseModel
	StorageClass  StorageClassType
	Location      string
	Version       string
	AllowOrigin   string
	AllowMethod   string
	AllowHeader   string
	MaxAgeSeconds int
	ExposeHeader  string
	Epid          string
}

type BucketLoggingStatus struct {
	XMLName      xml.Name `xml:"BucketLoggingStatus"`
	Agency       string   `xml:"Agency,omitempty"`
	TargetBucket string   `xml:"LoggingEnabled>TargetBucket,omitempty"`
	TargetPrefix string   `xml:"LoggingEnabled>TargetPrefix,omitempty"`
	TargetGrants []Grant  `xml:"LoggingEnabled>TargetGrants>Grant,omitempty"`
}

type SetBucketLoggingConfigurationInput struct {
	Bucket string `xml:"-"`
	BucketLoggingStatus
}

type GetBucketLoggingConfigurationOutput struct {
	BaseModel
	BucketLoggingStatus
}

type Transition struct {
	XMLName      xml.Name         `xml:"Transition"`
	Date         time.Time        `xml:"Date,omitempty"`
	Days         int              `xml:"Days,omitempty"`
	StorageClass StorageClassType `xml:"StorageClass"`
}

type Expiration struct {
	XMLName xml.Name  `xml:"Expiration"`
	Date    time.Time `xml:"Date,omitempty"`
	Days    int       `xml:"Days,omitempty"`
}

type NoncurrentVersionTransition struct {
	XMLName        xml.Name         `xml:"NoncurrentVersionTransition"`
	NoncurrentDays int              `xml:"NoncurrentDays"`
	StorageClass   StorageClassType `xml:"StorageClass"`
}

type NoncurrentVersionExpiration struct {
	XMLName        xml.Name `xml:"NoncurrentVersionExpiration"`
	NoncurrentDays int      `xml:"NoncurrentDays"`
}

type LifecycleRule struct {
	ID                           string                        `xml:"ID,omitempty"`
	Prefix                       string                        `xml:"Prefix"`
	Status                       RuleStatusType                `xml:"Status"`
	Transitions                  []Transition                  `xml:"Transition,omitempty"`
	Expiration                   Expiration                    `xml:"Expiration,omitempty"`
	NoncurrentVersionTransitions []NoncurrentVersionTransition `xml:"NoncurrentVersionTransition,omitempty"`
	NoncurrentVersionExpiration  NoncurrentVersionExpiration   `xml:"NoncurrentVersionExpiration,omitempty"`
}

type BucketLifecyleConfiguration struct {
	XMLName        xml.Name        `xml:"LifecycleConfiguration"`
	LifecycleRules []LifecycleRule `xml:"Rule"`
}

type SetBucketLifecycleConfigurationInput struct {
	Bucket string `xml:"-"`
	BucketLifecyleConfiguration
}

type GetBucketLifecycleConfigurationOutput struct {
	BaseModel
	BucketLifecyleConfiguration
}

type Tag struct {
	XMLName xml.Name `xml:"Tag"`
	Key     string   `xml:"Key"`
	Value   string   `xml:"Value"`
}

type BucketTagging struct {
	XMLName xml.Name `xml:"Tagging"`
	Tags    []Tag    `xml:"TagSet>Tag"`
}

type SetBucketTaggingInput struct {
	Bucket string `xml:"-"`
	BucketTagging
}

type GetBucketTaggingOutput struct {
	BaseModel
	BucketTagging
}

type FilterRule struct {
	XMLName xml.Name `xml:"FilterRule"`
	Name    string   `xml:"Name,omitempty"`
	Value   string   `xml:"Value,omitempty"`
}

type TopicConfiguration struct {
	XMLName     xml.Name     `xml:"TopicConfiguration"`
	ID          string       `xml:"Id,omitempty"`
	Topic       string       `xml:"Topic"`
	Events      []EventType  `xml:"Event"`
	FilterRules []FilterRule `xml:"Filter>Object>FilterRule"`
}

type BucketNotification struct {
	XMLName             xml.Name             `xml:"NotificationConfiguration"`
	TopicConfigurations []TopicConfiguration `xml:"TopicConfiguration"`
}

type SetBucketNotificationInput struct {
	Bucket string `xml:"-"`
	BucketNotification
}

type topicConfigurationS3 struct {
	XMLName     xml.Name     `xml:"TopicConfiguration"`
	ID          string       `xml:"Id,omitempty"`
	Topic       string       `xml:"Topic"`
	Events      []string     `xml:"Event"`
	FilterRules []FilterRule `xml:"Filter>S3Key>FilterRule"`
}

type bucketNotificationS3 struct {
	XMLName             xml.Name               `xml:"NotificationConfiguration"`
	TopicConfigurations []topicConfigurationS3 `xml:"TopicConfiguration"`
}

type getBucketNotificationOutputS3 struct {
	BaseModel
	bucketNotificationS3
}

type GetBucketNotificationOutput struct {
	BaseModel
	BucketNotification
}

type DeleteObjectInput struct {
	Bucket    string
	Key       string
	VersionId string
}

type DeleteObjectOutput struct {
	BaseModel
	VersionId    string
	DeleteMarker bool
}

type ObjectToDelete struct {
	XMLName   xml.Name `xml:"Object"`
	Key       string   `xml:"Key"`
	VersionId string   `xml:"VersionId,omitempty"`
}

type DeleteObjectsInput struct {
	Bucket  string           `xml:"-"`
	XMLName xml.Name         `xml:"Delete"`
	Quiet   bool             `xml:"Quiet,omitempty"`
	Objects []ObjectToDelete `xml:"Object"`
}

type Deleted struct {
	XMLName               xml.Name `xml:"Deleted"`
	Key                   string   `xml:"Key"`
	VersionId             string   `xml:"VersionId"`
	DeleteMarker          bool     `xml:"DeleteMarker"`
	DeleteMarkerVersionId string   `xml:"DeleteMarkerVersionId"`
}

type Error struct {
	XMLName   xml.Name `xml:"Error"`
	Key       string   `xml:"Key"`
	VersionId string   `xml:"VersionId"`
	Code      string   `xml:"Code"`
	Message   string   `xml:"Message"`
}

type DeleteObjectsOutput struct {
	BaseModel
	XMLName  xml.Name  `xml:"DeleteResult"`
	Deleteds []Deleted `xml:"Deleted"`
	Errors   []Error   `xml:"Error"`
}

type SetObjectAclInput struct {
	Bucket    string  `xml:"-"`
	Key       string  `xml:"-"`
	VersionId string  `xml:"-"`
	ACL       AclType `xml:"-"`
	AccessControlPolicy
}

type GetObjectAclInput struct {
	Bucket    string
	Key       string
	VersionId string
}

type GetObjectAclOutput struct {
	BaseModel
	VersionId string
	AccessControlPolicy
}

type RestoreObjectInput struct {
	Bucket    string          `xml:"-"`
	Key       string          `xml:"-"`
	VersionId string          `xml:"-"`
	XMLName   xml.Name        `xml:"RestoreRequest"`
	Days      int             `xml:"Days"`
	Tier      RestoreTierType `xml:"GlacierJobParameters>Tier,omitempty"`
}

type ISseHeader interface {
	GetEncryption() string
	GetKey() string
}

type SseKmsHeader struct {
	Encryption string
	Key        string
	isObs      bool
}

type SseCHeader struct {
	Encryption string
	Key        string
	KeyMD5     string
}

type GetObjectMetadataInput struct {
	Bucket        string
	Key           string
	VersionId     string
	Origin        string
	RequestHeader string
	SseHeader     ISseHeader
}

type GetObjectMetadataOutput struct {
	BaseModel
	VersionId               string
	WebsiteRedirectLocation string
	Expiration              string
	Restore                 string
	ObjectType              string
	NextAppendPosition      string
	StorageClass            StorageClassType
	ContentLength           int64
	ContentType             string
	ETag                    string
	AllowOrigin             string
	AllowHeader             string
	AllowMethod             string
	ExposeHeader            string
	MaxAgeSeconds           int
	LastModified            time.Time
	SseHeader               ISseHeader
	Metadata                map[string]string
}

type GetObjectInput struct {
	GetObjectMetadataInput
	IfMatch                    string
	IfNoneMatch                string
	IfUnmodifiedSince          time.Time
	IfModifiedSince            time.Time
	RangeStart                 int64
	RangeEnd                   int64
	ImageProcess               string
	ResponseCacheControl       string
	ResponseContentDisposition string
	ResponseContentEncoding    string
	ResponseContentLanguage    string
	ResponseContentType        string
	ResponseExpires            string
}

type GetObjectOutput struct {
	GetObjectMetadataOutput
	DeleteMarker       bool
	CacheControl       string
	ContentDisposition string
	ContentEncoding    string
	ContentLanguage    string
	Expires            string
	Body               io.ReadCloser
}

type ObjectOperationInput struct {
	Bucket                  string
	Key                     string
	ACL                     AclType
	GrantReadId             string
	GrantReadAcpId          string
	GrantWriteAcpId         string
	GrantFullControlId      string
	StorageClass            StorageClassType
	WebsiteRedirectLocation string
	Expires                 int64
	SseHeader               ISseHeader
	Metadata                map[string]string
}

type PutObjectBasicInput struct {
	ObjectOperationInput
	ContentType   string
	ContentMD5    string
	ContentLength int64
}

type PutObjectInput struct {
	PutObjectBasicInput
	Body io.Reader
}

type PutFileInput struct {
	PutObjectBasicInput
	SourceFile string
}

type PutObjectOutput struct {
	BaseModel
	VersionId    string
	SseHeader    ISseHeader
	StorageClass StorageClassType
	ETag         string
}

type CopyObjectInput struct {
	ObjectOperationInput
	CopySourceBucket            string
	CopySourceKey               string
	CopySourceVersionId         string
	CopySourceIfMatch           string
	CopySourceIfNoneMatch       string
	CopySourceIfUnmodifiedSince time.Time
	CopySourceIfModifiedSince   time.Time
	SourceSseHeader             ISseHeader
	CacheControl                string
	ContentDisposition          string
	ContentEncoding             string
	ContentLanguage             string
	ContentType                 string
	Expires                     string
	MetadataDirective           MetadataDirectiveType
	SuccessActionRedirect       string
}

type CopyObjectOutput struct {
	BaseModel
	CopySourceVersionId string     `xml:"-"`
	VersionId           string     `xml:"-"`
	SseHeader           ISseHeader `xml:"-"`
	XMLName             xml.Name   `xml:"CopyObjectResult"`
	LastModified        time.Time  `xml:"LastModified"`
	ETag                string     `xml:"ETag"`
}

type AbortMultipartUploadInput struct {
	Bucket   string
	Key      string
	UploadId string
}

type InitiateMultipartUploadInput struct {
	ObjectOperationInput
	ContentType string
}

type InitiateMultipartUploadOutput struct {
	BaseModel
	XMLName   xml.Name `xml:"InitiateMultipartUploadResult"`
	Bucket    string   `xml:"Bucket"`
	Key       string   `xml:"Key"`
	UploadId  string   `xml:"UploadId"`
	SseHeader ISseHeader
}

type UploadPartInput struct {
	Bucket     string
	Key        string
	PartNumber int
	UploadId   string
	ContentMD5 string
	SseHeader  ISseHeader
	Body       io.Reader
	SourceFile string
	Offset     int64
	PartSize   int64
}

type UploadPartOutput struct {
	BaseModel
	PartNumber int
	ETag       string
	SseHeader  ISseHeader
}

type Part struct {
	XMLName      xml.Name  `xml:"Part"`
	PartNumber   int       `xml:"PartNumber"`
	ETag         string    `xml:"ETag"`
	LastModified time.Time `xml:"LastModified,omitempty"`
	Size         int64     `xml:"Size,omitempty"`
}

type CompleteMultipartUploadInput struct {
	Bucket   string   `xml:"-"`
	Key      string   `xml:"-"`
	UploadId string   `xml:"-"`
	XMLName  xml.Name `xml:"CompleteMultipartUpload"`
	Parts    []Part   `xml:"Part"`
}

type CompleteMultipartUploadOutput struct {
	BaseModel
	VersionId string     `xml:"-"`
	SseHeader ISseHeader `xml:"-"`
	XMLName   xml.Name   `xml:"CompleteMultipartUploadResult"`
	Location  string     `xml:"Location"`
	Bucket    string     `xml:"Bucket"`
	Key       string     `xml:"Key"`
	ETag      string     `xml:"ETag"`
}

type ListPartsInput struct {
	Bucket           string
	Key              string
	UploadId         string
	MaxParts         int
	PartNumberMarker int
}

type ListPartsOutput struct {
	BaseModel
	XMLName              xml.Name         `xml:"ListPartsResult"`
	Bucket               string           `xml:"Bucket"`
	Key                  string           `xml:"Key"`
	UploadId             string           `xml:"UploadId"`
	PartNumberMarker     int              `xml:"PartNumberMarker"`
	NextPartNumberMarker int              `xml:"NextPartNumberMarker"`
	MaxParts             int              `xml:"MaxParts"`
	IsTruncated          bool             `xml:"IsTruncated"`
	StorageClass         StorageClassType `xml:"StorageClass"`
	Initiator            Initiator        `xml:"Initiator"`
	Owner                Owner            `xml:"Owner"`
	Parts                []Part           `xml:"Part"`
}

type CopyPartInput struct {
	Bucket               string
	Key                  string
	UploadId             string
	PartNumber           int
	CopySourceBucket     string
	CopySourceKey        string
	CopySourceVersionId  string
	CopySourceRangeStart int64
	CopySourceRangeEnd   int64
	SseHeader            ISseHeader
	SourceSseHeader      ISseHeader
}

type CopyPartOutput struct {
	BaseModel
	XMLName      xml.Name   `xml:"CopyPartResult"`
	PartNumber   int        `xml:"-"`
	ETag         string     `xml:"ETag"`
	LastModified time.Time  `xml:"LastModified"`
	SseHeader    ISseHeader `xml:"-"`
}

type CreateSignedUrlInput struct {
	Method      HttpMethodType
	Bucket      string
	Key         string
	SubResource SubResourceType
	Expires     int
	Headers     map[string]string
	QueryParams map[string]string
}

type CreateSignedUrlOutput struct {
	SignedUrl                  string
	ActualSignedRequestHeaders http.Header
}

type CreateBrowserBasedSignatureInput struct {
	Bucket     string
	Key        string
	Expires    int
	FormParams map[string]string
}

type CreateBrowserBasedSignatureOutput struct {
	OriginPolicy string
	Policy       string
	Algorithm    string
	Credential   string
	Date         string
	Signature    string
}
