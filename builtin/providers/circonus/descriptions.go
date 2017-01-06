package circonus

var (
	checkDescription       map[string]string
	checkMetricDescription map[string]string
	collectorDescription   map[string]string
	contactDescription     map[string]string
)

func init() {
	checkDescription = map[string]string{
		checkActiveAttr:                "If the check is activate or disabled",
		checkCollectorAttr:             "The collector(s) that are responsible for gathering the metrics",
		checkConfigAuthMethodAttr:      "The HTTP Authentication method",
		checkConfigAuthPasswordAttr:    "The HTTP Authentication user password",
		checkConfigAuthUserAttr:        "The HTTP Authentication user name",
		checkConfigCAChainAttr:         "A path to a file containing all the certificate authorities that should be loaded to validate the remote certificate (for SSL checks)",
		checkConfigCertificateFileAttr: "A path to a file containing the client certificate that will be presented to the remote server (for SSL checks)",
		checkConfigCiphersAttr:         "A list of ciphers to be used in the SSL protocol (for SSL checks)",
		checkConfigHTTPHeadersAttr:     "Map of HTTP Headers to send along with HTTP Requests",
		checkConfigHTTPVersionAttr:     "Sets the HTTP version for the check to use",
		checkConfigKeyFileAttr:         "A path to a file containing key to be used in conjunction with the cilent certificate (for SSL checks)",
		checkConfigMethodAttr:          "The HTTP method to use",
		checkConfigPayloadAttr:         "The information transferred as the payload of an HTTP request",
		checkConfigPortAttr:            "Specifies the port on which the management interface can be reached",
		checkConfigReadLimitAttr:       "Sets an approximate limit on the data read (0 means no limit)",
		checkConfigRedirectsAttr:       `The maximum number of Location header redirects to follow (0 means no limit)`,
		checkConfigURLAttr:             "The URL including schema and hostname (as you would type into a browser's location bar)",
		checkMetricLimitAttr:           `Setting a metric_limit will enable all (-1), disable (0), or allow up to the specified limit of metrics for this check ("N+", where N is a positive integer)`,
		checkMetricNamesAttr:           "A list of metric names found within this check",
		checkNameAttr:                  "The name of the check bundle that will be displayed in the web interface",
		checkNotesAttr:                 "Notes about this check bundle",
		checkPeriodAttr:                "The period between each time the check is made",
		checkTagsAttr:                  "A list of tags assigned to the check",
		checkTargetAttr:                "The target of the check (e.g. hostname, URL, IP, etc)",
		checkTimeoutAttr:               "The length of time in seconds (and fractions of a second) before the check will timeout if no response is returned to the collector",
		checkTypeAttr:                  "The check type",
	}

	checkMetricDescription = map[string]string{
		checkMetricActiveAttr: "True if metric is active and collecting data",
		checkMetricNameAttr:   "The name of a metric",
		checkMetricTagsAttr:   "A list of tags assigned to a metric",
		checkMetricTypeAttr:   "Type of the metric",
		checkMetricUnitsAttr:  "Units for the metric",
	}

	// NOTE(sean@): needs to be completed
	collectorDescription = map[string]string{
		collectorDetailsAttr: "Details associated with individual collectors (a.k.a. broker)",
		collectorTagsAttr:    "Tags assigned to a collector",
	}

	// NOTE(sean@): needs to be completed
	contactDescription = map[string]string{
		contactSlackUsernameAttr: "Username Slackbot uses in Slack",
	}
}
