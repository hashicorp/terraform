package google

func canonicalizeServiceScope(scope string) string {
	// This is a convenience map of short names used by the gcloud tool
	// to the GCE auth endpoints they alias to.
	scopeMap := map[string]string{
		"bigquery":              "https://www.googleapis.com/auth/bigquery",
		"cloud-platform":        "https://www.googleapis.com/auth/cloud-platform",
		"cloud-source-repos":    "https://www.googleapis.com/auth/source.full_control",
		"cloud-source-repos-ro": "https://www.googleapis.com/auth/source.read_only",
		"compute-ro":            "https://www.googleapis.com/auth/compute.readonly",
		"compute-rw":            "https://www.googleapis.com/auth/compute",
		"datastore":             "https://www.googleapis.com/auth/datastore",
		"logging-write":         "https://www.googleapis.com/auth/logging.write",
		"monitoring":            "https://www.googleapis.com/auth/monitoring",
		"monitoring-write":      "https://www.googleapis.com/auth/monitoring.write",
		"pubsub":                "https://www.googleapis.com/auth/pubsub",
		"service-control":       "https://www.googleapis.com/auth/servicecontrol",
		"service-management":    "https://www.googleapis.com/auth/service.management.readonly",
		"sql":                   "https://www.googleapis.com/auth/sqlservice",
		"sql-admin":             "https://www.googleapis.com/auth/sqlservice.admin",
		"storage-full":          "https://www.googleapis.com/auth/devstorage.full_control",
		"storage-ro":            "https://www.googleapis.com/auth/devstorage.read_only",
		"storage-rw":            "https://www.googleapis.com/auth/devstorage.read_write",
		"taskqueue":             "https://www.googleapis.com/auth/taskqueue",
		"trace-append":          "https://www.googleapis.com/auth/trace.append",
		"trace-ro":              "https://www.googleapis.com/auth/trace.readonly",
		"useraccounts-ro":       "https://www.googleapis.com/auth/cloud.useraccounts.readonly",
		"useraccounts-rw":       "https://www.googleapis.com/auth/cloud.useraccounts",
		"userinfo-email":        "https://www.googleapis.com/auth/userinfo.email",
	}

	if matchedURL, ok := scopeMap[scope]; ok {
		return matchedURL
	}

	return scope
}
