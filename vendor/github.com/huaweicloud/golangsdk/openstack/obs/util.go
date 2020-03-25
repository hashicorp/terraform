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
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var regex = regexp.MustCompile("^[\u4e00-\u9fa5]$")
var ipRegex = regexp.MustCompile("^((2[0-4]\\d|25[0-5]|[01]?\\d\\d?)\\.){3}(2[0-4]\\d|25[0-5]|[01]?\\d\\d?)$")
var v4AuthRegex = regexp.MustCompile("Credential=(.+?),SignedHeaders=(.+?),Signature=.+")
var regionRegex = regexp.MustCompile(".+/\\d+/(.+?)/.+")

func StringContains(src string, subStr string, subTranscoding string) string {
	return strings.Replace(src, subStr, subTranscoding, -1)
}
func XmlTranscoding(src string) string {
	srcTmp := StringContains(src, "&", "&amp;")
	srcTmp = StringContains(srcTmp, "<", "&lt;")
	srcTmp = StringContains(srcTmp, ">", "&gt;")
	srcTmp = StringContains(srcTmp, "'", "&apos;")
	srcTmp = StringContains(srcTmp, "\"", "&quot;")
	return srcTmp
}
func StringToInt(value string, def int) int {
	ret, err := strconv.Atoi(value)
	if err != nil {
		ret = def
	}
	return ret
}

func StringToInt64(value string, def int64) int64 {
	ret, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		ret = def
	}
	return ret
}

func IntToString(value int) string {
	return strconv.Itoa(value)
}

func Int64ToString(value int64) string {
	return strconv.FormatInt(value, 10)
}

func GetCurrentTimestamp() int64 {
	return time.Now().UnixNano() / 1000000
}

func FormatUtcNow(format string) string {
	return time.Now().UTC().Format(format)
}

func FormatUtcToRfc1123(t time.Time) string {
	ret := t.UTC().Format(time.RFC1123)
	return ret[:strings.LastIndex(ret, "UTC")] + "GMT"
}

func Md5(value []byte) []byte {
	m := md5.New()
	_, err := m.Write(value)
	if err != nil {
		doLog(LEVEL_WARN, "MD5 failed to write with reason: %v", err)
	}
	return m.Sum(nil)
}

func HmacSha1(key, value []byte) []byte {
	mac := hmac.New(sha1.New, key)
	_, err := mac.Write(value)
	if err != nil {
		doLog(LEVEL_WARN, "HmacSha1 failed to write with reason: %v", err)
	}
	return mac.Sum(nil)
}

func HmacSha256(key, value []byte) []byte {
	mac := hmac.New(sha256.New, key)
	_, err := mac.Write(value)
	if err != nil {
		doLog(LEVEL_WARN, "HmacSha256 failed to write with reason: %v", err)
	}
	return mac.Sum(nil)
}

func Base64Encode(value []byte) string {
	return base64.StdEncoding.EncodeToString(value)
}

func Base64Decode(value string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(value)
}

func HexMd5(value []byte) string {
	return Hex(Md5(value))
}

func Base64Md5(value []byte) string {
	return Base64Encode(Md5(value))
}

func Sha256Hash(value []byte) []byte {
	hash := sha256.New()
	_, err := hash.Write(value)
	if err != nil {
		doLog(LEVEL_WARN, "Sha256Hash failed to write with reason: %v", err)
	}
	return hash.Sum(nil)
}

func ParseXml(value []byte, result interface{}) error {
	if len(value) == 0 {
		return nil
	}
	return xml.Unmarshal(value, result)
}

func TransToXml(value interface{}) ([]byte, error) {
	if value == nil {
		return []byte{}, nil
	}
	return xml.Marshal(value)
}

func Hex(value []byte) string {
	return hex.EncodeToString(value)
}

func HexSha256(value []byte) string {
	return Hex(Sha256Hash(value))
}

func UrlDecode(value string) (string, error) {
	ret, err := url.QueryUnescape(value)
	if err == nil {
		return ret, nil
	}
	return "", err
}

func UrlDecodeWithoutError(value string) string {
	ret, err := UrlDecode(value)
	if err == nil {
		return ret
	}
	if isErrorLogEnabled() {
		doLog(LEVEL_ERROR, "Url decode error: %v", err)
	}
	return ""
}

func IsIP(value string) bool {
	return ipRegex.MatchString(value)
}

func UrlEncode(value string, chineseOnly bool) string {
	if chineseOnly {
		values := make([]string, 0, len(value))
		for _, val := range value {
			_value := string(val)
			if regex.MatchString(_value) {
				_value = url.QueryEscape(_value)
			}
			values = append(values, _value)
		}
		return strings.Join(values, "")
	}
	return url.QueryEscape(value)
}

func copyHeaders(m map[string][]string) (ret map[string][]string) {
	if m != nil {
		ret = make(map[string][]string, len(m))
		for key, values := range m {
			_values := make([]string, 0, len(values))
			for _, value := range values {
				_values = append(_values, value)
			}
			ret[strings.ToLower(key)] = _values
		}
	} else {
		ret = make(map[string][]string)
	}

	return
}

func parseHeaders(headers map[string][]string) (signature string, region string, signedHeaders string) {
	signature = "v2"
	if receviedAuthorization, ok := headers[strings.ToLower(HEADER_AUTH_CAMEL)]; ok && len(receviedAuthorization) > 0 {
		if strings.HasPrefix(receviedAuthorization[0], V4_HASH_PREFIX) {
			signature = "v4"
			matches := v4AuthRegex.FindStringSubmatch(receviedAuthorization[0])
			if len(matches) >= 3 {
				region = matches[1]
				regions := regionRegex.FindStringSubmatch(region)
				if len(regions) >= 2 {
					region = regions[1]
				}
				signedHeaders = matches[2]
			}

		} else if strings.HasPrefix(receviedAuthorization[0], V2_HASH_PREFIX) {
			signature = "v2"
		}
	}
	return
}

func getTemporaryKeys() []string {
	return []string{
		"Signature",
		"signature",
		"X-Amz-Signature",
		"x-amz-signature",
	}
}

func getIsObs(isTemporary bool, querys []string, headers map[string][]string) bool {
	isObs := true
	if isTemporary {
		for _, value := range querys {
			keyPrefix := strings.ToLower(value)
			if strings.HasPrefix(keyPrefix, HEADER_PREFIX) {
				isObs = false
			} else if strings.HasPrefix(value, HEADER_ACCESSS_KEY_AMZ) {
				isObs = false
			}
		}
	} else {
		for key, _ := range headers {
			keyPrefix := strings.ToLower(key)
			if strings.HasPrefix(keyPrefix, HEADER_PREFIX) {
				isObs = false
				break
			}
		}
	}
	return isObs
}

func GetV2Authorization(ak, sk, method, bucketName, objectKey, queryUrl string, headers map[string][]string) (ret map[string]string) {

	if strings.HasPrefix(queryUrl, "?") {
		queryUrl = queryUrl[1:]
	}

	method = strings.ToUpper(method)

	querys := strings.Split(queryUrl, "&")
	querysResult := make([]string, 0)
	for _, value := range querys {
		if value != "=" && len(value) != 0 {
			querysResult = append(querysResult, value)
		}
	}
	params := make(map[string]string)

	for _, value := range querysResult {
		kv := strings.Split(value, "=")
		length := len(kv)
		if length == 1 {
			key := UrlDecodeWithoutError(kv[0])
			params[key] = ""
		} else if length >= 2 {
			key := UrlDecodeWithoutError(kv[0])
			vals := make([]string, 0, length-1)
			for i := 1; i < length; i++ {
				val := UrlDecodeWithoutError(kv[i])
				vals = append(vals, val)
			}
			params[key] = strings.Join(vals, "=")
		}
	}
	headers = copyHeaders(headers)
	pathStyle := false
	if receviedHost, ok := headers[HEADER_HOST]; ok && len(receviedHost) > 0 && !strings.HasPrefix(receviedHost[0], bucketName+".") {
		pathStyle = true
	}
	conf := &config{securityProvider: &securityProvider{ak: ak, sk: sk},
		urlHolder: &urlHolder{scheme: "https", host: "dummy", port: 443},
		pathStyle: pathStyle}
	conf.signature = SignatureObs
	_, canonicalizedURL := conf.formatUrls(bucketName, objectKey, params, false)
	ret = v2Auth(ak, sk, method, canonicalizedURL, headers, true)
	v2HashPrefix := OBS_HASH_PREFIX
	ret[HEADER_AUTH_CAMEL] = fmt.Sprintf("%s %s:%s", v2HashPrefix, ak, ret["Signature"])
	return
}

func GetAuthorization(ak, sk, method, bucketName, objectKey, queryUrl string, headers map[string][]string) (ret map[string]string) {

	if strings.HasPrefix(queryUrl, "?") {
		queryUrl = queryUrl[1:]
	}

	method = strings.ToUpper(method)

	querys := strings.Split(queryUrl, "&")
	querysResult := make([]string, 0)
	for _, value := range querys {
		if value != "=" && len(value) != 0 {
			querysResult = append(querysResult, value)
		}
	}
	params := make(map[string]string)

	for _, value := range querysResult {
		kv := strings.Split(value, "=")
		length := len(kv)
		if length == 1 {
			key := UrlDecodeWithoutError(kv[0])
			params[key] = ""
		} else if length >= 2 {
			key := UrlDecodeWithoutError(kv[0])
			vals := make([]string, 0, length-1)
			for i := 1; i < length; i++ {
				val := UrlDecodeWithoutError(kv[i])
				vals = append(vals, val)
			}
			params[key] = strings.Join(vals, "=")
		}
	}
	isTemporary := false
	signature := "v2"
	temporaryKeys := getTemporaryKeys()
	for _, key := range temporaryKeys {
		if _, ok := params[key]; ok {
			isTemporary = true
			if strings.ToLower(key) == "signature" {
				signature = "v2"
			} else if strings.ToLower(key) == "x-amz-signature" {
				signature = "v4"
			}
			break
		}
	}
	isObs := getIsObs(isTemporary, querysResult, headers)
	headers = copyHeaders(headers)
	pathStyle := false
	if receviedHost, ok := headers[HEADER_HOST]; ok && len(receviedHost) > 0 && !strings.HasPrefix(receviedHost[0], bucketName+".") {
		pathStyle = true
	}
	conf := &config{securityProvider: &securityProvider{ak: ak, sk: sk},
		urlHolder: &urlHolder{scheme: "https", host: "dummy", port: 443},
		pathStyle: pathStyle}

	if isTemporary {
		return getTemporaryAuthorization(ak, sk, method, bucketName, objectKey, signature, conf, params, headers, isObs)
	} else {
		signature, region, signedHeaders := parseHeaders(headers)
		if signature == "v4" {
			conf.signature = SignatureV4
			requestUrl, canonicalizedUrl := conf.formatUrls(bucketName, objectKey, params, false)
			parsedRequestUrl, _err := url.Parse(requestUrl)
			if _err != nil {
				doLog(LEVEL_WARN, "Failed to parse requestUrl with reason: %v", _err)
				return nil
			}
			headerKeys := strings.Split(signedHeaders, ";")
			_headers := make(map[string][]string, len(headerKeys))
			for _, headerKey := range headerKeys {
				_headers[headerKey] = headers[headerKey]
			}
			ret = v4Auth(ak, sk, region, method, canonicalizedUrl, parsedRequestUrl.RawQuery, _headers)
			ret[HEADER_AUTH_CAMEL] = fmt.Sprintf("%s Credential=%s,SignedHeaders=%s,Signature=%s", V4_HASH_PREFIX, ret["Credential"], ret["SignedHeaders"], ret["Signature"])
		} else if signature == "v2" {
			if isObs {
				conf.signature = SignatureObs
			} else {
				conf.signature = SignatureV2
			}
			_, canonicalizedUrl := conf.formatUrls(bucketName, objectKey, params, false)
			ret = v2Auth(ak, sk, method, canonicalizedUrl, headers, isObs)
			v2HashPrefix := V2_HASH_PREFIX
			if isObs {
				v2HashPrefix = OBS_HASH_PREFIX
			}
			ret[HEADER_AUTH_CAMEL] = fmt.Sprintf("%s %s:%s", v2HashPrefix, ak, ret["Signature"])
		}
		return
	}

}

func getTemporaryAuthorization(ak, sk, method, bucketName, objectKey, signature string, conf *config, params map[string]string,
	headers map[string][]string, isObs bool) (ret map[string]string) {

	if signature == "v4" {
		conf.signature = SignatureV4

		longDate, ok := params[PARAM_DATE_AMZ_CAMEL]
		if !ok {
			longDate = params[HEADER_DATE_AMZ]
		}
		shortDate := longDate[:8]

		credential, ok := params[PARAM_CREDENTIAL_AMZ_CAMEL]
		if !ok {
			credential = params[strings.ToLower(PARAM_CREDENTIAL_AMZ_CAMEL)]
		}

		_credential := UrlDecodeWithoutError(credential)

		regions := regionRegex.FindStringSubmatch(_credential)
		var region string
		if len(regions) >= 2 {
			region = regions[1]
		}

		_, scope := getCredential(ak, region, shortDate)

		expires, ok := params[PARAM_EXPIRES_AMZ_CAMEL]
		if !ok {
			expires = params[strings.ToLower(PARAM_EXPIRES_AMZ_CAMEL)]
		}

		signedHeaders, ok := params[PARAM_SIGNEDHEADERS_AMZ_CAMEL]
		if !ok {
			signedHeaders = params[strings.ToLower(PARAM_SIGNEDHEADERS_AMZ_CAMEL)]
		}

		algorithm, ok := params[PARAM_ALGORITHM_AMZ_CAMEL]
		if !ok {
			algorithm = params[strings.ToLower(PARAM_ALGORITHM_AMZ_CAMEL)]
		}

		if _, ok := params[PARAM_SIGNATURE_AMZ_CAMEL]; ok {
			delete(params, PARAM_SIGNATURE_AMZ_CAMEL)
		} else if _, ok := params[strings.ToLower(PARAM_SIGNATURE_AMZ_CAMEL)]; ok {
			delete(params, strings.ToLower(PARAM_SIGNATURE_AMZ_CAMEL))
		}

		ret = make(map[string]string, 6)
		ret[PARAM_ALGORITHM_AMZ_CAMEL] = algorithm
		ret[PARAM_CREDENTIAL_AMZ_CAMEL] = credential
		ret[PARAM_DATE_AMZ_CAMEL] = longDate
		ret[PARAM_EXPIRES_AMZ_CAMEL] = expires
		ret[PARAM_SIGNEDHEADERS_AMZ_CAMEL] = signedHeaders

		requestUrl, canonicalizedUrl := conf.formatUrls(bucketName, objectKey, params, false)
		parsedRequestUrl, _err := url.Parse(requestUrl)
		if _err != nil {
			doLog(LEVEL_WARN, "Failed to parse requestUrl with reason: %v", _err)
			return nil
		}
		stringToSign := getV4StringToSign(method, canonicalizedUrl, parsedRequestUrl.RawQuery, scope, longDate, UNSIGNED_PAYLOAD, strings.Split(signedHeaders, ";"), headers)
		ret[PARAM_SIGNATURE_AMZ_CAMEL] = UrlEncode(getSignature(stringToSign, sk, region, shortDate), false)
	} else if signature == "v2" {
		if isObs {
			conf.signature = SignatureObs
		} else {
			conf.signature = SignatureV2
		}
		_, canonicalizedUrl := conf.formatUrls(bucketName, objectKey, params, false)
		expires, ok := params["Expires"]
		if !ok {
			expires = params["expires"]
		}
		headers[HEADER_DATE_CAMEL] = []string{expires}
		stringToSign := getV2StringToSign(method, canonicalizedUrl, headers, isObs)
		ret = make(map[string]string, 3)
		ret["Signature"] = UrlEncode(Base64Encode(HmacSha1([]byte(sk), []byte(stringToSign))), false)
		ret["AWSAccessKeyId"] = UrlEncode(ak, false)
		ret["Expires"] = UrlEncode(expires, false)
	}

	return
}
