package google

func canonicalizeServiceScope(scope string) string {
	// This is a convenience map of short names used by the gcloud tool
	// to the GCE auth endpoints they alias to.
	scopeMap := map[string]string{
		"bigquery":        "https://www.googleapis.com/auth/bigquery",
		"cloud-platform":  "https://www.googleapis.com/auth/cloud-platform",
		"compute-ro":      "https://www.googleapis.com/auth/compute.readonly",
		"compute-rw":      "https://www.googleapis.com/auth/compute",
		"datastore":       "https://www.googleapis.com/auth/datastore",
		"logging-write":   "https://www.googleapis.com/auth/logging.write",
		"monitoring":      "https://www.googleapis.com/auth/monitoring",
		"sql":             "https://www.googleapis.com/auth/sqlservice",
		"sql-admin":       "https://www.googleapis.com/auth/sqlservice.admin",
		"storage-full":    "https://www.googleapis.com/auth/devstorage.full_control",
		"storage-ro":      "https://www.googleapis.com/auth/devstorage.read_only",
		"storage-rw":      "https://www.googleapis.com/auth/devstorage.read_write",
		"taskqueue":       "https://www.googleapis.com/auth/taskqueue",
		"useraccounts-ro": "https://www.googleapis.com/auth/cloud.useraccounts.readonly",
		"useraccounts-rw": "https://www.googleapis.com/auth/cloud.useraccounts",
		"userinfo-email":  "https://www.googleapis.com/auth/userinfo.email",
	}

	if matchedUrl, ok := scopeMap[scope]; ok {
		return matchedUrl
	} else {
		return scope
	}
}
