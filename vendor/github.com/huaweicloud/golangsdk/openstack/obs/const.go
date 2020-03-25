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

const (
	obs_sdk_version        = "3.19.11"
	USER_AGENT             = "obs-sdk-go/" + obs_sdk_version
	HEADER_PREFIX          = "x-amz-"
	HEADER_PREFIX_META     = "x-amz-meta-"
	HEADER_PREFIX_OBS      = "x-obs-"
	HEADER_PREFIX_META_OBS = "x-obs-meta-"
	HEADER_DATE_AMZ        = "x-amz-date"
	HEADER_DATE_OBS        = "x-obs-date"
	HEADER_STS_TOKEN_AMZ   = "x-amz-security-token"
	HEADER_STS_TOKEN_OBS   = "x-obs-security-token"
	HEADER_ACCESSS_KEY_AMZ = "AWSAccessKeyId"
	PREFIX_META            = "meta-"

	HEADER_CONTENT_SHA256_AMZ               = "x-amz-content-sha256"
	HEADER_ACL_AMZ                          = "x-amz-acl"
	HEADER_ACL_OBS                          = "x-obs-acl"
	HEADER_ACL                              = "acl"
	HEADER_LOCATION_AMZ                     = "location"
	HEADER_BUCKET_LOCATION_OBS              = "bucket-location"
	HEADER_COPY_SOURCE                      = "copy-source"
	HEADER_COPY_SOURCE_RANGE                = "copy-source-range"
	HEADER_RANGE                            = "Range"
	HEADER_STORAGE_CLASS                    = "x-default-storage-class"
	HEADER_STORAGE_CLASS_OBS                = "x-obs-storage-class"
	HEADER_VERSION_OBS                      = "version"
	HEADER_GRANT_READ_OBS                   = "grant-read"
	HEADER_GRANT_WRITE_OBS                  = "grant-write"
	HEADER_GRANT_READ_ACP_OBS               = "grant-read-acp"
	HEADER_GRANT_WRITE_ACP_OBS              = "grant-write-acp"
	HEADER_GRANT_FULL_CONTROL_OBS           = "grant-full-control"
	HEADER_GRANT_READ_DELIVERED_OBS         = "grant-read-delivered"
	HEADER_GRANT_FULL_CONTROL_DELIVERED_OBS = "grant-full-control-delivered"
	HEADER_REQUEST_ID                       = "request-id"
	HEADER_BUCKET_REGION                    = "bucket-region"
	HEADER_ACCESS_CONRTOL_ALLOW_ORIGIN      = "access-control-allow-origin"
	HEADER_ACCESS_CONRTOL_ALLOW_HEADERS     = "access-control-allow-headers"
	HEADER_ACCESS_CONRTOL_MAX_AGE           = "access-control-max-age"
	HEADER_ACCESS_CONRTOL_ALLOW_METHODS     = "access-control-allow-methods"
	HEADER_ACCESS_CONRTOL_EXPOSE_HEADERS    = "access-control-expose-headers"
	HEADER_EPID_HEADERS                     = "epid"
	HEADER_VERSION_ID                       = "version-id"
	HEADER_COPY_SOURCE_VERSION_ID           = "copy-source-version-id"
	HEADER_DELETE_MARKER                    = "delete-marker"
	HEADER_WEBSITE_REDIRECT_LOCATION        = "website-redirect-location"
	HEADER_METADATA_DIRECTIVE               = "metadata-directive"
	HEADER_EXPIRATION                       = "expiration"
	HEADER_EXPIRES_OBS                      = "x-obs-expires"
	HEADER_RESTORE                          = "restore"
	HEADER_OBJECT_TYPE                      = "object-type"
	HEADER_NEXT_APPEND_POSITION             = "next-append-position"
	HEADER_STORAGE_CLASS2                   = "storage-class"
	HEADER_CONTENT_LENGTH                   = "content-length"
	HEADER_CONTENT_TYPE                     = "content-type"
	HEADER_CONTENT_LANGUAGE                 = "content-language"
	HEADER_EXPIRES                          = "expires"
	HEADER_CACHE_CONTROL                    = "cache-control"
	HEADER_CONTENT_DISPOSITION              = "content-disposition"
	HEADER_CONTENT_ENCODING                 = "content-encoding"

	HEADER_ETAG         = "etag"
	HEADER_LASTMODIFIED = "last-modified"

	HEADER_COPY_SOURCE_IF_MATCH            = "copy-source-if-match"
	HEADER_COPY_SOURCE_IF_NONE_MATCH       = "copy-source-if-none-match"
	HEADER_COPY_SOURCE_IF_MODIFIED_SINCE   = "copy-source-if-modified-since"
	HEADER_COPY_SOURCE_IF_UNMODIFIED_SINCE = "copy-source-if-unmodified-since"

	HEADER_IF_MATCH            = "If-Match"
	HEADER_IF_NONE_MATCH       = "If-None-Match"
	HEADER_IF_MODIFIED_SINCE   = "If-Modified-Since"
	HEADER_IF_UNMODIFIED_SINCE = "If-Unmodified-Since"

	HEADER_SSEC_ENCRYPTION = "server-side-encryption-customer-algorithm"
	HEADER_SSEC_KEY        = "server-side-encryption-customer-key"
	HEADER_SSEC_KEY_MD5    = "server-side-encryption-customer-key-MD5"

	HEADER_SSEKMS_ENCRYPTION      = "server-side-encryption"
	HEADER_SSEKMS_KEY             = "server-side-encryption-aws-kms-key-id"
	HEADER_SSEKMS_ENCRYPT_KEY_OBS = "server-side-encryption-kms-key-id"

	HEADER_SSEC_COPY_SOURCE_ENCRYPTION = "copy-source-server-side-encryption-customer-algorithm"
	HEADER_SSEC_COPY_SOURCE_KEY        = "copy-source-server-side-encryption-customer-key"
	HEADER_SSEC_COPY_SOURCE_KEY_MD5    = "copy-source-server-side-encryption-customer-key-MD5"

	HEADER_SSEKMS_KEY_AMZ = "x-amz-server-side-encryption-aws-kms-key-id"

	HEADER_SSEKMS_KEY_OBS = "x-obs-server-side-encryption-kms-key-id"

	HEADER_SUCCESS_ACTION_REDIRECT = "success_action_redirect"

	HEADER_DATE_CAMEL                          = "Date"
	HEADER_HOST_CAMEL                          = "Host"
	HEADER_HOST                                = "host"
	HEADER_AUTH_CAMEL                          = "Authorization"
	HEADER_MD5_CAMEL                           = "Content-MD5"
	HEADER_LOCATION_CAMEL                      = "Location"
	HEADER_CONTENT_LENGTH_CAMEL                = "Content-Length"
	HEADER_CONTENT_TYPE_CAML                   = "Content-Type"
	HEADER_USER_AGENT_CAMEL                    = "User-Agent"
	HEADER_ORIGIN_CAMEL                        = "Origin"
	HEADER_ACCESS_CONTROL_REQUEST_HEADER_CAMEL = "Access-Control-Request-Headers"
	HEADER_CACHE_CONTROL_CAMEL                 = "Cache-Control"
	HEADER_CONTENT_DISPOSITION_CAMEL           = "Content-Disposition"
	HEADER_CONTENT_ENCODING_CAMEL              = "Content-Encoding"
	HEADER_CONTENT_LANGUAGE_CAMEL              = "Content-Language"
	HEADER_EXPIRES_CAMEL                       = "Expires"

	PARAM_VERSION_ID                   = "versionId"
	PARAM_RESPONSE_CONTENT_TYPE        = "response-content-type"
	PARAM_RESPONSE_CONTENT_LANGUAGE    = "response-content-language"
	PARAM_RESPONSE_EXPIRES             = "response-expires"
	PARAM_RESPONSE_CACHE_CONTROL       = "response-cache-control"
	PARAM_RESPONSE_CONTENT_DISPOSITION = "response-content-disposition"
	PARAM_RESPONSE_CONTENT_ENCODING    = "response-content-encoding"
	PARAM_IMAGE_PROCESS                = "x-image-process"

	PARAM_ALGORITHM_AMZ_CAMEL     = "X-Amz-Algorithm"
	PARAM_CREDENTIAL_AMZ_CAMEL    = "X-Amz-Credential"
	PARAM_DATE_AMZ_CAMEL          = "X-Amz-Date"
	PARAM_DATE_OBS_CAMEL          = "X-Obs-Date"
	PARAM_EXPIRES_AMZ_CAMEL       = "X-Amz-Expires"
	PARAM_SIGNEDHEADERS_AMZ_CAMEL = "X-Amz-SignedHeaders"
	PARAM_SIGNATURE_AMZ_CAMEL     = "X-Amz-Signature"

	DEFAULT_SIGNATURE            = SignatureV2
	DEFAULT_REGION               = "region"
	DEFAULT_CONNECT_TIMEOUT      = 60
	DEFAULT_SOCKET_TIMEOUT       = 60
	DEFAULT_HEADER_TIMEOUT       = 60
	DEFAULT_IDLE_CONN_TIMEOUT    = 30
	DEFAULT_MAX_RETRY_COUNT      = 3
	DEFAULT_MAX_REDIRECT_COUNT   = 3
	DEFAULT_MAX_CONN_PER_HOST    = 1000
	EMPTY_CONTENT_SHA256         = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	UNSIGNED_PAYLOAD             = "UNSIGNED-PAYLOAD"
	LONG_DATE_FORMAT             = "20060102T150405Z"
	SHORT_DATE_FORMAT            = "20060102"
	ISO8601_DATE_FORMAT          = "2006-01-02T15:04:05Z"
	ISO8601_MIDNIGHT_DATE_FORMAT = "2006-01-02T00:00:00Z"
	RFC1123_FORMAT               = "Mon, 02 Jan 2006 15:04:05 GMT"

	V4_SERVICE_NAME   = "s3"
	V4_SERVICE_SUFFIX = "aws4_request"

	V2_HASH_PREFIX  = "AWS"
	OBS_HASH_PREFIX = "OBS"

	V4_HASH_PREFIX = "AWS4-HMAC-SHA256"
	V4_HASH_PRE    = "AWS4"

	DEFAULT_SSE_KMS_ENCRYPTION     = "aws:kms"
	DEFAULT_SSE_KMS_ENCRYPTION_OBS = "kms"

	DEFAULT_SSE_C_ENCRYPTION = "AES256"

	HTTP_GET     = "GET"
	HTTP_POST    = "POST"
	HTTP_PUT     = "PUT"
	HTTP_DELETE  = "DELETE"
	HTTP_HEAD    = "HEAD"
	HTTP_OPTIONS = "OPTIONS"
)

type SignatureType string

const (
	SignatureV2  SignatureType = "v2"
	SignatureV4  SignatureType = "v4"
	SignatureObs SignatureType = "OBS"
)

var (
	interested_headers = []string{"content-md5", "content-type", "date"}

	allowed_response_http_header_metadata_names = map[string]bool{
		"content-type":                  true,
		"content-md5":                   true,
		"content-length":                true,
		"content-language":              true,
		"expires":                       true,
		"origin":                        true,
		"cache-control":                 true,
		"content-disposition":           true,
		"content-encoding":              true,
		"x-default-storage-class":       true,
		"location":                      true,
		"date":                          true,
		"etag":                          true,
		"host":                          true,
		"last-modified":                 true,
		"content-range":                 true,
		"x-reserved":                    true,
		"x-reserved-indicator":          true,
		"access-control-allow-origin":   true,
		"access-control-allow-headers":  true,
		"access-control-max-age":        true,
		"access-control-allow-methods":  true,
		"access-control-expose-headers": true,
		"connection":                    true,
	}

	allowed_request_http_header_metadata_names = map[string]bool{
		"content-type":                   true,
		"content-md5":                    true,
		"content-length":                 true,
		"content-language":               true,
		"expires":                        true,
		"origin":                         true,
		"cache-control":                  true,
		"content-disposition":            true,
		"content-encoding":               true,
		"access-control-request-method":  true,
		"access-control-request-headers": true,
		"x-default-storage-class":        true,
		"location":                       true,
		"date":                           true,
		"etag":                           true,
		"range":                          true,
		"host":                           true,
		"if-modified-since":              true,
		"if-unmodified-since":            true,
		"if-match":                       true,
		"if-none-match":                  true,
		"last-modified":                  true,
		"content-range":                  true,
	}

	allowed_resource_parameter_names = map[string]bool{
		"acl":                          true,
		"backtosource":                 true,
		"policy":                       true,
		"torrent":                      true,
		"logging":                      true,
		"location":                     true,
		"storageinfo":                  true,
		"quota":                        true,
		"storageclass":                 true,
		"storagepolicy":                true,
		"requestpayment":               true,
		"versions":                     true,
		"versioning":                   true,
		"versionid":                    true,
		"uploads":                      true,
		"uploadid":                     true,
		"partnumber":                   true,
		"website":                      true,
		"notification":                 true,
		"lifecycle":                    true,
		"deletebucket":                 true,
		"delete":                       true,
		"cors":                         true,
		"restore":                      true,
		"tagging":                      true,
		"append":                       true,
		"position":                     true,
		"replication":                  true,
		"response-content-type":        true,
		"response-content-language":    true,
		"response-expires":             true,
		"response-cache-control":       true,
		"response-content-disposition": true,
		"response-content-encoding":    true,
		"x-image-process":              true,
		"x-oss-process":                true,
		"x-image-save-bucket":          true,
		"x-image-save-object":          true,
	}

	mime_types = map[string]string{
		"001":     "application/x-001",
		"301":     "application/x-301",
		"323":     "text/h323",
		"7z":      "application/x-7z-compressed",
		"906":     "application/x-906",
		"907":     "drawing/907",
		"IVF":     "video/x-ivf",
		"a11":     "application/x-a11",
		"aac":     "audio/x-aac",
		"acp":     "audio/x-mei-aac",
		"ai":      "application/postscript",
		"aif":     "audio/aiff",
		"aifc":    "audio/aiff",
		"aiff":    "audio/aiff",
		"anv":     "application/x-anv",
		"apk":     "application/vnd.android.package-archive",
		"asa":     "text/asa",
		"asf":     "video/x-ms-asf",
		"asp":     "text/asp",
		"asx":     "video/x-ms-asf",
		"atom":    "application/atom+xml",
		"au":      "audio/basic",
		"avi":     "video/avi",
		"awf":     "application/vnd.adobe.workflow",
		"biz":     "text/xml",
		"bmp":     "application/x-bmp",
		"bot":     "application/x-bot",
		"bz2":     "application/x-bzip2",
		"c4t":     "application/x-c4t",
		"c90":     "application/x-c90",
		"cal":     "application/x-cals",
		"cat":     "application/vnd.ms-pki.seccat",
		"cdf":     "application/x-netcdf",
		"cdr":     "application/x-cdr",
		"cel":     "application/x-cel",
		"cer":     "application/x-x509-ca-cert",
		"cg4":     "application/x-g4",
		"cgm":     "application/x-cgm",
		"cit":     "application/x-cit",
		"class":   "java/*",
		"cml":     "text/xml",
		"cmp":     "application/x-cmp",
		"cmx":     "application/x-cmx",
		"cot":     "application/x-cot",
		"crl":     "application/pkix-crl",
		"crt":     "application/x-x509-ca-cert",
		"csi":     "application/x-csi",
		"css":     "text/css",
		"csv":     "text/csv",
		"cu":      "application/cu-seeme",
		"cut":     "application/x-cut",
		"dbf":     "application/x-dbf",
		"dbm":     "application/x-dbm",
		"dbx":     "application/x-dbx",
		"dcd":     "text/xml",
		"dcx":     "application/x-dcx",
		"deb":     "application/x-debian-package",
		"der":     "application/x-x509-ca-cert",
		"dgn":     "application/x-dgn",
		"dib":     "application/x-dib",
		"dll":     "application/x-msdownload",
		"doc":     "application/msword",
		"docx":    "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		"dot":     "application/msword",
		"drw":     "application/x-drw",
		"dtd":     "text/xml",
		"dvi":     "application/x-dvi",
		"dwf":     "application/x-dwf",
		"dwg":     "application/x-dwg",
		"dxb":     "application/x-dxb",
		"dxf":     "application/x-dxf",
		"edn":     "application/vnd.adobe.edn",
		"emf":     "application/x-emf",
		"eml":     "message/rfc822",
		"ent":     "text/xml",
		"eot":     "application/vnd.ms-fontobject",
		"epi":     "application/x-epi",
		"eps":     "application/postscript",
		"epub":    "application/epub+zip",
		"etd":     "application/x-ebx",
		"etx":     "text/x-setext",
		"exe":     "application/x-msdownload",
		"fax":     "image/fax",
		"fdf":     "application/vnd.fdf",
		"fif":     "application/fractals",
		"flac":    "audio/flac",
		"flv":     "video/x-flv",
		"fo":      "text/xml",
		"frm":     "application/x-frm",
		"g4":      "application/x-g4",
		"gbr":     "application/x-gbr",
		"gif":     "image/gif",
		"gl2":     "application/x-gl2",
		"gp4":     "application/x-gp4",
		"gz":      "application/gzip",
		"hgl":     "application/x-hgl",
		"hmr":     "application/x-hmr",
		"hpg":     "application/x-hpgl",
		"hpl":     "application/x-hpl",
		"hqx":     "application/mac-binhex40",
		"hrf":     "application/x-hrf",
		"hta":     "application/hta",
		"htc":     "text/x-component",
		"htm":     "text/html",
		"html":    "text/html",
		"htt":     "text/webviewhtml",
		"htx":     "text/html",
		"icb":     "application/x-icb",
		"ico":     "application/x-ico",
		"ics":     "text/calendar",
		"iff":     "application/x-iff",
		"ig4":     "application/x-g4",
		"igs":     "application/x-igs",
		"iii":     "application/x-iphone",
		"img":     "application/x-img",
		"ini":     "text/plain",
		"ins":     "application/x-internet-signup",
		"ipa":     "application/vnd.iphone",
		"iso":     "application/x-iso9660-image",
		"isp":     "application/x-internet-signup",
		"jar":     "application/java-archive",
		"java":    "java/*",
		"jfif":    "image/jpeg",
		"jpe":     "image/jpeg",
		"jpeg":    "image/jpeg",
		"jpg":     "image/jpeg",
		"js":      "application/x-javascript",
		"json":    "application/json",
		"jsp":     "text/html",
		"la1":     "audio/x-liquid-file",
		"lar":     "application/x-laplayer-reg",
		"latex":   "application/x-latex",
		"lavs":    "audio/x-liquid-secure",
		"lbm":     "application/x-lbm",
		"lmsff":   "audio/x-la-lms",
		"log":     "text/plain",
		"ls":      "application/x-javascript",
		"ltr":     "application/x-ltr",
		"m1v":     "video/x-mpeg",
		"m2v":     "video/x-mpeg",
		"m3u":     "audio/mpegurl",
		"m4a":     "audio/mp4",
		"m4e":     "video/mpeg4",
		"m4v":     "video/mp4",
		"mac":     "application/x-mac",
		"man":     "application/x-troff-man",
		"math":    "text/xml",
		"mdb":     "application/msaccess",
		"mfp":     "application/x-shockwave-flash",
		"mht":     "message/rfc822",
		"mhtml":   "message/rfc822",
		"mi":      "application/x-mi",
		"mid":     "audio/mid",
		"midi":    "audio/mid",
		"mil":     "application/x-mil",
		"mml":     "text/xml",
		"mnd":     "audio/x-musicnet-download",
		"mns":     "audio/x-musicnet-stream",
		"mocha":   "application/x-javascript",
		"mov":     "video/quicktime",
		"movie":   "video/x-sgi-movie",
		"mp1":     "audio/mp1",
		"mp2":     "audio/mp2",
		"mp2v":    "video/mpeg",
		"mp3":     "audio/mp3",
		"mp4":     "video/mpeg4",
		"mp4a":    "audio/mp4",
		"mp4v":    "video/mp4",
		"mpa":     "video/x-mpg",
		"mpd":     "application/vnd.ms-project",
		"mpe":     "video/x-mpeg",
		"mpeg":    "video/mpg",
		"mpg":     "video/mpg",
		"mpg4":    "video/mp4",
		"mpga":    "audio/rn-mpeg",
		"mpp":     "application/vnd.ms-project",
		"mps":     "video/x-mpeg",
		"mpt":     "application/vnd.ms-project",
		"mpv":     "video/mpg",
		"mpv2":    "video/mpeg",
		"mpw":     "application/vnd.ms-project",
		"mpx":     "application/vnd.ms-project",
		"mtx":     "text/xml",
		"mxp":     "application/x-mmxp",
		"net":     "image/pnetvue",
		"nrf":     "application/x-nrf",
		"nws":     "message/rfc822",
		"odc":     "text/x-ms-odc",
		"oga":     "audio/ogg",
		"ogg":     "audio/ogg",
		"ogv":     "video/ogg",
		"ogx":     "application/ogg",
		"out":     "application/x-out",
		"p10":     "application/pkcs10",
		"p12":     "application/x-pkcs12",
		"p7b":     "application/x-pkcs7-certificates",
		"p7c":     "application/pkcs7-mime",
		"p7m":     "application/pkcs7-mime",
		"p7r":     "application/x-pkcs7-certreqresp",
		"p7s":     "application/pkcs7-signature",
		"pbm":     "image/x-portable-bitmap",
		"pc5":     "application/x-pc5",
		"pci":     "application/x-pci",
		"pcl":     "application/x-pcl",
		"pcx":     "application/x-pcx",
		"pdf":     "application/pdf",
		"pdx":     "application/vnd.adobe.pdx",
		"pfx":     "application/x-pkcs12",
		"pgl":     "application/x-pgl",
		"pgm":     "image/x-portable-graymap",
		"pic":     "application/x-pic",
		"pko":     "application/vnd.ms-pki.pko",
		"pl":      "application/x-perl",
		"plg":     "text/html",
		"pls":     "audio/scpls",
		"plt":     "application/x-plt",
		"png":     "image/png",
		"pnm":     "image/x-portable-anymap",
		"pot":     "application/vnd.ms-powerpoint",
		"ppa":     "application/vnd.ms-powerpoint",
		"ppm":     "application/x-ppm",
		"pps":     "application/vnd.ms-powerpoint",
		"ppt":     "application/vnd.ms-powerpoint",
		"pptx":    "application/vnd.openxmlformats-officedocument.presentationml.presentation",
		"pr":      "application/x-pr",
		"prf":     "application/pics-rules",
		"prn":     "application/x-prn",
		"prt":     "application/x-prt",
		"ps":      "application/postscript",
		"ptn":     "application/x-ptn",
		"pwz":     "application/vnd.ms-powerpoint",
		"qt":      "video/quicktime",
		"r3t":     "text/vnd.rn-realtext3d",
		"ra":      "audio/vnd.rn-realaudio",
		"ram":     "audio/x-pn-realaudio",
		"rar":     "application/x-rar-compressed",
		"ras":     "application/x-ras",
		"rat":     "application/rat-file",
		"rdf":     "text/xml",
		"rec":     "application/vnd.rn-recording",
		"red":     "application/x-red",
		"rgb":     "application/x-rgb",
		"rjs":     "application/vnd.rn-realsystem-rjs",
		"rjt":     "application/vnd.rn-realsystem-rjt",
		"rlc":     "application/x-rlc",
		"rle":     "application/x-rle",
		"rm":      "application/vnd.rn-realmedia",
		"rmf":     "application/vnd.adobe.rmf",
		"rmi":     "audio/mid",
		"rmj":     "application/vnd.rn-realsystem-rmj",
		"rmm":     "audio/x-pn-realaudio",
		"rmp":     "application/vnd.rn-rn_music_package",
		"rms":     "application/vnd.rn-realmedia-secure",
		"rmvb":    "application/vnd.rn-realmedia-vbr",
		"rmx":     "application/vnd.rn-realsystem-rmx",
		"rnx":     "application/vnd.rn-realplayer",
		"rp":      "image/vnd.rn-realpix",
		"rpm":     "audio/x-pn-realaudio-plugin",
		"rsml":    "application/vnd.rn-rsml",
		"rss":     "application/rss+xml",
		"rt":      "text/vnd.rn-realtext",
		"rtf":     "application/x-rtf",
		"rv":      "video/vnd.rn-realvideo",
		"sam":     "application/x-sam",
		"sat":     "application/x-sat",
		"sdp":     "application/sdp",
		"sdw":     "application/x-sdw",
		"sgm":     "text/sgml",
		"sgml":    "text/sgml",
		"sis":     "application/vnd.symbian.install",
		"sisx":    "application/vnd.symbian.install",
		"sit":     "application/x-stuffit",
		"slb":     "application/x-slb",
		"sld":     "application/x-sld",
		"slk":     "drawing/x-slk",
		"smi":     "application/smil",
		"smil":    "application/smil",
		"smk":     "application/x-smk",
		"snd":     "audio/basic",
		"sol":     "text/plain",
		"sor":     "text/plain",
		"spc":     "application/x-pkcs7-certificates",
		"spl":     "application/futuresplash",
		"spp":     "text/xml",
		"ssm":     "application/streamingmedia",
		"sst":     "application/vnd.ms-pki.certstore",
		"stl":     "application/vnd.ms-pki.stl",
		"stm":     "text/html",
		"sty":     "application/x-sty",
		"svg":     "image/svg+xml",
		"swf":     "application/x-shockwave-flash",
		"tar":     "application/x-tar",
		"tdf":     "application/x-tdf",
		"tg4":     "application/x-tg4",
		"tga":     "application/x-tga",
		"tif":     "image/tiff",
		"tiff":    "image/tiff",
		"tld":     "text/xml",
		"top":     "drawing/x-top",
		"torrent": "application/x-bittorrent",
		"tsd":     "text/xml",
		"ttf":     "application/x-font-ttf",
		"txt":     "text/plain",
		"uin":     "application/x-icq",
		"uls":     "text/iuls",
		"vcf":     "text/x-vcard",
		"vda":     "application/x-vda",
		"vdx":     "application/vnd.visio",
		"vml":     "text/xml",
		"vpg":     "application/x-vpeg005",
		"vsd":     "application/vnd.visio",
		"vss":     "application/vnd.visio",
		"vst":     "application/x-vst",
		"vsw":     "application/vnd.visio",
		"vsx":     "application/vnd.visio",
		"vtx":     "application/vnd.visio",
		"vxml":    "text/xml",
		"wav":     "audio/wav",
		"wax":     "audio/x-ms-wax",
		"wb1":     "application/x-wb1",
		"wb2":     "application/x-wb2",
		"wb3":     "application/x-wb3",
		"wbmp":    "image/vnd.wap.wbmp",
		"webm":    "video/webm",
		"wiz":     "application/msword",
		"wk3":     "application/x-wk3",
		"wk4":     "application/x-wk4",
		"wkq":     "application/x-wkq",
		"wks":     "application/x-wks",
		"wm":      "video/x-ms-wm",
		"wma":     "audio/x-ms-wma",
		"wmd":     "application/x-ms-wmd",
		"wmf":     "application/x-wmf",
		"wml":     "text/vnd.wap.wml",
		"wmv":     "video/x-ms-wmv",
		"wmx":     "video/x-ms-wmx",
		"wmz":     "application/x-ms-wmz",
		"woff":    "application/x-font-woff",
		"wp6":     "application/x-wp6",
		"wpd":     "application/x-wpd",
		"wpg":     "application/x-wpg",
		"wpl":     "application/vnd.ms-wpl",
		"wq1":     "application/x-wq1",
		"wr1":     "application/x-wr1",
		"wri":     "application/x-wri",
		"wrk":     "application/x-wrk",
		"ws":      "application/x-ws",
		"ws2":     "application/x-ws",
		"wsc":     "text/scriptlet",
		"wsdl":    "text/xml",
		"wvx":     "video/x-ms-wvx",
		"x_b":     "application/x-x_b",
		"x_t":     "application/x-x_t",
		"xap":     "application/x-silverlight-app",
		"xbm":     "image/x-xbitmap",
		"xdp":     "application/vnd.adobe.xdp",
		"xdr":     "text/xml",
		"xfd":     "application/vnd.adobe.xfd",
		"xfdf":    "application/vnd.adobe.xfdf",
		"xhtml":   "text/html",
		"xls":     "application/vnd.ms-excel",
		"xlsx":    "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		"xlw":     "application/x-xlw",
		"xml":     "text/xml",
		"xpl":     "audio/scpls",
		"xpm":     "image/x-xpixmap",
		"xq":      "text/xml",
		"xql":     "text/xml",
		"xquery":  "text/xml",
		"xsd":     "text/xml",
		"xsl":     "text/xml",
		"xslt":    "text/xml",
		"xwd":     "application/x-xwd",
		"yaml":    "text/yaml",
		"yml":     "text/yaml",
		"zip":     "application/zip",
	}
)

type HttpMethodType string

const (
	HttpMethodGet     HttpMethodType = HTTP_GET
	HttpMethodPut     HttpMethodType = HTTP_PUT
	HttpMethodPost    HttpMethodType = HTTP_POST
	HttpMethodDelete  HttpMethodType = HTTP_DELETE
	HttpMethodHead    HttpMethodType = HTTP_HEAD
	HttpMethodOptions HttpMethodType = HTTP_OPTIONS
)

type SubResourceType string

const (
	SubResourceStoragePolicy SubResourceType = "storagePolicy"
	SubResourceStorageClass  SubResourceType = "storageClass"
	SubResourceQuota         SubResourceType = "quota"
	SubResourceStorageInfo   SubResourceType = "storageinfo"
	SubResourceLocation      SubResourceType = "location"
	SubResourceAcl           SubResourceType = "acl"
	SubResourcePolicy        SubResourceType = "policy"
	SubResourceCors          SubResourceType = "cors"
	SubResourceVersioning    SubResourceType = "versioning"
	SubResourceWebsite       SubResourceType = "website"
	SubResourceLogging       SubResourceType = "logging"
	SubResourceLifecycle     SubResourceType = "lifecycle"
	SubResourceNotification  SubResourceType = "notification"
	SubResourceTagging       SubResourceType = "tagging"
	SubResourceDelete        SubResourceType = "delete"
	SubResourceVersions      SubResourceType = "versions"
	SubResourceUploads       SubResourceType = "uploads"
	SubResourceRestore       SubResourceType = "restore"
	SubResourceMetadata      SubResourceType = "metadata"
)

type AclType string

const (
	AclPrivate                 AclType = "private"
	AclPublicRead              AclType = "public-read"
	AclPublicReadWrite         AclType = "public-read-write"
	AclAuthenticatedRead       AclType = "authenticated-read"
	AclBucketOwnerRead         AclType = "bucket-owner-read"
	AclBucketOwnerFullControl  AclType = "bucket-owner-full-control"
	AclLogDeliveryWrite        AclType = "log-delivery-write"
	AclPublicReadDelivery      AclType = "public-read-delivered"
	AclPublicReadWriteDelivery AclType = "public-read-write-delivered"
)

type StorageClassType string

const (
	StorageClassStandard StorageClassType = "STANDARD"
	StorageClassWarm     StorageClassType = "WARM"
	StorageClassCold     StorageClassType = "COLD"
)

type PermissionType string

const (
	PermissionRead        PermissionType = "READ"
	PermissionWrite       PermissionType = "WRITE"
	PermissionReadAcp     PermissionType = "READ_ACP"
	PermissionWriteAcp    PermissionType = "WRITE_ACP"
	PermissionFullControl PermissionType = "FULL_CONTROL"
)

type GranteeType string

const (
	GranteeGroup GranteeType = "Group"
	GranteeUser  GranteeType = "CanonicalUser"
)

type GroupUriType string

const (
	GroupAllUsers           GroupUriType = "AllUsers"
	GroupAuthenticatedUsers GroupUriType = "AuthenticatedUsers"
	GroupLogDelivery        GroupUriType = "LogDelivery"
)

type VersioningStatusType string

const (
	VersioningStatusEnabled   VersioningStatusType = "Enabled"
	VersioningStatusSuspended VersioningStatusType = "Suspended"
)

type ProtocolType string

const (
	ProtocolHttp  ProtocolType = "http"
	ProtocolHttps ProtocolType = "https"
)

type RuleStatusType string

const (
	RuleStatusEnabled  RuleStatusType = "Enabled"
	RuleStatusDisabled RuleStatusType = "Disabled"
)

type RestoreTierType string

const (
	RestoreTierExpedited RestoreTierType = "Expedited"
	RestoreTierStandard  RestoreTierType = "Standard"
	RestoreTierBulk      RestoreTierType = "Bulk"
)

type MetadataDirectiveType string

const (
	CopyMetadata    MetadataDirectiveType = "COPY"
	ReplaceNew      MetadataDirectiveType = "REPLACE_NEW"
	ReplaceMetadata MetadataDirectiveType = "REPLACE"
)

type EventType string

const (
	ObjectCreatedAll  EventType = "ObjectCreated:*"
	ObjectCreatedPut  EventType = "ObjectCreated:Put"
	ObjectCreatedPost EventType = "ObjectCreated:Post"

	ObjectCreatedCopy                    EventType = "ObjectCreated:Copy"
	ObjectCreatedCompleteMultipartUpload EventType = "ObjectCreated:CompleteMultipartUpload"
	ObjectRemovedAll                     EventType = "ObjectRemoved:*"
	ObjectRemovedDelete                  EventType = "ObjectRemoved:Delete"
	ObjectRemovedDeleteMarkerCreated     EventType = "ObjectRemoved:DeleteMarkerCreated"
)
