package nsone

// User wraps an NS1 /account/users resource
type User struct {
	Name        string               `json:"name"`
	Username    string               `json:"username"`
	Notify      NotificationSettings `json:"notify"`
	LastAccess  int                  `json:"last_access,omitempty"`
	Email       string               `json:"email"`
	Teams       []string             `json:"teams"`
	Permissions PermissionsMap       `json:"permissions"`
}

// NotificationSettings wraps a User's "notify" attribute
type NotificationSettings struct {
	Billing bool `json:"billing"`
}

// PermissionsMap wraps a User's "permissions" attribute
type PermissionsMap struct {
	Dns        PermissionsDns        `json:"dns"`
	Data       PermissionsData       `json:"data"`
	Account    PermissionsAccount    `json:"account"`
	Monitoring PermissionsMonitoring `json:"monitoring"`
}

// PermissionsDns wraps a User's "permissions.dns" attribute
type PermissionsDns struct {
	ViewZones           bool     `json:"view_zones"`
	ManageZones         bool     `json:"manage_zones"`
	ZonesAllowByDefault bool     `json:"zones_allow_by_default"`
	ZonesDeny           []string `json:"zones_deny"`
	ZonesAllow          []string `json:"zones_allow"`
}

// PermissionsData wraps a User's "permissions.data" attribute
type PermissionsData struct {
	PushToDatafeeds   bool `json:"push_to_datafeeds"`
	ManageDatasources bool `json:"manage_datasources"`
	ManageDatafeeds   bool `json:"manage_datafeeds"`
}

// PermissionsAccount wraps a User's "permissions.account" attribute
type PermissionsAccount struct {
	ManageUsers           bool `json:"manage_users"`
	ManagePaymentMethods  bool `json:"manage_payment_methods"`
	ManagePlan            bool `json:"manage_plan"`
	ManageTeams           bool `json:"manage_teams"`
	ManageApikeys         bool `json:"manage_apikeys"`
	ManageAccountSettings bool `json:"manage_account_settings"`
	ViewActivityLog       bool `json:"view_activity_log"`
	ViewInvoices          bool `json:"view_invoices"`
}

// PermissionsMonitoring wraps a User's "permissions.monitoring" attribute
type PermissionsMonitoring struct {
	ManageLists bool `json:"manage_lists"`
	ManageJobs  bool `json:"manage_jobs"`
	ViewJobs    bool `json:"view_jobs"`
}
