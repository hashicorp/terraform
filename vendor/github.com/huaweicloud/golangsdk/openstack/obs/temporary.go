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
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

func (obsClient ObsClient) CreateSignedUrl(input *CreateSignedUrlInput) (output *CreateSignedUrlOutput, err error) {
	if input == nil {
		return nil, errors.New("CreateSignedUrlInput is nil")
	}

	params := make(map[string]string, len(input.QueryParams))
	for key, value := range input.QueryParams {
		params[key] = value
	}

	if input.SubResource != "" {
		params[string(input.SubResource)] = ""
	}

	headers := make(map[string][]string, len(input.Headers))
	for key, value := range input.Headers {
		headers[key] = []string{value}
	}

	if input.Expires <= 0 {
		input.Expires = 300
	}

	requestUrl, err := obsClient.doAuthTemporary(string(input.Method), input.Bucket, input.Key, params, headers, int64(input.Expires))
	if err != nil {
		return nil, err
	}

	output = &CreateSignedUrlOutput{
		SignedUrl:                  requestUrl,
		ActualSignedRequestHeaders: headers,
	}
	return
}

func (obsClient ObsClient) CreateBrowserBasedSignature(input *CreateBrowserBasedSignatureInput) (output *CreateBrowserBasedSignatureOutput, err error) {
	if input == nil {
		return nil, errors.New("CreateBrowserBasedSignatureInput is nil")
	}

	params := make(map[string]string, len(input.FormParams))
	for key, value := range input.FormParams {
		params[key] = value
	}

	date := time.Now().UTC()
	shortDate := date.Format(SHORT_DATE_FORMAT)
	longDate := date.Format(LONG_DATE_FORMAT)

	credential, _ := getCredential(obsClient.conf.securityProvider.ak, obsClient.conf.region, shortDate)

	if input.Expires <= 0 {
		input.Expires = 300
	}

	expiration := date.Add(time.Second * time.Duration(input.Expires)).Format(ISO8601_DATE_FORMAT)
	params[PARAM_ALGORITHM_AMZ_CAMEL] = V4_HASH_PREFIX
	params[PARAM_CREDENTIAL_AMZ_CAMEL] = credential
	params[PARAM_DATE_AMZ_CAMEL] = longDate

	if obsClient.conf.securityProvider.securityToken != "" {
		if obsClient.conf.signature == SignatureObs {
			params[HEADER_STS_TOKEN_OBS] = obsClient.conf.securityProvider.securityToken
		} else {
			params[HEADER_STS_TOKEN_AMZ] = obsClient.conf.securityProvider.securityToken
		}
	}

	matchAnyBucket := true
	matchAnyKey := true
	count := 5
	if bucket := strings.TrimSpace(input.Bucket); bucket != "" {
		params["bucket"] = bucket
		matchAnyBucket = false
		count--
	}

	if key := strings.TrimSpace(input.Key); key != "" {
		params["key"] = key
		matchAnyKey = false
		count--
	}

	originPolicySlice := make([]string, 0, len(params)+count)
	originPolicySlice = append(originPolicySlice, fmt.Sprintf("{\"expiration\":\"%s\",", expiration))
	originPolicySlice = append(originPolicySlice, "\"conditions\":[")
	for key, value := range params {
		if _key := strings.TrimSpace(strings.ToLower(key)); _key != "" {
			originPolicySlice = append(originPolicySlice, fmt.Sprintf("{\"%s\":\"%s\"},", _key, value))
		}
	}

	if matchAnyBucket {
		originPolicySlice = append(originPolicySlice, "[\"starts-with\", \"$bucket\", \"\"],")
	}

	if matchAnyKey {
		originPolicySlice = append(originPolicySlice, "[\"starts-with\", \"$key\", \"\"],")
	}

	originPolicySlice = append(originPolicySlice, "]}")

	originPolicy := strings.Join(originPolicySlice, "")
	policy := Base64Encode([]byte(originPolicy))
	signature := getSignature(policy, obsClient.conf.securityProvider.sk, obsClient.conf.region, shortDate)

	output = &CreateBrowserBasedSignatureOutput{
		OriginPolicy: originPolicy,
		Policy:       policy,
		Algorithm:    params[PARAM_ALGORITHM_AMZ_CAMEL],
		Credential:   params[PARAM_CREDENTIAL_AMZ_CAMEL],
		Date:         params[PARAM_DATE_AMZ_CAMEL],
		Signature:    signature,
	}
	return
}

func (obsClient ObsClient) ListBucketsWithSignedUrl(signedUrl string, actualSignedRequestHeaders http.Header) (output *ListBucketsOutput, err error) {
	output = &ListBucketsOutput{}
	err = obsClient.doHttpWithSignedUrl("ListBuckets", HTTP_GET, signedUrl, actualSignedRequestHeaders, nil, output, true)
	if err != nil {
		output = nil
	}
	return
}

func (obsClient ObsClient) CreateBucketWithSignedUrl(signedUrl string, actualSignedRequestHeaders http.Header, data io.Reader) (output *BaseModel, err error) {
	output = &BaseModel{}
	err = obsClient.doHttpWithSignedUrl("CreateBucket", HTTP_PUT, signedUrl, actualSignedRequestHeaders, data, output, true)
	if err != nil {
		output = nil
	}
	return
}

func (obsClient ObsClient) DeleteBucketWithSignedUrl(signedUrl string, actualSignedRequestHeaders http.Header) (output *BaseModel, err error) {
	output = &BaseModel{}
	err = obsClient.doHttpWithSignedUrl("DeleteBucket", HTTP_DELETE, signedUrl, actualSignedRequestHeaders, nil, output, true)
	if err != nil {
		output = nil
	}
	return
}

func (obsClient ObsClient) SetBucketStoragePolicyWithSignedUrl(signedUrl string, actualSignedRequestHeaders http.Header, data io.Reader) (output *BaseModel, err error) {
	output = &BaseModel{}
	err = obsClient.doHttpWithSignedUrl("SetBucketStoragePolicy", HTTP_PUT, signedUrl, actualSignedRequestHeaders, data, output, true)
	if err != nil {
		output = nil
	}
	return
}

func (obsClient ObsClient) GetBucketStoragePolicyWithSignedUrl(signedUrl string, actualSignedRequestHeaders http.Header) (output *GetBucketStoragePolicyOutput, err error) {
	output = &GetBucketStoragePolicyOutput{}
	err = obsClient.doHttpWithSignedUrl("GetBucketStoragePolicy", HTTP_GET, signedUrl, actualSignedRequestHeaders, nil, output, true)
	if err != nil {
		output = nil
	}
	return
}

func (obsClient ObsClient) ListObjectsWithSignedUrl(signedUrl string, actualSignedRequestHeaders http.Header) (output *ListObjectsOutput, err error) {
	output = &ListObjectsOutput{}
	err = obsClient.doHttpWithSignedUrl("ListObjects", HTTP_GET, signedUrl, actualSignedRequestHeaders, nil, output, true)
	if err != nil {
		output = nil
	} else {
		if location, ok := output.ResponseHeaders[HEADER_BUCKET_REGION]; ok {
			output.Location = location[0]
		}
	}
	return
}

func (obsClient ObsClient) ListVersionsWithSignedUrl(signedUrl string, actualSignedRequestHeaders http.Header) (output *ListVersionsOutput, err error) {
	output = &ListVersionsOutput{}
	err = obsClient.doHttpWithSignedUrl("ListVersions", HTTP_GET, signedUrl, actualSignedRequestHeaders, nil, output, true)
	if err != nil {
		output = nil
	} else {
		if location, ok := output.ResponseHeaders[HEADER_BUCKET_REGION]; ok {
			output.Location = location[0]
		}
	}
	return
}

func (obsClient ObsClient) ListMultipartUploadsWithSignedUrl(signedUrl string, actualSignedRequestHeaders http.Header) (output *ListMultipartUploadsOutput, err error) {
	output = &ListMultipartUploadsOutput{}
	err = obsClient.doHttpWithSignedUrl("ListMultipartUploads", HTTP_GET, signedUrl, actualSignedRequestHeaders, nil, output, true)
	if err != nil {
		output = nil
	}
	return
}

func (obsClient ObsClient) SetBucketQuotaWithSignedUrl(signedUrl string, actualSignedRequestHeaders http.Header, data io.Reader) (output *BaseModel, err error) {
	output = &BaseModel{}
	err = obsClient.doHttpWithSignedUrl("SetBucketQuota", HTTP_PUT, signedUrl, actualSignedRequestHeaders, data, output, true)
	if err != nil {
		output = nil
	}
	return
}

func (obsClient ObsClient) GetBucketQuotaWithSignedUrl(signedUrl string, actualSignedRequestHeaders http.Header) (output *GetBucketQuotaOutput, err error) {
	output = &GetBucketQuotaOutput{}
	err = obsClient.doHttpWithSignedUrl("GetBucketQuota", HTTP_GET, signedUrl, actualSignedRequestHeaders, nil, output, true)
	if err != nil {
		output = nil
	}
	return
}

func (obsClient ObsClient) HeadBucketWithSignedUrl(signedUrl string, actualSignedRequestHeaders http.Header) (output *BaseModel, err error) {
	output = &BaseModel{}
	err = obsClient.doHttpWithSignedUrl("HeadBucket", HTTP_HEAD, signedUrl, actualSignedRequestHeaders, nil, output, true)
	if err != nil {
		output = nil
	}
	return
}

func (obsClient ObsClient) GetBucketMetadataWithSignedUrl(signedUrl string, actualSignedRequestHeaders http.Header) (output *GetBucketMetadataOutput, err error) {
	output = &GetBucketMetadataOutput{}
	err = obsClient.doHttpWithSignedUrl("GetBucketMetadata", HTTP_HEAD, signedUrl, actualSignedRequestHeaders, nil, output, true)
	if err != nil {
		output = nil
	} else {
		ParseGetBucketMetadataOutput(output)
	}
	return
}

func (obsClient ObsClient) GetBucketStorageInfoWithSignedUrl(signedUrl string, actualSignedRequestHeaders http.Header) (output *GetBucketStorageInfoOutput, err error) {
	output = &GetBucketStorageInfoOutput{}
	err = obsClient.doHttpWithSignedUrl("GetBucketStorageInfo", HTTP_GET, signedUrl, actualSignedRequestHeaders, nil, output, true)
	if err != nil {
		output = nil
	}
	return
}

func (obsClient ObsClient) GetBucketLocationWithSignedUrl(signedUrl string, actualSignedRequestHeaders http.Header) (output *GetBucketLocationOutput, err error) {
	output = &GetBucketLocationOutput{}
	err = obsClient.doHttpWithSignedUrl("GetBucketLocation", HTTP_GET, signedUrl, actualSignedRequestHeaders, nil, output, true)
	if err != nil {
		output = nil
	}
	return
}

func (obsClient ObsClient) SetBucketAclWithSignedUrl(signedUrl string, actualSignedRequestHeaders http.Header, data io.Reader) (output *BaseModel, err error) {
	output = &BaseModel{}
	err = obsClient.doHttpWithSignedUrl("SetBucketAcl", HTTP_PUT, signedUrl, actualSignedRequestHeaders, data, output, true)
	if err != nil {
		output = nil
	}
	return
}

func (obsClient ObsClient) GetBucketAclWithSignedUrl(signedUrl string, actualSignedRequestHeaders http.Header) (output *GetBucketAclOutput, err error) {
	output = &GetBucketAclOutput{}
	err = obsClient.doHttpWithSignedUrl("GetBucketAcl", HTTP_GET, signedUrl, actualSignedRequestHeaders, nil, output, true)
	if err != nil {
		output = nil
	}
	return
}

func (obsClient ObsClient) SetBucketPolicyWithSignedUrl(signedUrl string, actualSignedRequestHeaders http.Header, data io.Reader) (output *BaseModel, err error) {
	output = &BaseModel{}
	err = obsClient.doHttpWithSignedUrl("SetBucketPolicy", HTTP_PUT, signedUrl, actualSignedRequestHeaders, data, output, true)
	if err != nil {
		output = nil
	}
	return
}

func (obsClient ObsClient) GetBucketPolicyWithSignedUrl(signedUrl string, actualSignedRequestHeaders http.Header) (output *GetBucketPolicyOutput, err error) {
	output = &GetBucketPolicyOutput{}
	err = obsClient.doHttpWithSignedUrl("GetBucketPolicy", HTTP_GET, signedUrl, actualSignedRequestHeaders, nil, output, false)
	if err != nil {
		output = nil
	}
	return
}

func (obsClient ObsClient) DeleteBucketPolicyWithSignedUrl(signedUrl string, actualSignedRequestHeaders http.Header) (output *BaseModel, err error) {
	output = &BaseModel{}
	err = obsClient.doHttpWithSignedUrl("DeleteBucketPolicy", HTTP_DELETE, signedUrl, actualSignedRequestHeaders, nil, output, true)
	if err != nil {
		output = nil
	}
	return
}

func (obsClient ObsClient) SetBucketCorsWithSignedUrl(signedUrl string, actualSignedRequestHeaders http.Header, data io.Reader) (output *BaseModel, err error) {
	output = &BaseModel{}
	err = obsClient.doHttpWithSignedUrl("SetBucketCors", HTTP_PUT, signedUrl, actualSignedRequestHeaders, data, output, true)
	if err != nil {
		output = nil
	}
	return
}

func (obsClient ObsClient) GetBucketCorsWithSignedUrl(signedUrl string, actualSignedRequestHeaders http.Header) (output *GetBucketCorsOutput, err error) {
	output = &GetBucketCorsOutput{}
	err = obsClient.doHttpWithSignedUrl("GetBucketCors", HTTP_GET, signedUrl, actualSignedRequestHeaders, nil, output, true)
	if err != nil {
		output = nil
	}
	return
}

func (obsClient ObsClient) DeleteBucketCorsWithSignedUrl(signedUrl string, actualSignedRequestHeaders http.Header) (output *BaseModel, err error) {
	output = &BaseModel{}
	err = obsClient.doHttpWithSignedUrl("DeleteBucketCors", HTTP_DELETE, signedUrl, actualSignedRequestHeaders, nil, output, true)
	if err != nil {
		output = nil
	}
	return
}

func (obsClient ObsClient) SetBucketVersioningWithSignedUrl(signedUrl string, actualSignedRequestHeaders http.Header, data io.Reader) (output *BaseModel, err error) {
	output = &BaseModel{}
	err = obsClient.doHttpWithSignedUrl("SetBucketVersioning", HTTP_PUT, signedUrl, actualSignedRequestHeaders, data, output, true)
	if err != nil {
		output = nil
	}
	return
}

func (obsClient ObsClient) GetBucketVersioningWithSignedUrl(signedUrl string, actualSignedRequestHeaders http.Header) (output *GetBucketVersioningOutput, err error) {
	output = &GetBucketVersioningOutput{}
	err = obsClient.doHttpWithSignedUrl("GetBucketVersioning", HTTP_GET, signedUrl, actualSignedRequestHeaders, nil, output, true)
	if err != nil {
		output = nil
	}
	return
}

func (obsClient ObsClient) SetBucketWebsiteConfigurationWithSignedUrl(signedUrl string, actualSignedRequestHeaders http.Header, data io.Reader) (output *BaseModel, err error) {
	output = &BaseModel{}
	err = obsClient.doHttpWithSignedUrl("SetBucketWebsiteConfiguration", HTTP_PUT, signedUrl, actualSignedRequestHeaders, data, output, true)
	if err != nil {
		output = nil
	}
	return
}

func (obsClient ObsClient) GetBucketWebsiteConfigurationWithSignedUrl(signedUrl string, actualSignedRequestHeaders http.Header) (output *GetBucketWebsiteConfigurationOutput, err error) {
	output = &GetBucketWebsiteConfigurationOutput{}
	err = obsClient.doHttpWithSignedUrl("GetBucketWebsiteConfiguration", HTTP_GET, signedUrl, actualSignedRequestHeaders, nil, output, true)
	if err != nil {
		output = nil
	}
	return
}

func (obsClient ObsClient) DeleteBucketWebsiteConfigurationWithSignedUrl(signedUrl string, actualSignedRequestHeaders http.Header) (output *BaseModel, err error) {
	output = &BaseModel{}
	err = obsClient.doHttpWithSignedUrl("DeleteBucketWebsiteConfiguration", HTTP_DELETE, signedUrl, actualSignedRequestHeaders, nil, output, true)
	if err != nil {
		output = nil
	}
	return
}

func (obsClient ObsClient) SetBucketLoggingConfigurationWithSignedUrl(signedUrl string, actualSignedRequestHeaders http.Header, data io.Reader) (output *BaseModel, err error) {
	output = &BaseModel{}
	err = obsClient.doHttpWithSignedUrl("SetBucketLoggingConfiguration", HTTP_PUT, signedUrl, actualSignedRequestHeaders, data, output, true)
	if err != nil {
		output = nil
	}
	return
}

func (obsClient ObsClient) GetBucketLoggingConfigurationWithSignedUrl(signedUrl string, actualSignedRequestHeaders http.Header) (output *GetBucketLoggingConfigurationOutput, err error) {
	output = &GetBucketLoggingConfigurationOutput{}
	err = obsClient.doHttpWithSignedUrl("GetBucketLoggingConfiguration", HTTP_GET, signedUrl, actualSignedRequestHeaders, nil, output, true)
	if err != nil {
		output = nil
	}
	return
}

func (obsClient ObsClient) SetBucketLifecycleConfigurationWithSignedUrl(signedUrl string, actualSignedRequestHeaders http.Header, data io.Reader) (output *BaseModel, err error) {
	output = &BaseModel{}
	err = obsClient.doHttpWithSignedUrl("SetBucketLifecycleConfiguration", HTTP_PUT, signedUrl, actualSignedRequestHeaders, data, output, true)
	if err != nil {
		output = nil
	}
	return
}

func (obsClient ObsClient) GetBucketLifecycleConfigurationWithSignedUrl(signedUrl string, actualSignedRequestHeaders http.Header) (output *GetBucketLifecycleConfigurationOutput, err error) {
	output = &GetBucketLifecycleConfigurationOutput{}
	err = obsClient.doHttpWithSignedUrl("GetBucketLifecycleConfiguration", HTTP_GET, signedUrl, actualSignedRequestHeaders, nil, output, true)
	if err != nil {
		output = nil
	}
	return
}

func (obsClient ObsClient) DeleteBucketLifecycleConfigurationWithSignedUrl(signedUrl string, actualSignedRequestHeaders http.Header) (output *BaseModel, err error) {
	output = &BaseModel{}
	err = obsClient.doHttpWithSignedUrl("DeleteBucketLifecycleConfiguration", HTTP_DELETE, signedUrl, actualSignedRequestHeaders, nil, output, true)
	if err != nil {
		output = nil
	}
	return
}

func (obsClient ObsClient) SetBucketTaggingWithSignedUrl(signedUrl string, actualSignedRequestHeaders http.Header, data io.Reader) (output *BaseModel, err error) {
	output = &BaseModel{}
	err = obsClient.doHttpWithSignedUrl("SetBucketTagging", HTTP_PUT, signedUrl, actualSignedRequestHeaders, data, output, true)
	if err != nil {
		output = nil
	}
	return
}

func (obsClient ObsClient) GetBucketTaggingWithSignedUrl(signedUrl string, actualSignedRequestHeaders http.Header) (output *GetBucketTaggingOutput, err error) {
	output = &GetBucketTaggingOutput{}
	err = obsClient.doHttpWithSignedUrl("GetBucketTagging", HTTP_GET, signedUrl, actualSignedRequestHeaders, nil, output, true)
	if err != nil {
		output = nil
	}
	return
}

func (obsClient ObsClient) DeleteBucketTaggingWithSignedUrl(signedUrl string, actualSignedRequestHeaders http.Header) (output *BaseModel, err error) {
	output = &BaseModel{}
	err = obsClient.doHttpWithSignedUrl("DeleteBucketTagging", HTTP_DELETE, signedUrl, actualSignedRequestHeaders, nil, output, true)
	if err != nil {
		output = nil
	}
	return
}

func (obsClient ObsClient) SetBucketNotificationWithSignedUrl(signedUrl string, actualSignedRequestHeaders http.Header, data io.Reader) (output *BaseModel, err error) {
	output = &BaseModel{}
	err = obsClient.doHttpWithSignedUrl("SetBucketNotification", HTTP_PUT, signedUrl, actualSignedRequestHeaders, data, output, true)
	if err != nil {
		output = nil
	}
	return
}

func (obsClient ObsClient) GetBucketNotificationWithSignedUrl(signedUrl string, actualSignedRequestHeaders http.Header) (output *GetBucketNotificationOutput, err error) {
	output = &GetBucketNotificationOutput{}
	err = obsClient.doHttpWithSignedUrl("GetBucketNotification", HTTP_GET, signedUrl, actualSignedRequestHeaders, nil, output, true)
	if err != nil {
		output = nil
	}
	return
}

func (obsClient ObsClient) DeleteObjectWithSignedUrl(signedUrl string, actualSignedRequestHeaders http.Header) (output *DeleteObjectOutput, err error) {
	output = &DeleteObjectOutput{}
	err = obsClient.doHttpWithSignedUrl("DeleteObject", HTTP_DELETE, signedUrl, actualSignedRequestHeaders, nil, output, true)
	if err != nil {
		output = nil
	} else {
		ParseDeleteObjectOutput(output)
	}
	return
}

func (obsClient ObsClient) DeleteObjectsWithSignedUrl(signedUrl string, actualSignedRequestHeaders http.Header, data io.Reader) (output *DeleteObjectsOutput, err error) {
	output = &DeleteObjectsOutput{}
	err = obsClient.doHttpWithSignedUrl("DeleteObjects", HTTP_POST, signedUrl, actualSignedRequestHeaders, data, output, true)
	if err != nil {
		output = nil
	}
	return
}

func (obsClient ObsClient) SetObjectAclWithSignedUrl(signedUrl string, actualSignedRequestHeaders http.Header, data io.Reader) (output *BaseModel, err error) {
	output = &BaseModel{}
	err = obsClient.doHttpWithSignedUrl("SetObjectAcl", HTTP_PUT, signedUrl, actualSignedRequestHeaders, data, output, true)
	if err != nil {
		output = nil
	}
	return
}

func (obsClient ObsClient) GetObjectAclWithSignedUrl(signedUrl string, actualSignedRequestHeaders http.Header) (output *GetObjectAclOutput, err error) {
	output = &GetObjectAclOutput{}
	err = obsClient.doHttpWithSignedUrl("GetObjectAcl", HTTP_GET, signedUrl, actualSignedRequestHeaders, nil, output, true)
	if err != nil {
		output = nil
	} else {
		if versionId, ok := output.ResponseHeaders[HEADER_VERSION_ID]; ok {
			output.VersionId = versionId[0]
		}
	}
	return
}

func (obsClient ObsClient) RestoreObjectWithSignedUrl(signedUrl string, actualSignedRequestHeaders http.Header, data io.Reader) (output *BaseModel, err error) {
	output = &BaseModel{}
	err = obsClient.doHttpWithSignedUrl("RestoreObject", HTTP_POST, signedUrl, actualSignedRequestHeaders, data, output, true)
	if err != nil {
		output = nil
	}
	return
}

func (obsClient ObsClient) GetObjectMetadataWithSignedUrl(signedUrl string, actualSignedRequestHeaders http.Header) (output *GetObjectMetadataOutput, err error) {
	output = &GetObjectMetadataOutput{}
	err = obsClient.doHttpWithSignedUrl("GetObjectMetadata", HTTP_HEAD, signedUrl, actualSignedRequestHeaders, nil, output, true)
	if err != nil {
		output = nil
	} else {
		ParseGetObjectMetadataOutput(output)
	}
	return
}

func (obsClient ObsClient) GetObjectWithSignedUrl(signedUrl string, actualSignedRequestHeaders http.Header) (output *GetObjectOutput, err error) {
	output = &GetObjectOutput{}
	err = obsClient.doHttpWithSignedUrl("GetObject", HTTP_GET, signedUrl, actualSignedRequestHeaders, nil, output, true)
	if err != nil {
		output = nil
	} else {
		ParseGetObjectOutput(output)
	}
	return
}

func (obsClient ObsClient) PutObjectWithSignedUrl(signedUrl string, actualSignedRequestHeaders http.Header, data io.Reader) (output *PutObjectOutput, err error) {
	output = &PutObjectOutput{}
	err = obsClient.doHttpWithSignedUrl("PutObject", HTTP_PUT, signedUrl, actualSignedRequestHeaders, data, output, true)
	if err != nil {
		output = nil
	} else {
		ParsePutObjectOutput(output)
	}
	return
}

func (obsClient ObsClient) PutFileWithSignedUrl(signedUrl string, actualSignedRequestHeaders http.Header, sourceFile string) (output *PutObjectOutput, err error) {
	var data io.Reader
	sourceFile = strings.TrimSpace(sourceFile)
	if sourceFile != "" {
		fd, err := os.Open(sourceFile)
		if err != nil {
			return nil, err
		}
		defer fd.Close()

		stat, err := fd.Stat()
		if err != nil {
			return nil, err
		}
		fileReaderWrapper := &fileReaderWrapper{filePath: sourceFile}
		fileReaderWrapper.reader = fd

		var contentLength int64
		if value, ok := actualSignedRequestHeaders[HEADER_CONTENT_LENGTH_CAMEL]; ok {
			contentLength = StringToInt64(value[0], -1)
		} else if value, ok := actualSignedRequestHeaders[HEADER_CONTENT_LENGTH]; ok {
			contentLength = StringToInt64(value[0], -1)
		} else {
			contentLength = stat.Size()
		}
		if contentLength > stat.Size() {
			return nil, errors.New("ContentLength is larger than fileSize")
		}
		fileReaderWrapper.totalCount = contentLength
		data = fileReaderWrapper
	}

	output = &PutObjectOutput{}
	err = obsClient.doHttpWithSignedUrl("PutObject", HTTP_PUT, signedUrl, actualSignedRequestHeaders, data, output, true)
	if err != nil {
		output = nil
	} else {
		ParsePutObjectOutput(output)
	}
	return
}

func (obsClient ObsClient) CopyObjectWithSignedUrl(signedUrl string, actualSignedRequestHeaders http.Header) (output *CopyObjectOutput, err error) {
	output = &CopyObjectOutput{}
	err = obsClient.doHttpWithSignedUrl("CopyObject", HTTP_PUT, signedUrl, actualSignedRequestHeaders, nil, output, true)
	if err != nil {
		output = nil
	} else {
		ParseCopyObjectOutput(output)
	}
	return
}

func (obsClient ObsClient) AbortMultipartUploadWithSignedUrl(signedUrl string, actualSignedRequestHeaders http.Header) (output *BaseModel, err error) {
	output = &BaseModel{}
	err = obsClient.doHttpWithSignedUrl("AbortMultipartUpload", HTTP_DELETE, signedUrl, actualSignedRequestHeaders, nil, output, true)
	if err != nil {
		output = nil
	}
	return
}

func (obsClient ObsClient) InitiateMultipartUploadWithSignedUrl(signedUrl string, actualSignedRequestHeaders http.Header) (output *InitiateMultipartUploadOutput, err error) {
	output = &InitiateMultipartUploadOutput{}
	err = obsClient.doHttpWithSignedUrl("InitiateMultipartUpload", HTTP_POST, signedUrl, actualSignedRequestHeaders, nil, output, true)
	if err != nil {
		output = nil
	} else {
		ParseInitiateMultipartUploadOutput(output)
	}
	return
}

func (obsClient ObsClient) UploadPartWithSignedUrl(signedUrl string, actualSignedRequestHeaders http.Header, data io.Reader) (output *UploadPartOutput, err error) {
	output = &UploadPartOutput{}
	err = obsClient.doHttpWithSignedUrl("UploadPart", HTTP_PUT, signedUrl, actualSignedRequestHeaders, data, output, true)
	if err != nil {
		output = nil
	} else {
		ParseUploadPartOutput(output)
	}
	return
}

func (obsClient ObsClient) CompleteMultipartUploadWithSignedUrl(signedUrl string, actualSignedRequestHeaders http.Header, data io.Reader) (output *CompleteMultipartUploadOutput, err error) {
	output = &CompleteMultipartUploadOutput{}
	err = obsClient.doHttpWithSignedUrl("CompleteMultipartUpload", HTTP_POST, signedUrl, actualSignedRequestHeaders, data, output, true)
	if err != nil {
		output = nil
	} else {
		ParseCompleteMultipartUploadOutput(output)
	}
	return
}

func (obsClient ObsClient) ListPartsWithSignedUrl(signedUrl string, actualSignedRequestHeaders http.Header) (output *ListPartsOutput, err error) {
	output = &ListPartsOutput{}
	err = obsClient.doHttpWithSignedUrl("ListParts", HTTP_GET, signedUrl, actualSignedRequestHeaders, nil, output, true)
	if err != nil {
		output = nil
	}
	return
}

func (obsClient ObsClient) CopyPartWithSignedUrl(signedUrl string, actualSignedRequestHeaders http.Header) (output *CopyPartOutput, err error) {
	output = &CopyPartOutput{}
	err = obsClient.doHttpWithSignedUrl("CopyPart", HTTP_PUT, signedUrl, actualSignedRequestHeaders, nil, output, true)
	if err != nil {
		output = nil
	} else {
		ParseCopyPartOutput(output)
	}
	return
}
