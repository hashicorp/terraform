// CloudFront DistributionConfig structure helpers.
//
// These functions assist in pulling in data from Terraform resource
// configuration for the aws_cloudfront_distribution resource, as there are
// several sub-fields that require their own data type, and do not necessarily
// 1-1 translate to resource configuration.

package aws

import (
	"bytes"
	"fmt"
	"reflect"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudfront"
	"github.com/hashicorp/terraform/flatmap"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
)

// cloudFrontRoute53ZoneID defines the route 53 zone ID for CloudFront. This
// is used to set the zone_id attribute.
const cloudFrontRoute53ZoneID = "Z2FDTNDATAQYW2"

// Assemble the *cloudfront.DistributionConfig variable. Calls out to various
// expander functions to convert attributes and sub-attributes to the various
// complex structures which are necessary to properly build the
// DistributionConfig structure.
//
// Used by the aws_cloudfront_distribution Create and Update functions.
func expandDistributionConfig(d *schema.ResourceData) *cloudfront.DistributionConfig {
	distributionConfig := &cloudfront.DistributionConfig{
		CacheBehaviors:       expandCacheBehaviors(d.Get("cache_behavior").(*schema.Set)),
		CustomErrorResponses: expandCustomErrorResponses(d.Get("custom_error_response").(*schema.Set)),
		DefaultCacheBehavior: expandDefaultCacheBehavior(d.Get("default_cache_behavior").(*schema.Set).List()[0].(map[string]interface{})),
		Enabled:              aws.Bool(d.Get("enabled").(bool)),
		Origins:              expandOrigins(d.Get("origin").(*schema.Set)),
		PriceClass:           aws.String(d.Get("price_class").(string)),
	}
	// This sets CallerReference if it's still pending computation (ie: new resource)
	if v, ok := d.GetOk("caller_reference"); ok == false {
		distributionConfig.CallerReference = aws.String(time.Now().Format(time.RFC3339Nano))
	} else {
		distributionConfig.CallerReference = aws.String(v.(string))
	}
	if v, ok := d.GetOk("comment"); ok {
		distributionConfig.Comment = aws.String(v.(string))
	} else {
		distributionConfig.Comment = aws.String("")
	}
	if v, ok := d.GetOk("default_root_object"); ok {
		distributionConfig.DefaultRootObject = aws.String(v.(string))
	} else {
		distributionConfig.DefaultRootObject = aws.String("")
	}
	if v, ok := d.GetOk("logging_config"); ok {
		distributionConfig.Logging = expandLoggingConfig(v.(*schema.Set).List()[0].(map[string]interface{}))
	} else {
		distributionConfig.Logging = expandLoggingConfig(nil)
	}
	if v, ok := d.GetOk("aliases"); ok {
		distributionConfig.Aliases = expandAliases(v.(*schema.Set))
	} else {
		distributionConfig.Aliases = expandAliases(schema.NewSet(aliasesHash, []interface{}{}))
	}
	if v, ok := d.GetOk("restrictions"); ok {
		distributionConfig.Restrictions = expandRestrictions(v.(*schema.Set).List()[0].(map[string]interface{}))
	}
	if v, ok := d.GetOk("viewer_certificate"); ok {
		distributionConfig.ViewerCertificate = expandViewerCertificate(v.(*schema.Set).List()[0].(map[string]interface{}))
	}
	if v, ok := d.GetOk("web_acl_id"); ok {
		distributionConfig.WebACLId = aws.String(v.(string))
	} else {
		distributionConfig.WebACLId = aws.String("")
	}
	return distributionConfig
}

// Unpack the *cloudfront.DistributionConfig variable and set resource data.
// Calls out to flatten functions to convert the DistributionConfig
// sub-structures to their respective attributes in the
// aws_cloudfront_distribution resource.
//
// Used by the aws_cloudfront_distribution Read function.
func flattenDistributionConfig(d *schema.ResourceData, distributionConfig *cloudfront.DistributionConfig) error {
	var err error

	d.Set("enabled", distributionConfig.Enabled)
	d.Set("price_class", distributionConfig.PriceClass)
	d.Set("hosted_zone_id", cloudFrontRoute53ZoneID)

	err = d.Set("default_cache_behavior", flattenDefaultCacheBehavior(distributionConfig.DefaultCacheBehavior))
	if err != nil {
		return err
	}
	err = d.Set("viewer_certificate", flattenViewerCertificate(distributionConfig.ViewerCertificate))
	if err != nil {
		return err
	}

	if distributionConfig.CallerReference != nil {
		d.Set("caller_reference", distributionConfig.CallerReference)
	}
	if distributionConfig.Comment != nil {
		if *distributionConfig.Comment != "" {
			d.Set("comment", distributionConfig.Comment)
		}
	}
	if distributionConfig.DefaultRootObject != nil {
		d.Set("default_root_object", distributionConfig.DefaultRootObject)
	}
	if distributionConfig.WebACLId != nil {
		d.Set("web_acl_id", distributionConfig.WebACLId)
	}

	if distributionConfig.CustomErrorResponses != nil {
		err = d.Set("custom_error_response", flattenCustomErrorResponses(distributionConfig.CustomErrorResponses))
		if err != nil {
			return err
		}
	}
	if distributionConfig.CacheBehaviors != nil {
		err = d.Set("cache_behavior", flattenCacheBehaviors(distributionConfig.CacheBehaviors))
		if err != nil {
			return err
		}
	}

	if distributionConfig.Logging != nil && *distributionConfig.Logging.Enabled {
		err = d.Set("logging_config", flattenLoggingConfig(distributionConfig.Logging))
	} else {
		err = d.Set("logging_config", schema.NewSet(loggingConfigHash, []interface{}{}))
	}
	if err != nil {
		return err
	}

	if distributionConfig.Aliases != nil {
		err = d.Set("aliases", flattenAliases(distributionConfig.Aliases))
		if err != nil {
			return err
		}
	}
	if distributionConfig.Restrictions != nil {
		err = d.Set("restrictions", flattenRestrictions(distributionConfig.Restrictions))
		if err != nil {
			return err
		}
	}
	if *distributionConfig.Origins.Quantity > 0 {
		err = d.Set("origin", flattenOrigins(distributionConfig.Origins))
		if err != nil {
			return err
		}
	}

	return nil
}

func expandDefaultCacheBehavior(m map[string]interface{}) *cloudfront.DefaultCacheBehavior {
	cb := expandCacheBehavior(m)
	var dcb cloudfront.DefaultCacheBehavior

	simpleCopyStruct(cb, &dcb)
	return &dcb
}

func flattenDefaultCacheBehavior(dcb *cloudfront.DefaultCacheBehavior) *schema.Set {
	m := make(map[string]interface{})
	var cb cloudfront.CacheBehavior

	simpleCopyStruct(dcb, &cb)
	m = flattenCacheBehavior(&cb)
	return schema.NewSet(defaultCacheBehaviorHash, []interface{}{m})
}

// Assemble the hash for the aws_cloudfront_distribution default_cache_behavior
// TypeSet attribute.
func defaultCacheBehaviorHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	buf.WriteString(fmt.Sprintf("%t-", m["compress"].(bool)))
	buf.WriteString(fmt.Sprintf("%s-", m["viewer_protocol_policy"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["target_origin_id"].(string)))
	buf.WriteString(fmt.Sprintf("%d-", forwardedValuesHash(m["forwarded_values"].(*schema.Set).List()[0].(map[string]interface{}))))
	buf.WriteString(fmt.Sprintf("%d-", m["min_ttl"].(int)))
	if d, ok := m["trusted_signers"]; ok {
		for _, e := range sortInterfaceSlice(d.([]interface{})) {
			buf.WriteString(fmt.Sprintf("%s-", e.(string)))
		}
	}
	if d, ok := m["max_ttl"]; ok {
		buf.WriteString(fmt.Sprintf("%d-", d.(int)))
	}
	if d, ok := m["smooth_streaming"]; ok {
		buf.WriteString(fmt.Sprintf("%t-", d.(bool)))
	}
	if d, ok := m["default_ttl"]; ok {
		buf.WriteString(fmt.Sprintf("%d-", d.(int)))
	}
	if d, ok := m["allowed_methods"]; ok {
		for _, e := range sortInterfaceSlice(d.([]interface{})) {
			buf.WriteString(fmt.Sprintf("%s-", e.(string)))
		}
	}
	if d, ok := m["cached_methods"]; ok {
		for _, e := range sortInterfaceSlice(d.([]interface{})) {
			buf.WriteString(fmt.Sprintf("%s-", e.(string)))
		}
	}
	return hashcode.String(buf.String())
}

func expandCacheBehaviors(s *schema.Set) *cloudfront.CacheBehaviors {
	var qty int64
	var items []*cloudfront.CacheBehavior
	for _, v := range s.List() {
		items = append(items, expandCacheBehavior(v.(map[string]interface{})))
		qty++
	}
	return &cloudfront.CacheBehaviors{
		Quantity: aws.Int64(qty),
		Items:    items,
	}
}

func flattenCacheBehaviors(cbs *cloudfront.CacheBehaviors) *schema.Set {
	s := []interface{}{}
	for _, v := range cbs.Items {
		s = append(s, flattenCacheBehavior(v))
	}
	return schema.NewSet(cacheBehaviorHash, s)
}

func expandCacheBehavior(m map[string]interface{}) *cloudfront.CacheBehavior {
	cb := &cloudfront.CacheBehavior{
		Compress:             aws.Bool(m["compress"].(bool)),
		ViewerProtocolPolicy: aws.String(m["viewer_protocol_policy"].(string)),
		TargetOriginId:       aws.String(m["target_origin_id"].(string)),
		ForwardedValues:      expandForwardedValues(m["forwarded_values"].(*schema.Set).List()[0].(map[string]interface{})),
		MinTTL:               aws.Int64(int64(m["min_ttl"].(int))),
		MaxTTL:               aws.Int64(int64(m["max_ttl"].(int))),
		DefaultTTL:           aws.Int64(int64(m["default_ttl"].(int))),
	}
	if v, ok := m["trusted_signers"]; ok {
		cb.TrustedSigners = expandTrustedSigners(v.([]interface{}))
	} else {
		cb.TrustedSigners = expandTrustedSigners([]interface{}{})
	}
	if v, ok := m["smooth_streaming"]; ok {
		cb.SmoothStreaming = aws.Bool(v.(bool))
	}
	if v, ok := m["allowed_methods"]; ok {
		cb.AllowedMethods = expandAllowedMethods(v.([]interface{}))
	}
	if v, ok := m["cached_methods"]; ok {
		cb.AllowedMethods.CachedMethods = expandCachedMethods(v.([]interface{}))
	}
	if v, ok := m["path_pattern"]; ok {
		cb.PathPattern = aws.String(v.(string))
	}
	return cb
}

func flattenCacheBehavior(cb *cloudfront.CacheBehavior) map[string]interface{} {
	m := make(map[string]interface{})

	m["compress"] = *cb.Compress
	m["viewer_protocol_policy"] = *cb.ViewerProtocolPolicy
	m["target_origin_id"] = *cb.TargetOriginId
	m["forwarded_values"] = schema.NewSet(forwardedValuesHash, []interface{}{flattenForwardedValues(cb.ForwardedValues)})
	m["min_ttl"] = int(*cb.MinTTL)

	if len(cb.TrustedSigners.Items) > 0 {
		m["trusted_signers"] = flattenTrustedSigners(cb.TrustedSigners)
	}
	if cb.MaxTTL != nil {
		m["max_ttl"] = int(*cb.MaxTTL)
	}
	if cb.SmoothStreaming != nil {
		m["smooth_streaming"] = *cb.SmoothStreaming
	}
	if cb.DefaultTTL != nil {
		m["default_ttl"] = int(*cb.DefaultTTL)
	}
	if cb.AllowedMethods != nil {
		m["allowed_methods"] = flattenAllowedMethods(cb.AllowedMethods)
	}
	if cb.AllowedMethods.CachedMethods != nil {
		m["cached_methods"] = flattenCachedMethods(cb.AllowedMethods.CachedMethods)
	}
	if cb.PathPattern != nil {
		m["path_pattern"] = *cb.PathPattern
	}
	return m
}

// Assemble the hash for the aws_cloudfront_distribution cache_behavior
// TypeSet attribute.
func cacheBehaviorHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	buf.WriteString(fmt.Sprintf("%t-", m["compress"].(bool)))
	buf.WriteString(fmt.Sprintf("%s-", m["viewer_protocol_policy"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["target_origin_id"].(string)))
	buf.WriteString(fmt.Sprintf("%d-", forwardedValuesHash(m["forwarded_values"].(*schema.Set).List()[0].(map[string]interface{}))))
	buf.WriteString(fmt.Sprintf("%d-", m["min_ttl"].(int)))
	if d, ok := m["trusted_signers"]; ok {
		for _, e := range sortInterfaceSlice(d.([]interface{})) {
			buf.WriteString(fmt.Sprintf("%s-", e.(string)))
		}
	}
	if d, ok := m["max_ttl"]; ok {
		buf.WriteString(fmt.Sprintf("%d-", d.(int)))
	}
	if d, ok := m["smooth_streaming"]; ok {
		buf.WriteString(fmt.Sprintf("%t-", d.(bool)))
	}
	if d, ok := m["default_ttl"]; ok {
		buf.WriteString(fmt.Sprintf("%d-", d.(int)))
	}
	if d, ok := m["allowed_methods"]; ok {
		for _, e := range sortInterfaceSlice(d.([]interface{})) {
			buf.WriteString(fmt.Sprintf("%s-", e.(string)))
		}
	}
	if d, ok := m["cached_methods"]; ok {
		for _, e := range sortInterfaceSlice(d.([]interface{})) {
			buf.WriteString(fmt.Sprintf("%s-", e.(string)))
		}
	}
	if d, ok := m["path_pattern"]; ok {
		buf.WriteString(fmt.Sprintf("%s-", d))
	}
	return hashcode.String(buf.String())
}

func expandTrustedSigners(s []interface{}) *cloudfront.TrustedSigners {
	var ts cloudfront.TrustedSigners
	if len(s) > 0 {
		ts.Quantity = aws.Int64(int64(len(s)))
		ts.Items = expandStringList(s)
		ts.Enabled = aws.Bool(true)
	} else {
		ts.Quantity = aws.Int64(0)
		ts.Enabled = aws.Bool(false)
	}
	return &ts
}

func flattenTrustedSigners(ts *cloudfront.TrustedSigners) []interface{} {
	if ts.Items != nil {
		return flattenStringList(ts.Items)
	}
	return []interface{}{}
}

func expandForwardedValues(m map[string]interface{}) *cloudfront.ForwardedValues {
	fv := &cloudfront.ForwardedValues{
		QueryString: aws.Bool(m["query_string"].(bool)),
	}
	if v, ok := m["cookies"]; ok && v.(*schema.Set).Len() > 0 {
		fv.Cookies = expandCookiePreference(v.(*schema.Set).List()[0].(map[string]interface{}))
	}
	if v, ok := m["headers"]; ok {
		fv.Headers = expandHeaders(v.([]interface{}))
	}
	return fv
}

func flattenForwardedValues(fv *cloudfront.ForwardedValues) map[string]interface{} {
	m := make(map[string]interface{})
	m["query_string"] = *fv.QueryString
	if fv.Cookies != nil {
		m["cookies"] = schema.NewSet(cookiePreferenceHash, []interface{}{flattenCookiePreference(fv.Cookies)})
	}
	if fv.Headers != nil {
		m["headers"] = flattenHeaders(fv.Headers)
	}
	return m
}

// Assemble the hash for the aws_cloudfront_distribution forwarded_values
// TypeSet attribute.
func forwardedValuesHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	buf.WriteString(fmt.Sprintf("%t-", m["query_string"].(bool)))
	if d, ok := m["cookies"]; ok && d.(*schema.Set).Len() > 0 {
		buf.WriteString(fmt.Sprintf("%d-", cookiePreferenceHash(d.(*schema.Set).List()[0].(map[string]interface{}))))
	}
	if d, ok := m["headers"]; ok {
		for _, e := range sortInterfaceSlice(d.([]interface{})) {
			buf.WriteString(fmt.Sprintf("%s-", e.(string)))
		}
	}
	return hashcode.String(buf.String())
}

func expandHeaders(d []interface{}) *cloudfront.Headers {
	return &cloudfront.Headers{
		Quantity: aws.Int64(int64(len(d))),
		Items:    expandStringList(d),
	}
}

func flattenHeaders(h *cloudfront.Headers) []interface{} {
	if h.Items != nil {
		return flattenStringList(h.Items)
	}
	return []interface{}{}
}

func expandCookiePreference(m map[string]interface{}) *cloudfront.CookiePreference {
	cp := &cloudfront.CookiePreference{
		Forward: aws.String(m["forward"].(string)),
	}
	if v, ok := m["whitelisted_names"]; ok {
		cp.WhitelistedNames = expandCookieNames(v.([]interface{}))
	}
	return cp
}

func flattenCookiePreference(cp *cloudfront.CookiePreference) map[string]interface{} {
	m := make(map[string]interface{})
	m["forward"] = *cp.Forward
	if cp.WhitelistedNames != nil {
		m["whitelisted_names"] = flattenCookieNames(cp.WhitelistedNames)
	}
	return m
}

// Assemble the hash for the aws_cloudfront_distribution cookies
// TypeSet attribute.
func cookiePreferenceHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	buf.WriteString(fmt.Sprintf("%s-", m["forward"].(string)))
	if d, ok := m["whitelisted_names"]; ok {
		for _, e := range sortInterfaceSlice(d.([]interface{})) {
			buf.WriteString(fmt.Sprintf("%s-", e.(string)))
		}
	}
	return hashcode.String(buf.String())
}

func expandCookieNames(d []interface{}) *cloudfront.CookieNames {
	return &cloudfront.CookieNames{
		Quantity: aws.Int64(int64(len(d))),
		Items:    expandStringList(d),
	}
}

func flattenCookieNames(cn *cloudfront.CookieNames) []interface{} {
	if cn.Items != nil {
		return flattenStringList(cn.Items)
	}
	return []interface{}{}
}

func expandAllowedMethods(s []interface{}) *cloudfront.AllowedMethods {
	return &cloudfront.AllowedMethods{
		Quantity: aws.Int64(int64(len(s))),
		Items:    expandStringList(s),
	}
}

func flattenAllowedMethods(am *cloudfront.AllowedMethods) []interface{} {
	if am.Items != nil {
		return flattenStringList(am.Items)
	}
	return []interface{}{}
}

func expandCachedMethods(s []interface{}) *cloudfront.CachedMethods {
	return &cloudfront.CachedMethods{
		Quantity: aws.Int64(int64(len(s))),
		Items:    expandStringList(s),
	}
}

func flattenCachedMethods(cm *cloudfront.CachedMethods) []interface{} {
	if cm.Items != nil {
		return flattenStringList(cm.Items)
	}
	return []interface{}{}
}

func expandOrigins(s *schema.Set) *cloudfront.Origins {
	qty := 0
	items := []*cloudfront.Origin{}
	for _, v := range s.List() {
		items = append(items, expandOrigin(v.(map[string]interface{})))
		qty++
	}
	return &cloudfront.Origins{
		Quantity: aws.Int64(int64(qty)),
		Items:    items,
	}
}

func flattenOrigins(ors *cloudfront.Origins) *schema.Set {
	s := []interface{}{}
	for _, v := range ors.Items {
		s = append(s, flattenOrigin(v))
	}
	return schema.NewSet(originHash, s)
}

func expandOrigin(m map[string]interface{}) *cloudfront.Origin {
	origin := &cloudfront.Origin{
		Id:         aws.String(m["origin_id"].(string)),
		DomainName: aws.String(m["domain_name"].(string)),
	}
	if v, ok := m["custom_header"]; ok {
		origin.CustomHeaders = expandCustomHeaders(v.(*schema.Set))
	}
	if v, ok := m["custom_origin_config"]; ok {
		if s := v.(*schema.Set).List(); len(s) > 0 {
			origin.CustomOriginConfig = expandCustomOriginConfig(s[0].(map[string]interface{}))
		}
	}
	if v, ok := m["origin_path"]; ok {
		origin.OriginPath = aws.String(v.(string))
	}
	if v, ok := m["s3_origin_config"]; ok {
		if s := v.(*schema.Set).List(); len(s) > 0 {
			origin.S3OriginConfig = expandS3OriginConfig(s[0].(map[string]interface{}))
		}
	}

	// if both custom and s3 origin are missing, add an empty s3 origin
	// One or the other must be specified, but the S3 origin can be "empty"
	if origin.S3OriginConfig == nil && origin.CustomOriginConfig == nil {
		origin.S3OriginConfig = &cloudfront.S3OriginConfig{
			OriginAccessIdentity: aws.String(""),
		}
	}

	return origin
}

func flattenOrigin(or *cloudfront.Origin) map[string]interface{} {
	m := make(map[string]interface{})
	m["origin_id"] = *or.Id
	m["domain_name"] = *or.DomainName
	if or.CustomHeaders != nil {
		m["custom_header"] = flattenCustomHeaders(or.CustomHeaders)
	}
	if or.CustomOriginConfig != nil {
		m["custom_origin_config"] = schema.NewSet(customOriginConfigHash, []interface{}{flattenCustomOriginConfig(or.CustomOriginConfig)})
	}
	if or.OriginPath != nil {
		m["origin_path"] = *or.OriginPath
	}
	if or.S3OriginConfig != nil {
		if or.S3OriginConfig.OriginAccessIdentity != nil && *or.S3OriginConfig.OriginAccessIdentity != "" {
			m["s3_origin_config"] = schema.NewSet(s3OriginConfigHash, []interface{}{flattenS3OriginConfig(or.S3OriginConfig)})
		}
	}
	return m
}

// Assemble the hash for the aws_cloudfront_distribution origin
// TypeSet attribute.
func originHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	buf.WriteString(fmt.Sprintf("%s-", m["origin_id"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["domain_name"].(string)))
	if v, ok := m["custom_header"]; ok {
		buf.WriteString(fmt.Sprintf("%d-", customHeadersHash(v.(*schema.Set))))
	}
	if v, ok := m["custom_origin_config"]; ok {
		if s := v.(*schema.Set).List(); len(s) > 0 {
			buf.WriteString(fmt.Sprintf("%d-", customOriginConfigHash((s[0].(map[string]interface{})))))
		}
	}
	if v, ok := m["origin_path"]; ok {
		buf.WriteString(fmt.Sprintf("%s-", v.(string)))
	}
	if v, ok := m["s3_origin_config"]; ok {
		if s := v.(*schema.Set).List(); len(s) > 0 {
			buf.WriteString(fmt.Sprintf("%d-", s3OriginConfigHash((s[0].(map[string]interface{})))))
		}
	}
	return hashcode.String(buf.String())
}

func expandCustomHeaders(s *schema.Set) *cloudfront.CustomHeaders {
	qty := 0
	items := []*cloudfront.OriginCustomHeader{}
	for _, v := range s.List() {
		items = append(items, expandOriginCustomHeader(v.(map[string]interface{})))
		qty++
	}
	return &cloudfront.CustomHeaders{
		Quantity: aws.Int64(int64(qty)),
		Items:    items,
	}
}

func flattenCustomHeaders(chs *cloudfront.CustomHeaders) *schema.Set {
	s := []interface{}{}
	for _, v := range chs.Items {
		s = append(s, flattenOriginCustomHeader(v))
	}
	return schema.NewSet(originCustomHeaderHash, s)
}

func expandOriginCustomHeader(m map[string]interface{}) *cloudfront.OriginCustomHeader {
	return &cloudfront.OriginCustomHeader{
		HeaderName:  aws.String(m["name"].(string)),
		HeaderValue: aws.String(m["value"].(string)),
	}
}

func flattenOriginCustomHeader(och *cloudfront.OriginCustomHeader) map[string]interface{} {
	return map[string]interface{}{
		"name":  *och.HeaderName,
		"value": *och.HeaderValue,
	}
}

// Helper function used by originHash to get a composite hash for all
// aws_cloudfront_distribution custom_header attributes.
func customHeadersHash(s *schema.Set) int {
	var buf bytes.Buffer
	for _, v := range s.List() {
		buf.WriteString(fmt.Sprintf("%d-", originCustomHeaderHash(v)))
	}
	return hashcode.String(buf.String())
}

// Assemble the hash for the aws_cloudfront_distribution custom_header
// TypeSet attribute.
func originCustomHeaderHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	buf.WriteString(fmt.Sprintf("%s-", m["name"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["value"].(string)))
	return hashcode.String(buf.String())
}

func expandCustomOriginConfig(m map[string]interface{}) *cloudfront.CustomOriginConfig {
	return &cloudfront.CustomOriginConfig{
		OriginProtocolPolicy: aws.String(m["origin_protocol_policy"].(string)),
		HTTPPort:             aws.Int64(int64(m["http_port"].(int))),
		HTTPSPort:            aws.Int64(int64(m["https_port"].(int))),
		OriginSslProtocols:   expandCustomOriginConfigSSL(m["origin_ssl_protocols"].([]interface{})),
	}
}

func flattenCustomOriginConfig(cor *cloudfront.CustomOriginConfig) map[string]interface{} {
	return map[string]interface{}{
		"origin_protocol_policy": *cor.OriginProtocolPolicy,
		"http_port":              int(*cor.HTTPPort),
		"https_port":             int(*cor.HTTPSPort),
		"origin_ssl_protocols":   flattenCustomOriginConfigSSL(cor.OriginSslProtocols),
	}
}

// Assemble the hash for the aws_cloudfront_distribution custom_origin_config
// TypeSet attribute.
func customOriginConfigHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	buf.WriteString(fmt.Sprintf("%s-", m["origin_protocol_policy"].(string)))
	buf.WriteString(fmt.Sprintf("%d-", m["http_port"].(int)))
	buf.WriteString(fmt.Sprintf("%d-", m["https_port"].(int)))
	for _, v := range sortInterfaceSlice(m["origin_ssl_protocols"].([]interface{})) {
		buf.WriteString(fmt.Sprintf("%s-", v.(string)))
	}
	return hashcode.String(buf.String())
}

func expandCustomOriginConfigSSL(s []interface{}) *cloudfront.OriginSslProtocols {
	items := expandStringList(s)
	return &cloudfront.OriginSslProtocols{
		Quantity: aws.Int64(int64(len(items))),
		Items:    items,
	}
}

func flattenCustomOriginConfigSSL(osp *cloudfront.OriginSslProtocols) []interface{} {
	return flattenStringList(osp.Items)
}

func expandS3OriginConfig(m map[string]interface{}) *cloudfront.S3OriginConfig {
	return &cloudfront.S3OriginConfig{
		OriginAccessIdentity: aws.String(m["origin_access_identity"].(string)),
	}
}

func flattenS3OriginConfig(s3o *cloudfront.S3OriginConfig) map[string]interface{} {
	return map[string]interface{}{
		"origin_access_identity": *s3o.OriginAccessIdentity,
	}
}

// Assemble the hash for the aws_cloudfront_distribution s3_origin_config
// TypeSet attribute.
func s3OriginConfigHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	buf.WriteString(fmt.Sprintf("%s-", m["origin_access_identity"].(string)))
	return hashcode.String(buf.String())
}

func expandCustomErrorResponses(s *schema.Set) *cloudfront.CustomErrorResponses {
	qty := 0
	items := []*cloudfront.CustomErrorResponse{}
	for _, v := range s.List() {
		items = append(items, expandCustomErrorResponse(v.(map[string]interface{})))
		qty++
	}
	return &cloudfront.CustomErrorResponses{
		Quantity: aws.Int64(int64(qty)),
		Items:    items,
	}
}

func flattenCustomErrorResponses(ers *cloudfront.CustomErrorResponses) *schema.Set {
	s := []interface{}{}
	for _, v := range ers.Items {
		s = append(s, flattenCustomErrorResponse(v))
	}
	return schema.NewSet(customErrorResponseHash, s)
}

func expandCustomErrorResponse(m map[string]interface{}) *cloudfront.CustomErrorResponse {
	er := cloudfront.CustomErrorResponse{
		ErrorCode: aws.Int64(int64(m["error_code"].(int))),
	}
	if v, ok := m["error_caching_min_ttl"]; ok {
		er.ErrorCachingMinTTL = aws.Int64(int64(v.(int)))
	}
	if v, ok := m["response_code"]; ok && v.(int) != 0 {
		er.ResponseCode = aws.String(strconv.Itoa(v.(int)))
	} else {
		er.ResponseCode = aws.String("")
	}
	if v, ok := m["response_page_path"]; ok {
		er.ResponsePagePath = aws.String(v.(string))
	}

	return &er
}

func flattenCustomErrorResponse(er *cloudfront.CustomErrorResponse) map[string]interface{} {
	m := make(map[string]interface{})
	m["error_code"] = int(*er.ErrorCode)
	if er.ErrorCachingMinTTL != nil {
		m["error_caching_min_ttl"] = int(*er.ErrorCachingMinTTL)
	}
	if er.ResponseCode != nil {
		m["response_code"], _ = strconv.Atoi(*er.ResponseCode)
	}
	if er.ResponsePagePath != nil {
		m["response_page_path"] = *er.ResponsePagePath
	}
	return m
}

// Assemble the hash for the aws_cloudfront_distribution custom_error_response
// TypeSet attribute.
func customErrorResponseHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	buf.WriteString(fmt.Sprintf("%d-", m["error_code"].(int)))
	if v, ok := m["error_caching_min_ttl"]; ok {
		buf.WriteString(fmt.Sprintf("%d-", v.(int)))
	}
	if v, ok := m["response_code"]; ok {
		buf.WriteString(fmt.Sprintf("%d-", v.(int)))
	}
	if v, ok := m["response_page_path"]; ok {
		buf.WriteString(fmt.Sprintf("%s-", v.(string)))
	}
	return hashcode.String(buf.String())
}

func expandLoggingConfig(m map[string]interface{}) *cloudfront.LoggingConfig {
	var lc cloudfront.LoggingConfig
	if m != nil {
		lc.Prefix = aws.String(m["prefix"].(string))
		lc.Bucket = aws.String(m["bucket"].(string))
		lc.IncludeCookies = aws.Bool(m["include_cookies"].(bool))
		lc.Enabled = aws.Bool(true)
	} else {
		lc.Prefix = aws.String("")
		lc.Bucket = aws.String("")
		lc.IncludeCookies = aws.Bool(false)
		lc.Enabled = aws.Bool(false)
	}
	return &lc
}

func flattenLoggingConfig(lc *cloudfront.LoggingConfig) *schema.Set {
	m := make(map[string]interface{})
	m["prefix"] = *lc.Prefix
	m["bucket"] = *lc.Bucket
	m["include_cookies"] = *lc.IncludeCookies
	return schema.NewSet(loggingConfigHash, []interface{}{m})
}

// Assemble the hash for the aws_cloudfront_distribution logging_config
// TypeSet attribute.
func loggingConfigHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	buf.WriteString(fmt.Sprintf("%s-", m["prefix"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["bucket"].(string)))
	buf.WriteString(fmt.Sprintf("%t-", m["include_cookies"].(bool)))
	return hashcode.String(buf.String())
}

func expandAliases(as *schema.Set) *cloudfront.Aliases {
	s := as.List()
	var aliases cloudfront.Aliases
	if len(s) > 0 {
		aliases.Quantity = aws.Int64(int64(len(s)))
		aliases.Items = expandStringList(s)
	} else {
		aliases.Quantity = aws.Int64(0)
	}
	return &aliases
}

func flattenAliases(aliases *cloudfront.Aliases) *schema.Set {
	if aliases.Items != nil {
		return schema.NewSet(aliasesHash, flattenStringList(aliases.Items))
	}
	return schema.NewSet(aliasesHash, []interface{}{})
}

// Assemble the hash for the aws_cloudfront_distribution aliases
// TypeSet attribute.
func aliasesHash(v interface{}) int {
	return hashcode.String(v.(string))
}

func expandRestrictions(m map[string]interface{}) *cloudfront.Restrictions {
	return &cloudfront.Restrictions{
		GeoRestriction: expandGeoRestriction(m["geo_restriction"].(*schema.Set).List()[0].(map[string]interface{})),
	}
}

func flattenRestrictions(r *cloudfront.Restrictions) *schema.Set {
	m := make(map[string]interface{})
	s := schema.NewSet(geoRestrictionHash, []interface{}{flattenGeoRestriction(r.GeoRestriction)})
	m["geo_restriction"] = s
	return schema.NewSet(restrictionsHash, []interface{}{m})
}

// Assemble the hash for the aws_cloudfront_distribution restrictions
// TypeSet attribute.
func restrictionsHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	buf.WriteString(fmt.Sprintf("%d-", geoRestrictionHash(m["geo_restriction"].(*schema.Set).List()[0].(map[string]interface{}))))
	return hashcode.String(buf.String())
}

func expandGeoRestriction(m map[string]interface{}) *cloudfront.GeoRestriction {
	gr := cloudfront.GeoRestriction{
		RestrictionType: aws.String(m["restriction_type"].(string)),
	}
	if v, ok := m["locations"]; ok {
		gr.Quantity = aws.Int64(int64(len(v.([]interface{}))))
		gr.Items = expandStringList(v.([]interface{}))
	} else {
		gr.Quantity = aws.Int64(0)
	}
	return &gr
}

func flattenGeoRestriction(gr *cloudfront.GeoRestriction) map[string]interface{} {
	m := make(map[string]interface{})

	m["restriction_type"] = *gr.RestrictionType
	if gr.Items != nil {
		m["locations"] = flattenStringList(gr.Items)
	}
	return m
}

// Assemble the hash for the aws_cloudfront_distribution geo_restriction
// TypeSet attribute.
func geoRestrictionHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	// All keys added in alphabetical order.
	buf.WriteString(fmt.Sprintf("%s-", m["restriction_type"].(string)))
	if v, ok := m["locations"]; ok {
		for _, w := range sortInterfaceSlice(v.([]interface{})) {
			buf.WriteString(fmt.Sprintf("%s-", w.(string)))
		}
	}
	return hashcode.String(buf.String())
}

func expandViewerCertificate(m map[string]interface{}) *cloudfront.ViewerCertificate {
	var vc cloudfront.ViewerCertificate
	if v, ok := m["iam_certificate_id"]; ok && v != "" {
		vc.IAMCertificateId = aws.String(v.(string))
		vc.SSLSupportMethod = aws.String(m["ssl_support_method"].(string))
	} else if v, ok := m["acm_certificate_arn"]; ok && v != "" {
		vc.ACMCertificateArn = aws.String(v.(string))
		vc.SSLSupportMethod = aws.String(m["ssl_support_method"].(string))
	} else {
		vc.CloudFrontDefaultCertificate = aws.Bool(m["cloudfront_default_certificate"].(bool))
	}
	if v, ok := m["minimum_protocol_version"]; ok && v != "" {
		vc.MinimumProtocolVersion = aws.String(v.(string))
	}
	return &vc
}

func flattenViewerCertificate(vc *cloudfront.ViewerCertificate) *schema.Set {
	m := make(map[string]interface{})

	if vc.IAMCertificateId != nil {
		m["iam_certificate_id"] = *vc.IAMCertificateId
		m["ssl_support_method"] = *vc.SSLSupportMethod
	}
	if vc.ACMCertificateArn != nil {
		m["acm_certificate_arn"] = *vc.ACMCertificateArn
		m["ssl_support_method"] = *vc.SSLSupportMethod
	}
	if vc.CloudFrontDefaultCertificate != nil {
		m["cloudfront_default_certificate"] = *vc.CloudFrontDefaultCertificate
	}
	if vc.MinimumProtocolVersion != nil {
		m["minimum_protocol_version"] = *vc.MinimumProtocolVersion
	}
	return schema.NewSet(viewerCertificateHash, []interface{}{m})
}

// Assemble the hash for the aws_cloudfront_distribution viewer_certificate
// TypeSet attribute.
func viewerCertificateHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	if v, ok := m["iam_certificate_id"]; ok && v.(string) != "" {
		buf.WriteString(fmt.Sprintf("%s-", v.(string)))
		buf.WriteString(fmt.Sprintf("%s-", m["ssl_support_method"].(string)))
	} else if v, ok := m["acm_certificate_arn"]; ok && v.(string) != "" {
		buf.WriteString(fmt.Sprintf("%s-", v.(string)))
		buf.WriteString(fmt.Sprintf("%s-", m["ssl_support_method"].(string)))
	} else {
		buf.WriteString(fmt.Sprintf("%t-", m["cloudfront_default_certificate"].(bool)))
	}
	if v, ok := m["minimum_protocol_version"]; ok && v.(string) != "" {
		buf.WriteString(fmt.Sprintf("%s-", v.(string)))
	}
	return hashcode.String(buf.String())
}

// Do a top-level copy of struct fields from one struct to another. Used to
// copy fields between CacheBehavior and DefaultCacheBehavior structs.
func simpleCopyStruct(src, dst interface{}) {
	s := reflect.ValueOf(src).Elem()
	d := reflect.ValueOf(dst).Elem()

	for i := 0; i < s.NumField(); i++ {
		if s.Field(i).CanSet() == true {
			if s.Field(i).Interface() != nil {
				for j := 0; j < d.NumField(); j++ {
					if d.Type().Field(j).Name == s.Type().Field(i).Name {
						d.Field(j).Set(s.Field(i))
					}
				}
			}
		}
	}
}

// Convert *cloudfront.ActiveTrustedSigners to a flatmap.Map type, which ensures
// it can probably be inserted into the schema.TypeMap type used by the
// active_trusted_signers attribute.
func flattenActiveTrustedSigners(ats *cloudfront.ActiveTrustedSigners) flatmap.Map {
	m := make(map[string]interface{})
	s := []interface{}{}
	m["enabled"] = *ats.Enabled

	for _, v := range ats.Items {
		signer := make(map[string]interface{})
		signer["aws_account_number"] = *v.AwsAccountNumber
		signer["key_pair_ids"] = aws.StringValueSlice(v.KeyPairIds.Items)
		s = append(s, signer)
	}
	m["items"] = s
	return flatmap.Flatten(m)
}
