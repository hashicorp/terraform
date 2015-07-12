package azure

const (
	// terraformAzureLabel is used as the label for the hosted service created
	// by Terraform on Azure.
	terraformAzureLabel = "terraform-on-azure"

	// terraformAzureDescription is the description used for the hosted service
	// created by Terraform on Azure.
	terraformAzureDescription = "Hosted service automatically created by terraform."
)

// parameterDescriptions holds a list of descriptions for all the available
// parameters of an Azure configuration.
var parameterDescriptions = map[string]string{
	// provider descriptions:
	"management_url": "The URL of the management API all requests should be sent to.\n" +
		"Defaults to 'https://management.core.windows.net/', which is the default Azure API URL.\n" +
		"This should be filled in only if you have your own datacenter with its own hosted management API.",
	"management_certificate": "The certificate for connecting to the management API specified with 'management_url'",
	"subscription_id":        "The subscription ID to be used when connecting to the management API.",
	"publish_settings_file":  "The publish settings file, either created by you or downloaded from 'https://manage.windowsazure.com/publishsettings'",
	// general resource descriptions:
	"name":         "Name of the resource to be created as it will appear in the Azure dashboard.",
	"service_name": "Name of the hosted service within Azure. Will have a DNS entry as dns-name.cloudapp.net",
	"location": "The Azure location where the resource will be located.\n" +
		"A list of Azure locations can be found here: http://azure.microsoft.com/en-us/regions/",
	"reverse_dns_fqdn": "The reverse of the fully qualified domain name. Optional.",
	"label":            "Label by which the resource will be identified by. Optional.",
	"description":      "Brief description of the resource. Optional.",
	// hosted service descriptions:
	"ephemeral_contents": "Sets whether the associated contents of this resource should also be\n" +
		"deleted upon this resource's deletion. Default is false.",
	// instance descriptions:
	"image":                          "The image the new VM will be booted from. Mandatory.",
	"size":                           "The size in GB of the disk to be created. Mandatory.",
	"os_type":                        "The OS type of the VM. Either Windows or Linux. Mandatory.",
	"storage_account":                "The storage account (pool) name. Mandatory.",
	"storage_container":              "The storage container name from the storage pool given with 'storage_pool'.",
	"user_name":                      "The user name to be configured on the new VM.",
	"user_password":                  "The user password to be configured on the new VM.",
	"default_certificate_thumbprint": "The thumbprint of the WinRM Certificate to be used as a default.",
	// local network descriptions:
	"vpn_gateway_address":    "The IP address of the VPN gateway bridged through this virtual network.",
	"address_space_prefixes": "List of address space prefixes in the format '<IP>/netmask'",
	// dns descriptions:
	"dns_address": "Address of the DNS server. Required.",
}
