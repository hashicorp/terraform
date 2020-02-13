// Copyright (c) 2017-2018 THL A29 Limited, a Tencent company. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package v20180813

import (
    "encoding/json"

    tchttp "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/http"
)

type AddResourceTagRequest struct {
	*tchttp.BaseRequest

	// 标签键
	TagKey *string `json:"TagKey,omitempty" name:"TagKey"`

	// 标签值
	TagValue *string `json:"TagValue,omitempty" name:"TagValue"`

	// 资源六段式描述
	Resource *string `json:"Resource,omitempty" name:"Resource"`
}

func (r *AddResourceTagRequest) ToJsonString() string {
    b, _ := json.Marshal(r)
    return string(b)
}

func (r *AddResourceTagRequest) FromJsonString(s string) error {
    return json.Unmarshal([]byte(s), &r)
}

type AddResourceTagResponse struct {
	*tchttp.BaseResponse
	Response *struct {

		// 唯一请求 ID，每次请求都会返回。定位问题时需要提供该次请求的 RequestId。
		RequestId *string `json:"RequestId,omitempty" name:"RequestId"`
	} `json:"Response"`
}

func (r *AddResourceTagResponse) ToJsonString() string {
    b, _ := json.Marshal(r)
    return string(b)
}

func (r *AddResourceTagResponse) FromJsonString(s string) error {
    return json.Unmarshal([]byte(s), &r)
}

type CreateTagRequest struct {
	*tchttp.BaseRequest

	// 标签键
	TagKey *string `json:"TagKey,omitempty" name:"TagKey"`

	// 标签值
	TagValue *string `json:"TagValue,omitempty" name:"TagValue"`
}

func (r *CreateTagRequest) ToJsonString() string {
    b, _ := json.Marshal(r)
    return string(b)
}

func (r *CreateTagRequest) FromJsonString(s string) error {
    return json.Unmarshal([]byte(s), &r)
}

type CreateTagResponse struct {
	*tchttp.BaseResponse
	Response *struct {

		// 唯一请求 ID，每次请求都会返回。定位问题时需要提供该次请求的 RequestId。
		RequestId *string `json:"RequestId,omitempty" name:"RequestId"`
	} `json:"Response"`
}

func (r *CreateTagResponse) ToJsonString() string {
    b, _ := json.Marshal(r)
    return string(b)
}

func (r *CreateTagResponse) FromJsonString(s string) error {
    return json.Unmarshal([]byte(s), &r)
}

type DeleteResourceTagRequest struct {
	*tchttp.BaseRequest

	// 标签键
	TagKey *string `json:"TagKey,omitempty" name:"TagKey"`

	// 资源六段式描述
	Resource *string `json:"Resource,omitempty" name:"Resource"`
}

func (r *DeleteResourceTagRequest) ToJsonString() string {
    b, _ := json.Marshal(r)
    return string(b)
}

func (r *DeleteResourceTagRequest) FromJsonString(s string) error {
    return json.Unmarshal([]byte(s), &r)
}

type DeleteResourceTagResponse struct {
	*tchttp.BaseResponse
	Response *struct {

		// 唯一请求 ID，每次请求都会返回。定位问题时需要提供该次请求的 RequestId。
		RequestId *string `json:"RequestId,omitempty" name:"RequestId"`
	} `json:"Response"`
}

func (r *DeleteResourceTagResponse) ToJsonString() string {
    b, _ := json.Marshal(r)
    return string(b)
}

func (r *DeleteResourceTagResponse) FromJsonString(s string) error {
    return json.Unmarshal([]byte(s), &r)
}

type DeleteTagRequest struct {
	*tchttp.BaseRequest

	// 需要删除的标签键
	TagKey *string `json:"TagKey,omitempty" name:"TagKey"`

	// 需要删除的标签值
	TagValue *string `json:"TagValue,omitempty" name:"TagValue"`
}

func (r *DeleteTagRequest) ToJsonString() string {
    b, _ := json.Marshal(r)
    return string(b)
}

func (r *DeleteTagRequest) FromJsonString(s string) error {
    return json.Unmarshal([]byte(s), &r)
}

type DeleteTagResponse struct {
	*tchttp.BaseResponse
	Response *struct {

		// 唯一请求 ID，每次请求都会返回。定位问题时需要提供该次请求的 RequestId。
		RequestId *string `json:"RequestId,omitempty" name:"RequestId"`
	} `json:"Response"`
}

func (r *DeleteTagResponse) ToJsonString() string {
    b, _ := json.Marshal(r)
    return string(b)
}

func (r *DeleteTagResponse) FromJsonString(s string) error {
    return json.Unmarshal([]byte(s), &r)
}

type DescribeResourceTagsByResourceIdsRequest struct {
	*tchttp.BaseRequest

	// 业务类型
	ServiceType *string `json:"ServiceType,omitempty" name:"ServiceType"`

	// 资源前缀
	ResourcePrefix *string `json:"ResourcePrefix,omitempty" name:"ResourcePrefix"`

	// 资源唯一标记
	ResourceIds []*string `json:"ResourceIds,omitempty" name:"ResourceIds" list`

	// 资源所在地域
	ResourceRegion *string `json:"ResourceRegion,omitempty" name:"ResourceRegion"`

	// 数据偏移量，默认为 0, 必须为Limit参数的整数倍
	Offset *uint64 `json:"Offset,omitempty" name:"Offset"`

	// 每页大小，默认为 15
	Limit *uint64 `json:"Limit,omitempty" name:"Limit"`
}

func (r *DescribeResourceTagsByResourceIdsRequest) ToJsonString() string {
    b, _ := json.Marshal(r)
    return string(b)
}

func (r *DescribeResourceTagsByResourceIdsRequest) FromJsonString(s string) error {
    return json.Unmarshal([]byte(s), &r)
}

type DescribeResourceTagsByResourceIdsResponse struct {
	*tchttp.BaseResponse
	Response *struct {

		// 结果总数
		TotalCount *uint64 `json:"TotalCount,omitempty" name:"TotalCount"`

		// 数据位移偏量
		Offset *uint64 `json:"Offset,omitempty" name:"Offset"`

		// 每页大小
		Limit *uint64 `json:"Limit,omitempty" name:"Limit"`

		// 标签列表
		Tags []*TagResource `json:"Tags,omitempty" name:"Tags" list`

		// 唯一请求 ID，每次请求都会返回。定位问题时需要提供该次请求的 RequestId。
		RequestId *string `json:"RequestId,omitempty" name:"RequestId"`
	} `json:"Response"`
}

func (r *DescribeResourceTagsByResourceIdsResponse) ToJsonString() string {
    b, _ := json.Marshal(r)
    return string(b)
}

func (r *DescribeResourceTagsByResourceIdsResponse) FromJsonString(s string) error {
    return json.Unmarshal([]byte(s), &r)
}

type DescribeTagKeysRequest struct {
	*tchttp.BaseRequest

	// 创建者用户 Uin，不传或为空只将 Uin 作为条件查询
	CreateUin *uint64 `json:"CreateUin,omitempty" name:"CreateUin"`

	// 数据偏移量，默认为 0, 必须为Limit参数的整数倍
	Offset *uint64 `json:"Offset,omitempty" name:"Offset"`

	// 每页大小，默认为 15
	Limit *uint64 `json:"Limit,omitempty" name:"Limit"`
}

func (r *DescribeTagKeysRequest) ToJsonString() string {
    b, _ := json.Marshal(r)
    return string(b)
}

func (r *DescribeTagKeysRequest) FromJsonString(s string) error {
    return json.Unmarshal([]byte(s), &r)
}

type DescribeTagKeysResponse struct {
	*tchttp.BaseResponse
	Response *struct {

		// 结果总数
		TotalCount *uint64 `json:"TotalCount,omitempty" name:"TotalCount"`

		// 数据位移偏量
		Offset *uint64 `json:"Offset,omitempty" name:"Offset"`

		// 每页大小
		Limit *uint64 `json:"Limit,omitempty" name:"Limit"`

		// 标签列表
		Tags []*string `json:"Tags,omitempty" name:"Tags" list`

		// 唯一请求 ID，每次请求都会返回。定位问题时需要提供该次请求的 RequestId。
		RequestId *string `json:"RequestId,omitempty" name:"RequestId"`
	} `json:"Response"`
}

func (r *DescribeTagKeysResponse) ToJsonString() string {
    b, _ := json.Marshal(r)
    return string(b)
}

func (r *DescribeTagKeysResponse) FromJsonString(s string) error {
    return json.Unmarshal([]byte(s), &r)
}

type DescribeTagValuesRequest struct {
	*tchttp.BaseRequest

	// 标签键列表
	TagKeys []*string `json:"TagKeys,omitempty" name:"TagKeys" list`

	// 创建者用户 Uin，不传或为空只将 Uin 作为条件查询
	CreateUin *uint64 `json:"CreateUin,omitempty" name:"CreateUin"`

	// 数据偏移量，默认为 0, 必须为Limit参数的整数倍
	Offset *uint64 `json:"Offset,omitempty" name:"Offset"`

	// 每页大小，默认为 15
	Limit *uint64 `json:"Limit,omitempty" name:"Limit"`
}

func (r *DescribeTagValuesRequest) ToJsonString() string {
    b, _ := json.Marshal(r)
    return string(b)
}

func (r *DescribeTagValuesRequest) FromJsonString(s string) error {
    return json.Unmarshal([]byte(s), &r)
}

type DescribeTagValuesResponse struct {
	*tchttp.BaseResponse
	Response *struct {

		// 结果总数
		TotalCount *uint64 `json:"TotalCount,omitempty" name:"TotalCount"`

		// 数据位移偏量
		Offset *uint64 `json:"Offset,omitempty" name:"Offset"`

		// 每页大小
		Limit *uint64 `json:"Limit,omitempty" name:"Limit"`

		// 标签列表
		Tags []*Tag `json:"Tags,omitempty" name:"Tags" list`

		// 唯一请求 ID，每次请求都会返回。定位问题时需要提供该次请求的 RequestId。
		RequestId *string `json:"RequestId,omitempty" name:"RequestId"`
	} `json:"Response"`
}

func (r *DescribeTagValuesResponse) ToJsonString() string {
    b, _ := json.Marshal(r)
    return string(b)
}

func (r *DescribeTagValuesResponse) FromJsonString(s string) error {
    return json.Unmarshal([]byte(s), &r)
}

type DescribeTagsRequest struct {
	*tchttp.BaseRequest

	// 标签键,与标签值同时存在或同时不存在，不存在时表示查询该用户所有标签
	TagKey *string `json:"TagKey,omitempty" name:"TagKey"`

	// 标签值,与标签键同时存在或同时不存在，不存在时表示查询该用户所有标签
	TagValue *string `json:"TagValue,omitempty" name:"TagValue"`

	// 数据偏移量，默认为 0, 必须为Limit参数的整数倍
	Offset *uint64 `json:"Offset,omitempty" name:"Offset"`

	// 每页大小，默认为 15
	Limit *uint64 `json:"Limit,omitempty" name:"Limit"`

	// 创建者用户 Uin，不传或为空只将 Uin 作为条件查询
	CreateUin *uint64 `json:"CreateUin,omitempty" name:"CreateUin"`
}

func (r *DescribeTagsRequest) ToJsonString() string {
    b, _ := json.Marshal(r)
    return string(b)
}

func (r *DescribeTagsRequest) FromJsonString(s string) error {
    return json.Unmarshal([]byte(s), &r)
}

type DescribeTagsResponse struct {
	*tchttp.BaseResponse
	Response *struct {

		// 结果总数
		TotalCount *uint64 `json:"TotalCount,omitempty" name:"TotalCount"`

		// 数据位移偏量
		Offset *uint64 `json:"Offset,omitempty" name:"Offset"`

		// 每页大小
		Limit *uint64 `json:"Limit,omitempty" name:"Limit"`

		// 标签列表
		Tags []*TagWithDelete `json:"Tags,omitempty" name:"Tags" list`

		// 唯一请求 ID，每次请求都会返回。定位问题时需要提供该次请求的 RequestId。
		RequestId *string `json:"RequestId,omitempty" name:"RequestId"`
	} `json:"Response"`
}

func (r *DescribeTagsResponse) ToJsonString() string {
    b, _ := json.Marshal(r)
    return string(b)
}

func (r *DescribeTagsResponse) FromJsonString(s string) error {
    return json.Unmarshal([]byte(s), &r)
}

type ModifyResourceTagsRequest struct {
	*tchttp.BaseRequest

	// 资源的六段式描述
	Resource *string `json:"Resource,omitempty" name:"Resource"`

	// 需要增加或修改的标签集合。如果Resource描述的资源未关联输入的标签键，则增加关联；若已关联，则将该资源关联的键对应的标签值修改为输入值。本接口中ReplaceTags和DeleteTags二者必须存在其一，且二者不能包含相同的标签键
	ReplaceTags []*Tag `json:"ReplaceTags,omitempty" name:"ReplaceTags" list`

	// 需要解关联的标签集合。本接口中ReplaceTags和DeleteTags二者必须存在其一，且二者不能包含相同的标签键
	DeleteTags []*TagKeyObject `json:"DeleteTags,omitempty" name:"DeleteTags" list`
}

func (r *ModifyResourceTagsRequest) ToJsonString() string {
    b, _ := json.Marshal(r)
    return string(b)
}

func (r *ModifyResourceTagsRequest) FromJsonString(s string) error {
    return json.Unmarshal([]byte(s), &r)
}

type ModifyResourceTagsResponse struct {
	*tchttp.BaseResponse
	Response *struct {

		// 唯一请求 ID，每次请求都会返回。定位问题时需要提供该次请求的 RequestId。
		RequestId *string `json:"RequestId,omitempty" name:"RequestId"`
	} `json:"Response"`
}

func (r *ModifyResourceTagsResponse) ToJsonString() string {
    b, _ := json.Marshal(r)
    return string(b)
}

func (r *ModifyResourceTagsResponse) FromJsonString(s string) error {
    return json.Unmarshal([]byte(s), &r)
}

type Tag struct {

	// 标签键
	TagKey *string `json:"TagKey,omitempty" name:"TagKey"`

	// 标签值
	TagValue *string `json:"TagValue,omitempty" name:"TagValue"`
}

type TagKeyObject struct {

	// 标签键
	TagKey *string `json:"TagKey,omitempty" name:"TagKey"`
}

type TagResource struct {

	// 标签键
	TagKey *string `json:"TagKey,omitempty" name:"TagKey"`

	// 标签值
	TagValue *string `json:"TagValue,omitempty" name:"TagValue"`

	// 资源ID
	ResourceId *string `json:"ResourceId,omitempty" name:"ResourceId"`

	// 标签键MD5值
	TagKeyMd5 *string `json:"TagKeyMd5,omitempty" name:"TagKeyMd5"`

	// 标签值MD5值
	TagValueMd5 *string `json:"TagValueMd5,omitempty" name:"TagValueMd5"`
}

type TagWithDelete struct {

	// 标签键
	TagKey *string `json:"TagKey,omitempty" name:"TagKey"`

	// 标签值
	TagValue *string `json:"TagValue,omitempty" name:"TagValue"`

	// 是否可以删除
	CanDelete *uint64 `json:"CanDelete,omitempty" name:"CanDelete"`
}

type UpdateResourceTagValueRequest struct {
	*tchttp.BaseRequest

	// 资源关联的标签键
	TagKey *string `json:"TagKey,omitempty" name:"TagKey"`

	// 修改后的标签值
	TagValue *string `json:"TagValue,omitempty" name:"TagValue"`

	// 资源的六段式描述
	Resource *string `json:"Resource,omitempty" name:"Resource"`
}

func (r *UpdateResourceTagValueRequest) ToJsonString() string {
    b, _ := json.Marshal(r)
    return string(b)
}

func (r *UpdateResourceTagValueRequest) FromJsonString(s string) error {
    return json.Unmarshal([]byte(s), &r)
}

type UpdateResourceTagValueResponse struct {
	*tchttp.BaseResponse
	Response *struct {

		// 唯一请求 ID，每次请求都会返回。定位问题时需要提供该次请求的 RequestId。
		RequestId *string `json:"RequestId,omitempty" name:"RequestId"`
	} `json:"Response"`
}

func (r *UpdateResourceTagValueResponse) ToJsonString() string {
    b, _ := json.Marshal(r)
    return string(b)
}

func (r *UpdateResourceTagValueResponse) FromJsonString(s string) error {
    return json.Unmarshal([]byte(s), &r)
}
