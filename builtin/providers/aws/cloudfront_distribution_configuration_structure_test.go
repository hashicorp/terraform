package aws

import (
	"reflect"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudfront"
	"github.com/hashicorp/terraform/helper/schema"
)

func defaultCacheBehaviorConf() map[string]interface{} {
	return map[string]interface{}{
		"viewer_protocol_policy":      "allow-all",
		"target_origin_id":            "myS3Origin",
		"forwarded_values":            schema.NewSet(forwardedValuesHash, []interface{}{forwardedValuesConf()}),
		"min_ttl":                     86400,
		"trusted_signers":             trustedSignersConf(),
		"lambda_function_association": lambdaFunctionAssociationsConf(),
		"max_ttl":                     365000000,
		"smooth_streaming":            false,
		"default_ttl":                 86400,
		"allowed_methods":             allowedMethodsConf(),
		"cached_methods":              cachedMethodsConf(),
		"compress":                    true,
	}
}

func cacheBehaviorConf1() map[string]interface{} {
	cb := defaultCacheBehaviorConf()
	cb["path_pattern"] = "/path1"
	return cb
}

func cacheBehaviorConf2() map[string]interface{} {
	cb := defaultCacheBehaviorConf()
	cb["path_pattern"] = "/path2"
	return cb
}

func cacheBehaviorsConf() *schema.Set {
	return schema.NewSet(cacheBehaviorHash, []interface{}{cacheBehaviorConf1(), cacheBehaviorConf2()})
}

func trustedSignersConf() []interface{} {
	return []interface{}{"1234567890EX", "1234567891EX"}
}

func lambdaFunctionAssociationsConf() *schema.Set {
	x := []interface{}{
		map[string]interface{}{
			"event_type": "viewer-request",
			"lambda_arn": "arn:aws:lambda:us-east-1:999999999:function1:alias",
		},
		map[string]interface{}{
			"event_type": "origin-response",
			"lambda_arn": "arn:aws:lambda:us-east-1:999999999:function2:alias",
		},
	}

	return schema.NewSet(lambdaFunctionAssociationHash, x)
}

func forwardedValuesConf() map[string]interface{} {
	return map[string]interface{}{
		"query_string":            true,
		"query_string_cache_keys": queryStringCacheKeysConf(),
		"cookies":                 schema.NewSet(cookiePreferenceHash, []interface{}{cookiePreferenceConf()}),
		"headers":                 headersConf(),
	}
}

func headersConf() []interface{} {
	return []interface{}{"X-Example1", "X-Example2"}
}

func queryStringCacheKeysConf() []interface{} {
	return []interface{}{"foo", "bar"}
}

func cookiePreferenceConf() map[string]interface{} {
	return map[string]interface{}{
		"forward":           "whitelist",
		"whitelisted_names": cookieNamesConf(),
	}
}

func cookieNamesConf() []interface{} {
	return []interface{}{"Example1", "Example2"}
}

func allowedMethodsConf() []interface{} {
	return []interface{}{"DELETE", "GET", "HEAD", "OPTIONS", "PATCH", "POST", "PUT"}
}

func cachedMethodsConf() []interface{} {
	return []interface{}{"GET", "HEAD", "OPTIONS"}
}

func originCustomHeadersConf() *schema.Set {
	return schema.NewSet(originCustomHeaderHash, []interface{}{originCustomHeaderConf1(), originCustomHeaderConf2()})
}

func originCustomHeaderConf1() map[string]interface{} {
	return map[string]interface{}{
		"name":  "X-Custom-Header1",
		"value": "samplevalue",
	}
}

func originCustomHeaderConf2() map[string]interface{} {
	return map[string]interface{}{
		"name":  "X-Custom-Header2",
		"value": "samplevalue",
	}
}

func customOriginConf() map[string]interface{} {
	return map[string]interface{}{
		"origin_protocol_policy":   "http-only",
		"http_port":                80,
		"https_port":               443,
		"origin_ssl_protocols":     customOriginSslProtocolsConf(),
		"origin_read_timeout":      30,
		"origin_keepalive_timeout": 5,
	}
}

func customOriginSslProtocolsConf() []interface{} {
	return []interface{}{"SSLv3", "TLSv1", "TLSv1.1", "TLSv1.2"}
}

func s3OriginConf() map[string]interface{} {
	return map[string]interface{}{
		"origin_access_identity": "origin-access-identity/cloudfront/E127EXAMPLE51Z",
	}
}

func originWithCustomConf() map[string]interface{} {
	return map[string]interface{}{
		"origin_id":            "CustomOrigin",
		"domain_name":          "www.example.com",
		"origin_path":          "/",
		"custom_origin_config": schema.NewSet(customOriginConfigHash, []interface{}{customOriginConf()}),
		"custom_header":        originCustomHeadersConf(),
	}
}
func originWithS3Conf() map[string]interface{} {
	return map[string]interface{}{
		"origin_id":        "S3Origin",
		"domain_name":      "s3.example.com",
		"origin_path":      "/",
		"s3_origin_config": schema.NewSet(s3OriginConfigHash, []interface{}{s3OriginConf()}),
		"custom_header":    originCustomHeadersConf(),
	}
}

func multiOriginConf() *schema.Set {
	return schema.NewSet(originHash, []interface{}{originWithCustomConf(), originWithS3Conf()})
}

func geoRestrictionWhitelistConf() map[string]interface{} {
	return map[string]interface{}{
		"restriction_type": "whitelist",
		"locations":        []interface{}{"CA", "GB", "US"},
	}
}

func geoRestrictionsConf() map[string]interface{} {
	return map[string]interface{}{
		"geo_restriction": schema.NewSet(geoRestrictionHash, []interface{}{geoRestrictionWhitelistConf()}),
	}
}

func geoRestrictionConfNoItems() map[string]interface{} {
	return map[string]interface{}{
		"restriction_type": "none",
	}
}

func customErrorResponsesConf() []interface{} {
	return []interface{}{
		map[string]interface{}{
			"error_code":            404,
			"error_caching_min_ttl": 30,
			"response_code":         200,
			"response_page_path":    "/error-pages/404.html",
		},
		map[string]interface{}{
			"error_code":            403,
			"error_caching_min_ttl": 15,
			"response_code":         404,
			"response_page_path":    "/error-pages/404.html",
		},
	}
}

func aliasesConf() *schema.Set {
	return schema.NewSet(aliasesHash, []interface{}{"example.com", "www.example.com"})
}

func loggingConfigConf() map[string]interface{} {
	return map[string]interface{}{
		"include_cookies": false,
		"bucket":          "mylogs.s3.amazonaws.com",
		"prefix":          "myprefix",
	}
}

func customErrorResponsesConfSet() *schema.Set {
	return schema.NewSet(customErrorResponseHash, customErrorResponsesConf())
}

func customErrorResponsesConfFirst() map[string]interface{} {
	return customErrorResponsesConf()[0].(map[string]interface{})
}

func customErrorResponseConfNoResponseCode() map[string]interface{} {
	er := customErrorResponsesConf()[0].(map[string]interface{})
	er["response_code"] = 0
	er["response_page_path"] = ""
	return er
}

func viewerCertificateConfSetCloudFrontDefault() map[string]interface{} {
	return map[string]interface{}{
		"acm_certificate_arn":            "",
		"cloudfront_default_certificate": true,
		"iam_certificate_id":             "",
		"minimum_protocol_version":       "",
		"ssl_support_method":             "",
	}
}

func viewerCertificateConfSetIAM() map[string]interface{} {
	return map[string]interface{}{
		"acm_certificate_arn":            "",
		"cloudfront_default_certificate": false,
		"iam_certificate_id":             "iamcert-01234567",
		"ssl_support_method":             "vip",
		"minimum_protocol_version":       "TLSv1",
	}
}

func viewerCertificateConfSetACM() map[string]interface{} {
	return map[string]interface{}{
		"acm_certificate_arn":            "arn:aws:acm:us-east-1:123456789012:certificate/12345678-1234-1234-1234-123456789012",
		"cloudfront_default_certificate": false,
		"iam_certificate_id":             "",
		"ssl_support_method":             "sni-only",
		"minimum_protocol_version":       "TLSv1",
	}
}

func TestCloudFrontStructure_expandDefaultCacheBehavior(t *testing.T) {
	data := defaultCacheBehaviorConf()
	dcb := expandDefaultCacheBehavior(data)
	if dcb == nil {
		t.Fatalf("ExpandDefaultCacheBehavior returned nil")
	}
	if *dcb.Compress != true {
		t.Fatalf("Expected Compress to be true, got %v", *dcb.Compress)
	}
	if *dcb.ViewerProtocolPolicy != "allow-all" {
		t.Fatalf("Expected ViewerProtocolPolicy to be allow-all, got %v", *dcb.ViewerProtocolPolicy)
	}
	if *dcb.TargetOriginId != "myS3Origin" {
		t.Fatalf("Expected TargetOriginId to be allow-all, got %v", *dcb.TargetOriginId)
	}
	if reflect.DeepEqual(dcb.ForwardedValues.Headers.Items, expandStringList(headersConf())) != true {
		t.Fatalf("Expected Items to be %v, got %v", headersConf(), dcb.ForwardedValues.Headers.Items)
	}
	if *dcb.MinTTL != 86400 {
		t.Fatalf("Expected MinTTL to be 86400, got %v", *dcb.MinTTL)
	}
	if reflect.DeepEqual(dcb.TrustedSigners.Items, expandStringList(trustedSignersConf())) != true {
		t.Fatalf("Expected TrustedSigners.Items to be %v, got %v", trustedSignersConf(), dcb.TrustedSigners.Items)
	}
	if *dcb.MaxTTL != 365000000 {
		t.Fatalf("Expected MaxTTL to be 365000000, got %v", *dcb.MaxTTL)
	}
	if *dcb.SmoothStreaming != false {
		t.Fatalf("Expected SmoothStreaming to be false, got %v", *dcb.SmoothStreaming)
	}
	if *dcb.DefaultTTL != 86400 {
		t.Fatalf("Expected DefaultTTL to be 86400, got %v", *dcb.DefaultTTL)
	}
	if *dcb.LambdaFunctionAssociations.Quantity != 2 {
		t.Fatalf("Expected LambdaFunctionAssociations to be 2, got %v", *dcb.LambdaFunctionAssociations.Quantity)
	}
	if reflect.DeepEqual(dcb.AllowedMethods.Items, expandStringList(allowedMethodsConf())) != true {
		t.Fatalf("Expected TrustedSigners.Items to be %v, got %v", allowedMethodsConf(), dcb.AllowedMethods.Items)
	}
	if reflect.DeepEqual(dcb.AllowedMethods.CachedMethods.Items, expandStringList(cachedMethodsConf())) != true {
		t.Fatalf("Expected TrustedSigners.Items to be %v, got %v", cachedMethodsConf(), dcb.AllowedMethods.CachedMethods.Items)
	}
}

func TestCloudFrontStructure_flattenDefaultCacheBehavior(t *testing.T) {
	in := defaultCacheBehaviorConf()
	dcb := expandDefaultCacheBehavior(in)
	out := flattenDefaultCacheBehavior(dcb)
	diff := schema.NewSet(defaultCacheBehaviorHash, []interface{}{in}).Difference(out)

	if len(diff.List()) > 0 {
		t.Fatalf("Expected out to be %v, got %v, diff: %v", in, out, diff)
	}
}

func TestCloudFrontStructure_expandCacheBehavior(t *testing.T) {
	data := cacheBehaviorConf1()
	cb := expandCacheBehavior(data)
	if *cb.Compress != true {
		t.Fatalf("Expected Compress to be true, got %v", *cb.Compress)
	}
	if *cb.ViewerProtocolPolicy != "allow-all" {
		t.Fatalf("Expected ViewerProtocolPolicy to be allow-all, got %v", *cb.ViewerProtocolPolicy)
	}
	if *cb.TargetOriginId != "myS3Origin" {
		t.Fatalf("Expected TargetOriginId to be myS3Origin, got %v", *cb.TargetOriginId)
	}
	if reflect.DeepEqual(cb.ForwardedValues.Headers.Items, expandStringList(headersConf())) != true {
		t.Fatalf("Expected Items to be %v, got %v", headersConf(), cb.ForwardedValues.Headers.Items)
	}
	if *cb.MinTTL != 86400 {
		t.Fatalf("Expected MinTTL to be 86400, got %v", *cb.MinTTL)
	}
	if reflect.DeepEqual(cb.TrustedSigners.Items, expandStringList(trustedSignersConf())) != true {
		t.Fatalf("Expected TrustedSigners.Items to be %v, got %v", trustedSignersConf(), cb.TrustedSigners.Items)
	}
	if *cb.MaxTTL != 365000000 {
		t.Fatalf("Expected MaxTTL to be 365000000, got %v", *cb.MaxTTL)
	}
	if *cb.SmoothStreaming != false {
		t.Fatalf("Expected SmoothStreaming to be false, got %v", *cb.SmoothStreaming)
	}
	if *cb.DefaultTTL != 86400 {
		t.Fatalf("Expected DefaultTTL to be 86400, got %v", *cb.DefaultTTL)
	}
	if *cb.LambdaFunctionAssociations.Quantity != 2 {
		t.Fatalf("Expected LambdaFunctionAssociations to be 2, got %v", *cb.LambdaFunctionAssociations.Quantity)
	}
	if reflect.DeepEqual(cb.AllowedMethods.Items, expandStringList(allowedMethodsConf())) != true {
		t.Fatalf("Expected AllowedMethods.Items to be %v, got %v", allowedMethodsConf(), cb.AllowedMethods.Items)
	}
	if reflect.DeepEqual(cb.AllowedMethods.CachedMethods.Items, expandStringList(cachedMethodsConf())) != true {
		t.Fatalf("Expected AllowedMethods.CachedMethods.Items to be %v, got %v", cachedMethodsConf(), cb.AllowedMethods.CachedMethods.Items)
	}
	if *cb.PathPattern != "/path1" {
		t.Fatalf("Expected PathPattern to be /path1, got %v", *cb.PathPattern)
	}
}

func TestCloudFrontStructure_flattenCacheBehavior(t *testing.T) {
	in := cacheBehaviorConf1()
	cb := expandCacheBehavior(in)
	out := flattenCacheBehavior(cb)
	var diff *schema.Set
	if out["compress"] != true {
		t.Fatalf("Expected out[compress] to be true, got %v", out["compress"])
	}
	if out["viewer_protocol_policy"] != "allow-all" {
		t.Fatalf("Expected out[viewer_protocol_policy] to be allow-all, got %v", out["viewer_protocol_policy"])
	}
	if out["target_origin_id"] != "myS3Origin" {
		t.Fatalf("Expected out[target_origin_id] to be myS3Origin, got %v", out["target_origin_id"])
	}

	var outSet, ok = out["lambda_function_association"].(*schema.Set)
	if !ok {
		t.Fatalf("out['lambda_function_association'] is not a slice as expected: %#v", out["lambda_function_association"])
	}

	inSet, ok := in["lambda_function_association"].(*schema.Set)
	if !ok {
		t.Fatalf("in['lambda_function_association'] is not a set as expected: %#v", in["lambda_function_association"])
	}

	if !inSet.Equal(outSet) {
		t.Fatalf("in / out sets are not equal, in: \n%#v\n\nout: \n%#v\n", inSet, outSet)
	}

	diff = out["forwarded_values"].(*schema.Set).Difference(in["forwarded_values"].(*schema.Set))
	if len(diff.List()) > 0 {
		t.Fatalf("Expected out[forwarded_values] to be %v, got %v, diff: %v", out["forwarded_values"], in["forwarded_values"], diff)
	}
	if out["min_ttl"] != int(86400) {
		t.Fatalf("Expected out[min_ttl] to be 86400 (int), got %v", out["forwarded_values"])
	}
	if reflect.DeepEqual(out["trusted_signers"], in["trusted_signers"]) != true {
		t.Fatalf("Expected out[trusted_signers] to be %v, got %v", in["trusted_signers"], out["trusted_signers"])
	}
	if out["max_ttl"] != int(365000000) {
		t.Fatalf("Expected out[max_ttl] to be 365000000 (int), got %v", out["max_ttl"])
	}
	if out["smooth_streaming"] != false {
		t.Fatalf("Expected out[smooth_streaming] to be false, got %v", out["smooth_streaming"])
	}
	if out["default_ttl"] != int(86400) {
		t.Fatalf("Expected out[default_ttl] to be 86400 (int), got %v", out["default_ttl"])
	}
	if reflect.DeepEqual(out["allowed_methods"], in["allowed_methods"]) != true {
		t.Fatalf("Expected out[allowed_methods] to be %v, got %v", in["allowed_methods"], out["allowed_methods"])
	}
	if reflect.DeepEqual(out["cached_methods"], in["cached_methods"]) != true {
		t.Fatalf("Expected out[cached_methods] to be %v, got %v", in["cached_methods"], out["cached_methods"])
	}
	if out["path_pattern"] != "/path1" {
		t.Fatalf("Expected out[path_pattern] to be /path1, got %v", out["path_pattern"])
	}
}

func TestCloudFrontStructure_expandCacheBehaviors(t *testing.T) {
	data := cacheBehaviorsConf()
	cbs := expandCacheBehaviors(data)
	if *cbs.Quantity != 2 {
		t.Fatalf("Expected Quantity to be 2, got %v", *cbs.Quantity)
	}
	if *cbs.Items[0].TargetOriginId != "myS3Origin" {
		t.Fatalf("Expected first Item's TargetOriginId to be 	myS3Origin, got %v", *cbs.Items[0].TargetOriginId)
	}
}

func TestCloudFrontStructure_flattenCacheBehaviors(t *testing.T) {
	in := cacheBehaviorsConf()
	cbs := expandCacheBehaviors(in)
	out := flattenCacheBehaviors(cbs)
	diff := in.Difference(out)

	if len(diff.List()) > 0 {
		t.Fatalf("Expected out to be %v, got %v, diff: %v", in, out, diff)
	}
}

func TestCloudFrontStructure_expandTrustedSigners(t *testing.T) {
	data := trustedSignersConf()
	ts := expandTrustedSigners(data)
	if *ts.Quantity != 2 {
		t.Fatalf("Expected Quantity to be 2, got %v", *ts.Quantity)
	}
	if *ts.Enabled != true {
		t.Fatalf("Expected Enabled to be true, got %v", *ts.Enabled)
	}
	if reflect.DeepEqual(ts.Items, expandStringList(data)) != true {
		t.Fatalf("Expected Items to be %v, got %v", data, ts.Items)
	}
}

func TestCloudFrontStructure_flattenTrustedSigners(t *testing.T) {
	in := trustedSignersConf()
	ts := expandTrustedSigners(in)
	out := flattenTrustedSigners(ts)

	if reflect.DeepEqual(in, out) != true {
		t.Fatalf("Expected out to be %v, got %v", in, out)
	}
}

func TestCloudFrontStructure_expandTrustedSigners_empty(t *testing.T) {
	data := []interface{}{}
	ts := expandTrustedSigners(data)
	if *ts.Quantity != 0 {
		t.Fatalf("Expected Quantity to be 0, got %v", *ts.Quantity)
	}
	if *ts.Enabled != false {
		t.Fatalf("Expected Enabled to be true, got %v", *ts.Enabled)
	}
	if ts.Items != nil {
		t.Fatalf("Expected Items to be nil, got %v", ts.Items)
	}
}

func TestCloudFrontStructure_expandLambdaFunctionAssociations(t *testing.T) {
	data := lambdaFunctionAssociationsConf()
	lfa := expandLambdaFunctionAssociations(data.List())
	if *lfa.Quantity != 2 {
		t.Fatalf("Expected Quantity to be 2, got %v", *lfa.Quantity)
	}
	if len(lfa.Items) != 2 {
		t.Fatalf("Expected Items to be len 2, got %v", len(lfa.Items))
	}
	if et := "viewer-request"; *lfa.Items[0].EventType != et {
		t.Fatalf("Expected first Item's EventType to be %q, got %q", et, *lfa.Items[0].EventType)
	}
	if et := "origin-response"; *lfa.Items[1].EventType != et {
		t.Fatalf("Expected second Item's EventType to be %q, got %q", et, *lfa.Items[1].EventType)
	}
}

func TestCloudFrontStructure_flattenlambdaFunctionAssociations(t *testing.T) {
	in := lambdaFunctionAssociationsConf()
	lfa := expandLambdaFunctionAssociations(in.List())
	out := flattenLambdaFunctionAssociations(lfa)

	if reflect.DeepEqual(in.List(), out.List()) != true {
		t.Fatalf("Expected out to be %v, got %v", in, out)
	}
}

func TestCloudFrontStructure_expandlambdaFunctionAssociations_empty(t *testing.T) {
	data := new(schema.Set)
	lfa := expandLambdaFunctionAssociations(data.List())
	if *lfa.Quantity != 0 {
		t.Fatalf("Expected Quantity to be 0, got %v", *lfa.Quantity)
	}
	if len(lfa.Items) != 0 {
		t.Fatalf("Expected Items to be len 0, got %v", len(lfa.Items))
	}
	if reflect.DeepEqual(lfa.Items, []*cloudfront.LambdaFunctionAssociation{}) != true {
		t.Fatalf("Expected Items to be empty, got %v", lfa.Items)
	}
}

func TestCloudFrontStructure_expandForwardedValues(t *testing.T) {
	data := forwardedValuesConf()
	fv := expandForwardedValues(data)
	if *fv.QueryString != true {
		t.Fatalf("Expected QueryString to be true, got %v", *fv.QueryString)
	}
	if reflect.DeepEqual(fv.Cookies.WhitelistedNames.Items, expandStringList(cookieNamesConf())) != true {
		t.Fatalf("Expected Cookies.WhitelistedNames.Items to be %v, got %v", cookieNamesConf(), fv.Cookies.WhitelistedNames.Items)
	}
	if reflect.DeepEqual(fv.Headers.Items, expandStringList(headersConf())) != true {
		t.Fatalf("Expected Headers.Items to be %v, got %v", headersConf(), fv.Headers.Items)
	}
}

func TestCloudFrontStructure_flattenForwardedValues(t *testing.T) {
	in := forwardedValuesConf()
	fv := expandForwardedValues(in)
	out := flattenForwardedValues(fv)

	if out["query_string"] != true {
		t.Fatalf("Expected out[query_string] to be true, got %v", out["query_string"])
	}
	if out["cookies"].(*schema.Set).Equal(in["cookies"].(*schema.Set)) != true {
		t.Fatalf("Expected out[cookies] to be %v, got %v", in["cookies"], out["cookies"])
	}
	if reflect.DeepEqual(out["headers"], in["headers"]) != true {
		t.Fatalf("Expected out[headers] to be %v, got %v", in["headers"], out["headers"])
	}
}

func TestCloudFrontStructure_expandHeaders(t *testing.T) {
	data := headersConf()
	h := expandHeaders(data)
	if *h.Quantity != 2 {
		t.Fatalf("Expected Quantity to be 2, got %v", *h.Quantity)
	}
	if reflect.DeepEqual(h.Items, expandStringList(data)) != true {
		t.Fatalf("Expected Items to be %v, got %v", data, h.Items)
	}
}

func TestCloudFrontStructure_flattenHeaders(t *testing.T) {
	in := headersConf()
	h := expandHeaders(in)
	out := flattenHeaders(h)

	if reflect.DeepEqual(in, out) != true {
		t.Fatalf("Expected out to be %v, got %v", in, out)
	}
}

func TestCloudFrontStructure_expandQueryStringCacheKeys(t *testing.T) {
	data := queryStringCacheKeysConf()
	k := expandQueryStringCacheKeys(data)
	if *k.Quantity != 2 {
		t.Fatalf("Expected Quantity to be 2, got %v", *k.Quantity)
	}
	if reflect.DeepEqual(k.Items, expandStringList(data)) != true {
		t.Fatalf("Expected Items to be %v, got %v", data, k.Items)
	}
}

func TestCloudFrontStructure_flattenQueryStringCacheKeys(t *testing.T) {
	in := queryStringCacheKeysConf()
	k := expandQueryStringCacheKeys(in)
	out := flattenQueryStringCacheKeys(k)

	if reflect.DeepEqual(in, out) != true {
		t.Fatalf("Expected out to be %v, got %v", in, out)
	}
}

func TestCloudFrontStructure_expandCookiePreference(t *testing.T) {
	data := cookiePreferenceConf()
	cp := expandCookiePreference(data)
	if *cp.Forward != "whitelist" {
		t.Fatalf("Expected Forward to be whitelist, got %v", *cp.Forward)
	}
	if reflect.DeepEqual(cp.WhitelistedNames.Items, expandStringList(cookieNamesConf())) != true {
		t.Fatalf("Expected WhitelistedNames.Items to be %v, got %v", cookieNamesConf(), cp.WhitelistedNames.Items)
	}
}

func TestCloudFrontStructure_flattenCookiePreference(t *testing.T) {
	in := cookiePreferenceConf()
	cp := expandCookiePreference(in)
	out := flattenCookiePreference(cp)

	if reflect.DeepEqual(in, out) != true {
		t.Fatalf("Expected out to be %v, got %v", in, out)
	}
}

func TestCloudFrontStructure_expandCookieNames(t *testing.T) {
	data := cookieNamesConf()
	cn := expandCookieNames(data)
	if *cn.Quantity != 2 {
		t.Fatalf("Expected Quantity to be 2, got %v", *cn.Quantity)
	}
	if reflect.DeepEqual(cn.Items, expandStringList(data)) != true {
		t.Fatalf("Expected Items to be %v, got %v", data, cn.Items)
	}
}

func TestCloudFrontStructure_flattenCookieNames(t *testing.T) {
	in := cookieNamesConf()
	cn := expandCookieNames(in)
	out := flattenCookieNames(cn)

	if reflect.DeepEqual(in, out) != true {
		t.Fatalf("Expected out to be %v, got %v", in, out)
	}
}

func TestCloudFrontStructure_expandAllowedMethods(t *testing.T) {
	data := allowedMethodsConf()
	am := expandAllowedMethods(data)
	if *am.Quantity != 7 {
		t.Fatalf("Expected Quantity to be 7, got %v", *am.Quantity)
	}
	if reflect.DeepEqual(am.Items, expandStringList(data)) != true {
		t.Fatalf("Expected Items to be %v, got %v", data, am.Items)
	}
}

func TestCloudFrontStructure_flattenAllowedMethods(t *testing.T) {
	in := allowedMethodsConf()
	am := expandAllowedMethods(in)
	out := flattenAllowedMethods(am)

	if reflect.DeepEqual(in, out) != true {
		t.Fatalf("Expected out to be %v, got %v", in, out)
	}
}

func TestCloudFrontStructure_expandCachedMethods(t *testing.T) {
	data := cachedMethodsConf()
	cm := expandCachedMethods(data)
	if *cm.Quantity != 3 {
		t.Fatalf("Expected Quantity to be 3, got %v", *cm.Quantity)
	}
	if reflect.DeepEqual(cm.Items, expandStringList(data)) != true {
		t.Fatalf("Expected Items to be %v, got %v", data, cm.Items)
	}
}

func TestCloudFrontStructure_flattenCachedMethods(t *testing.T) {
	in := cachedMethodsConf()
	cm := expandCachedMethods(in)
	out := flattenCachedMethods(cm)

	if reflect.DeepEqual(in, out) != true {
		t.Fatalf("Expected out to be %v, got %v", in, out)
	}
}

func TestCloudFrontStructure_expandOrigins(t *testing.T) {
	data := multiOriginConf()
	origins := expandOrigins(data)
	if *origins.Quantity != 2 {
		t.Fatalf("Expected Quantity to be 2, got %v", *origins.Quantity)
	}
	if *origins.Items[0].OriginPath != "/" {
		t.Fatalf("Expected first Item's OriginPath to be /, got %v", *origins.Items[0].OriginPath)
	}
}

func TestCloudFrontStructure_flattenOrigins(t *testing.T) {
	in := multiOriginConf()
	origins := expandOrigins(in)
	out := flattenOrigins(origins)
	diff := in.Difference(out)

	if len(diff.List()) > 0 {
		t.Fatalf("Expected out to be %v, got %v, diff: %v", in, out, diff)
	}
}

func TestCloudFrontStructure_expandOrigin(t *testing.T) {
	data := originWithCustomConf()
	or := expandOrigin(data)
	if *or.Id != "CustomOrigin" {
		t.Fatalf("Expected Id to be CustomOrigin, got %v", *or.Id)
	}
	if *or.DomainName != "www.example.com" {
		t.Fatalf("Expected DomainName to be www.example.com, got %v", *or.DomainName)
	}
	if *or.OriginPath != "/" {
		t.Fatalf("Expected OriginPath to be /, got %v", *or.OriginPath)
	}
	if *or.CustomOriginConfig.OriginProtocolPolicy != "http-only" {
		t.Fatalf("Expected CustomOriginConfig.OriginProtocolPolicy to be http-only, got %v", *or.CustomOriginConfig.OriginProtocolPolicy)
	}
	if *or.CustomHeaders.Items[0].HeaderValue != "samplevalue" {
		t.Fatalf("Expected CustomHeaders.Items[0].HeaderValue to be samplevalue, got %v", *or.CustomHeaders.Items[0].HeaderValue)
	}
}

func TestCloudFrontStructure_flattenOrigin(t *testing.T) {
	in := originWithCustomConf()
	or := expandOrigin(in)
	out := flattenOrigin(or)

	if out["origin_id"] != "CustomOrigin" {
		t.Fatalf("Expected out[origin_id] to be CustomOrigin, got %v", out["origin_id"])
	}
	if out["domain_name"] != "www.example.com" {
		t.Fatalf("Expected out[domain_name] to be www.example.com, got %v", out["domain_name"])
	}
	if out["origin_path"] != "/" {
		t.Fatalf("Expected out[origin_path] to be /, got %v", out["origin_path"])
	}
	if out["custom_origin_config"].(*schema.Set).Equal(in["custom_origin_config"].(*schema.Set)) != true {
		t.Fatalf("Expected out[custom_origin_config] to be %v, got %v", in["custom_origin_config"], out["custom_origin_config"])
	}
}

func TestCloudFrontStructure_expandCustomHeaders(t *testing.T) {
	in := originCustomHeadersConf()
	chs := expandCustomHeaders(in)
	if *chs.Quantity != 2 {
		t.Fatalf("Expected Quantity to be 2, got %v", *chs.Quantity)
	}
	if *chs.Items[0].HeaderValue != "samplevalue" {
		t.Fatalf("Expected first Item's HeaderValue to be samplevalue, got %v", *chs.Items[0].HeaderValue)
	}
}

func TestCloudFrontStructure_flattenCustomHeaders(t *testing.T) {
	in := originCustomHeadersConf()
	chs := expandCustomHeaders(in)
	out := flattenCustomHeaders(chs)
	diff := in.Difference(out)

	if len(diff.List()) > 0 {
		t.Fatalf("Expected out to be %v, got %v, diff: %v", in, out, diff)
	}
}

func TestCloudFrontStructure_flattenOriginCustomHeader(t *testing.T) {
	in := originCustomHeaderConf1()
	och := expandOriginCustomHeader(in)
	out := flattenOriginCustomHeader(och)

	if out["name"] != "X-Custom-Header1" {
		t.Fatalf("Expected out[name] to be X-Custom-Header1, got %v", out["name"])
	}
	if out["value"] != "samplevalue" {
		t.Fatalf("Expected out[value] to be samplevalue, got %v", out["value"])
	}
}

func TestCloudFrontStructure_expandOriginCustomHeader(t *testing.T) {
	in := originCustomHeaderConf1()
	och := expandOriginCustomHeader(in)

	if *och.HeaderName != "X-Custom-Header1" {
		t.Fatalf("Expected HeaderName to be X-Custom-Header1, got %v", *och.HeaderName)
	}
	if *och.HeaderValue != "samplevalue" {
		t.Fatalf("Expected HeaderValue to be samplevalue, got %v", *och.HeaderValue)
	}
}

func TestCloudFrontStructure_expandCustomOriginConfig(t *testing.T) {
	data := customOriginConf()
	co := expandCustomOriginConfig(data)
	if *co.OriginProtocolPolicy != "http-only" {
		t.Fatalf("Expected OriginProtocolPolicy to be http-only, got %v", *co.OriginProtocolPolicy)
	}
	if *co.HTTPPort != 80 {
		t.Fatalf("Expected HTTPPort to be 80, got %v", *co.HTTPPort)
	}
	if *co.HTTPSPort != 443 {
		t.Fatalf("Expected HTTPSPort to be 443, got %v", *co.HTTPSPort)
	}
	if *co.OriginReadTimeout != 30 {
		t.Fatalf("Expected Origin Read Timeout to be 30, got %v", *co.OriginReadTimeout)
	}
	if *co.OriginKeepaliveTimeout != 5 {
		t.Fatalf("Expected Origin Keepalive Timeout to be 5, got %v", *co.OriginKeepaliveTimeout)
	}
}

func TestCloudFrontStructure_flattenCustomOriginConfig(t *testing.T) {
	in := customOriginConf()
	co := expandCustomOriginConfig(in)
	out := flattenCustomOriginConfig(co)

	if reflect.DeepEqual(in, out) != true {
		t.Fatalf("Expected out to be %v, got %v", in, out)
	}
}

func TestCloudFrontStructure_expandCustomOriginConfigSSL(t *testing.T) {
	in := customOriginSslProtocolsConf()
	ocs := expandCustomOriginConfigSSL(in)
	if *ocs.Quantity != 4 {
		t.Fatalf("Expected Quantity to be 4, got %v", *ocs.Quantity)
	}
	if *ocs.Items[0] != "SSLv3" {
		t.Fatalf("Expected first Item to be SSLv3, got %v", *ocs.Items[0])
	}
}

func TestCloudFrontStructure_flattenCustomOriginConfigSSL(t *testing.T) {
	in := customOriginSslProtocolsConf()
	ocs := expandCustomOriginConfigSSL(in)
	out := flattenCustomOriginConfigSSL(ocs)

	if reflect.DeepEqual(in, out) != true {
		t.Fatalf("Expected out to be %v, got %v", in, out)
	}
}

func TestCloudFrontStructure_expandS3OriginConfig(t *testing.T) {
	data := s3OriginConf()
	s3o := expandS3OriginConfig(data)
	if *s3o.OriginAccessIdentity != "origin-access-identity/cloudfront/E127EXAMPLE51Z" {
		t.Fatalf("Expected OriginAccessIdentity to be origin-access-identity/cloudfront/E127EXAMPLE51Z, got %v", *s3o.OriginAccessIdentity)
	}
}

func TestCloudFrontStructure_flattenS3OriginConfig(t *testing.T) {
	in := s3OriginConf()
	s3o := expandS3OriginConfig(in)
	out := flattenS3OriginConfig(s3o)

	if reflect.DeepEqual(in, out) != true {
		t.Fatalf("Expected out to be %v, got %v", in, out)
	}
}

func TestCloudFrontStructure_expandCustomErrorResponses(t *testing.T) {
	data := customErrorResponsesConfSet()
	ers := expandCustomErrorResponses(data)
	if *ers.Quantity != 2 {
		t.Fatalf("Expected Quantity to be 2, got %v", *ers.Quantity)
	}
	if *ers.Items[0].ResponsePagePath != "/error-pages/404.html" {
		t.Fatalf("Expected ResponsePagePath in first Item to be /error-pages/404.html, got %v", *ers.Items[0].ResponsePagePath)
	}
}

func TestCloudFrontStructure_flattenCustomErrorResponses(t *testing.T) {
	in := customErrorResponsesConfSet()
	ers := expandCustomErrorResponses(in)
	out := flattenCustomErrorResponses(ers)

	if in.Equal(out) != true {
		t.Fatalf("Expected out to be %v, got %v", in, out)
	}
}

func TestCloudFrontStructure_expandCustomErrorResponse(t *testing.T) {
	data := customErrorResponsesConfFirst()
	er := expandCustomErrorResponse(data)
	if *er.ErrorCode != 404 {
		t.Fatalf("Expected ErrorCode to be 404, got %v", *er.ErrorCode)
	}
	if *er.ErrorCachingMinTTL != 30 {
		t.Fatalf("Expected ErrorCachingMinTTL to be 30, got %v", *er.ErrorCachingMinTTL)
	}
	if *er.ResponseCode != "200" {
		t.Fatalf("Expected ResponseCode to be 200 (as string), got %v", *er.ResponseCode)
	}
	if *er.ResponsePagePath != "/error-pages/404.html" {
		t.Fatalf("Expected ResponsePagePath to be /error-pages/404.html, got %v", *er.ResponsePagePath)
	}
}

func TestCloudFrontStructure_expandCustomErrorResponse_emptyResponseCode(t *testing.T) {
	data := customErrorResponseConfNoResponseCode()
	er := expandCustomErrorResponse(data)
	if *er.ResponseCode != "" {
		t.Fatalf("Expected ResponseCode to be empty string, got %v", *er.ResponseCode)
	}
	if *er.ResponsePagePath != "" {
		t.Fatalf("Expected ResponsePagePath to be empty string, got %v", *er.ResponsePagePath)
	}
}

func TestCloudFrontStructure_flattenCustomErrorResponse(t *testing.T) {
	in := customErrorResponsesConfFirst()
	er := expandCustomErrorResponse(in)
	out := flattenCustomErrorResponse(er)

	if reflect.DeepEqual(in, out) != true {
		t.Fatalf("Expected out to be %v, got %v", in, out)
	}
}

func TestCloudFrontStructure_expandLoggingConfig(t *testing.T) {
	data := loggingConfigConf()

	lc := expandLoggingConfig(data)
	if *lc.Enabled != true {
		t.Fatalf("Expected Enabled to be true, got %v", *lc.Enabled)
	}
	if *lc.Prefix != "myprefix" {
		t.Fatalf("Expected Prefix to be myprefix, got %v", *lc.Prefix)
	}
	if *lc.Bucket != "mylogs.s3.amazonaws.com" {
		t.Fatalf("Expected Bucket to be mylogs.s3.amazonaws.com, got %v", *lc.Bucket)
	}
	if *lc.IncludeCookies != false {
		t.Fatalf("Expected IncludeCookies to be false, got %v", *lc.IncludeCookies)
	}
}

func TestCloudFrontStructure_expandLoggingConfig_nilValue(t *testing.T) {
	lc := expandLoggingConfig(nil)
	if *lc.Enabled != false {
		t.Fatalf("Expected Enabled to be false, got %v", *lc.Enabled)
	}
	if *lc.Prefix != "" {
		t.Fatalf("Expected Prefix to be blank, got %v", *lc.Prefix)
	}
	if *lc.Bucket != "" {
		t.Fatalf("Expected Bucket to be blank, got %v", *lc.Bucket)
	}
	if *lc.IncludeCookies != false {
		t.Fatalf("Expected IncludeCookies to be false, got %v", *lc.IncludeCookies)
	}
}

func TestCloudFrontStructure_flattenLoggingConfig(t *testing.T) {
	in := loggingConfigConf()
	lc := expandLoggingConfig(in)
	out := flattenLoggingConfig(lc)
	diff := schema.NewSet(loggingConfigHash, []interface{}{in}).Difference(out)

	if len(diff.List()) > 0 {
		t.Fatalf("Expected out to be %v, got %v, diff: %v", in, out, diff)
	}
}

func TestCloudFrontStructure_expandAliases(t *testing.T) {
	data := aliasesConf()
	a := expandAliases(data)
	if *a.Quantity != 2 {
		t.Fatalf("Expected Quantity to be 2, got %v", *a.Quantity)
	}
	if reflect.DeepEqual(a.Items, expandStringList(data.List())) != true {
		t.Fatalf("Expected Items to be [example.com www.example.com], got %v", a.Items)
	}
}

func TestCloudFrontStructure_flattenAliases(t *testing.T) {
	in := aliasesConf()
	a := expandAliases(in)
	out := flattenAliases(a)
	diff := in.Difference(out)

	if len(diff.List()) > 0 {
		t.Fatalf("Expected out to be %v, got %v, diff: %v", in, out, diff)
	}
}

func TestCloudFrontStructure_expandRestrictions(t *testing.T) {
	data := geoRestrictionsConf()
	r := expandRestrictions(data)
	if *r.GeoRestriction.RestrictionType != "whitelist" {
		t.Fatalf("Expected GeoRestriction.RestrictionType to be whitelist, got %v", *r.GeoRestriction.RestrictionType)
	}
}

func TestCloudFrontStructure_flattenRestrictions(t *testing.T) {
	in := geoRestrictionsConf()
	r := expandRestrictions(in)
	out := flattenRestrictions(r)
	diff := schema.NewSet(restrictionsHash, []interface{}{in}).Difference(out)

	if len(diff.List()) > 0 {
		t.Fatalf("Expected out to be %v, got %v, diff: %v", in, out, diff)
	}
}

func TestCloudFrontStructure_expandGeoRestriction_whitelist(t *testing.T) {
	data := geoRestrictionWhitelistConf()
	gr := expandGeoRestriction(data)
	if *gr.RestrictionType != "whitelist" {
		t.Fatalf("Expected RestrictionType to be whitelist, got %v", *gr.RestrictionType)
	}
	if *gr.Quantity != 3 {
		t.Fatalf("Expected Quantity to be 3, got %v", *gr.Quantity)
	}
	if reflect.DeepEqual(gr.Items, aws.StringSlice([]string{"CA", "GB", "US"})) != true {
		t.Fatalf("Expected Items be [CA, GB, US], got %v", gr.Items)
	}
}

func TestCloudFrontStructure_flattenGeoRestriction_whitelist(t *testing.T) {
	in := geoRestrictionWhitelistConf()
	gr := expandGeoRestriction(in)
	out := flattenGeoRestriction(gr)

	if reflect.DeepEqual(in, out) != true {
		t.Fatalf("Expected out to be %v, got %v", in, out)
	}
}

func TestCloudFrontStructure_expandGeoRestriction_no_items(t *testing.T) {
	data := geoRestrictionConfNoItems()
	gr := expandGeoRestriction(data)
	if *gr.RestrictionType != "none" {
		t.Fatalf("Expected RestrictionType to be none, got %v", *gr.RestrictionType)
	}
	if *gr.Quantity != 0 {
		t.Fatalf("Expected Quantity to be 0, got %v", *gr.Quantity)
	}
	if gr.Items != nil {
		t.Fatalf("Expected Items to not be set, got %v", gr.Items)
	}
}

func TestCloudFrontStructure_flattenGeoRestriction_no_items(t *testing.T) {
	in := geoRestrictionConfNoItems()
	gr := expandGeoRestriction(in)
	out := flattenGeoRestriction(gr)

	if reflect.DeepEqual(in, out) != true {
		t.Fatalf("Expected out to be %v, got %v", in, out)
	}
}

func TestCloudFrontStructure_expandViewerCertificate_cloudfront_default_certificate(t *testing.T) {
	data := viewerCertificateConfSetCloudFrontDefault()
	vc := expandViewerCertificate(data)
	if vc.ACMCertificateArn != nil {
		t.Fatalf("Expected ACMCertificateArn to be unset, got %v", *vc.ACMCertificateArn)
	}
	if *vc.CloudFrontDefaultCertificate != true {
		t.Fatalf("Expected CloudFrontDefaultCertificate to be true, got %v", *vc.CloudFrontDefaultCertificate)
	}
	if vc.IAMCertificateId != nil {
		t.Fatalf("Expected IAMCertificateId to not be set, got %v", *vc.IAMCertificateId)
	}
	if vc.SSLSupportMethod != nil {
		t.Fatalf("Expected IAMCertificateId to not be set, got %v", *vc.SSLSupportMethod)
	}
	if vc.MinimumProtocolVersion != nil {
		t.Fatalf("Expected IAMCertificateId to not be set, got %v", *vc.MinimumProtocolVersion)
	}
}

func TestCloudFrontStructure_flattenViewerCertificate_cloudfront_default_certificate(t *testing.T) {
	in := viewerCertificateConfSetCloudFrontDefault()
	vc := expandViewerCertificate(in)
	out := flattenViewerCertificate(vc)
	diff := schema.NewSet(viewerCertificateHash, []interface{}{in}).Difference(out)

	if len(diff.List()) > 0 {
		t.Fatalf("Expected out to be %v, got %v, diff: %v", in, out, diff)
	}
}

func TestCloudFrontStructure_expandViewerCertificate_iam_certificate_id(t *testing.T) {
	data := viewerCertificateConfSetIAM()
	vc := expandViewerCertificate(data)
	if vc.ACMCertificateArn != nil {
		t.Fatalf("Expected ACMCertificateArn to be unset, got %v", *vc.ACMCertificateArn)
	}
	if vc.CloudFrontDefaultCertificate != nil {
		t.Fatalf("Expected CloudFrontDefaultCertificate to be unset, got %v", *vc.CloudFrontDefaultCertificate)
	}
	if *vc.IAMCertificateId != "iamcert-01234567" {
		t.Fatalf("Expected IAMCertificateId to be iamcert-01234567, got %v", *vc.IAMCertificateId)
	}
	if *vc.SSLSupportMethod != "vip" {
		t.Fatalf("Expected IAMCertificateId to be vip, got %v", *vc.SSLSupportMethod)
	}
	if *vc.MinimumProtocolVersion != "TLSv1" {
		t.Fatalf("Expected IAMCertificateId to be TLSv1, got %v", *vc.MinimumProtocolVersion)
	}
}

func TestCloudFrontStructure_expandViewerCertificate_acm_certificate_arn(t *testing.T) {
	data := viewerCertificateConfSetACM()
	vc := expandViewerCertificate(data)
	if *vc.ACMCertificateArn != "arn:aws:acm:us-east-1:123456789012:certificate/12345678-1234-1234-1234-123456789012" {
		t.Fatalf("Expected ACMCertificateArn to be arn:aws:acm:us-east-1:123456789012:certificate/12345678-1234-1234-1234-123456789012, got %v", *vc.ACMCertificateArn)
	}
	if vc.CloudFrontDefaultCertificate != nil {
		t.Fatalf("Expected CloudFrontDefaultCertificate to be unset, got %v", *vc.CloudFrontDefaultCertificate)
	}
	if vc.IAMCertificateId != nil {
		t.Fatalf("Expected IAMCertificateId to be unset, got %v", *vc.IAMCertificateId)
	}
	if *vc.SSLSupportMethod != "sni-only" {
		t.Fatalf("Expected IAMCertificateId to be sni-only, got %v", *vc.SSLSupportMethod)
	}
	if *vc.MinimumProtocolVersion != "TLSv1" {
		t.Fatalf("Expected IAMCertificateId to be TLSv1, got %v", *vc.MinimumProtocolVersion)
	}
}

func TestCloudFrontStructure_falttenViewerCertificate_iam_certificate_id(t *testing.T) {
	in := viewerCertificateConfSetIAM()
	vc := expandViewerCertificate(in)
	out := flattenViewerCertificate(vc)
	diff := schema.NewSet(viewerCertificateHash, []interface{}{in}).Difference(out)

	if len(diff.List()) > 0 {
		t.Fatalf("Expected out to be %v, got %v, diff: %v", in, out, diff)
	}
}

func TestCloudFrontStructure_falttenViewerCertificate_acm_certificate_arn(t *testing.T) {
	in := viewerCertificateConfSetACM()
	vc := expandViewerCertificate(in)
	out := flattenViewerCertificate(vc)
	diff := schema.NewSet(viewerCertificateHash, []interface{}{in}).Difference(out)

	if len(diff.List()) > 0 {
		t.Fatalf("Expected out to be %v, got %v, diff: %v", in, out, diff)
	}
}

func TestCloudFrontStructure_viewerCertificateHash_IAM(t *testing.T) {
	in := viewerCertificateConfSetIAM()
	out := viewerCertificateHash(in)
	expected := 1157261784

	if expected != out {
		t.Fatalf("Expected %v, got %v", expected, out)
	}
}

func TestCloudFrontStructure_viewerCertificateHash_ACM(t *testing.T) {
	in := viewerCertificateConfSetACM()
	out := viewerCertificateHash(in)
	expected := 2883600425

	if expected != out {
		t.Fatalf("Expected %v, got %v", expected, out)
	}
}

func TestCloudFrontStructure_viewerCertificateHash_default(t *testing.T) {
	in := viewerCertificateConfSetCloudFrontDefault()
	out := viewerCertificateHash(in)
	expected := 69840937

	if expected != out {
		t.Fatalf("Expected %v, got %v", expected, out)
	}
}
