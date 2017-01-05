package akamai

import (
    "fmt"

    "github.com/hashicorp/terraform/helper/schema"
)

// PapiProperties is a representation of the Akamai PAPI
// properties response available at:
// http://apibase.com/papi/v0/properties/?contractId=contractId&groupId=groupId
type PapiProperties struct {
    Properties struct {
        Items []PapiPropertySummary `json:"items"`
    } `json:"properties"`
}

// PapiPropertySummary is a representation of the Akamai PAPI
// property summary associated with each property returned by
// the properties response at:
// http://apibase.com/papi/v0/properties/?contractId=contractId&groupId=groupId
type PapiPropertySummary struct {
    AccountID         string `json:"accountId"`
    ContractID        string `json:"contractId"`
    GroupID           string `json:"groupId"`
    PropertyID        string `json:"propertyId"`
    Name              string `json:"propertyName"`
    LatestVersion     int    `json:"latestVersion"`
    StagingVersion    int    `json:"stagingVersion"`
    ProductionVersion int    `json:"productionVersion"`
    Note              string `json:"note"`
}

func resourceProperty() *schema.Resource {
    return &schema.Resource{
        Create: resourcePropertyCreate,
        Read:   resourcePropertyRead,
        Update: resourcePropertyUpdate,
        Delete: resourcePropertyDelete,

        Schema: map[string]*schema.Schema{
            "contract_id": &schema.Schema{
                Type:     schema.TypeString,
                Required: true,
                ForceNew: true,
                ValidateFunc: validateContractId,
            },
            "group_id": &schema.Schema{
                Type:     schema.TypeString,
                Required: true,
                ForceNew: true,
                ValidateFunc: validateGroupId,
            },
            "id": &schema.Schema{
                Type:     schema.TypeString,
                Optional: true,
                Computed: true,
            },
            "name": &schema.Schema{
                Type:     schema.TypeString,
                Required: true,
            },
        },
    }
}

func resourcePropertyCreate(d *schema.ResourceData, m interface{}) error {
    return nil
}

func resourcePropertyRead(d *schema.ResourceData, m interface{}) error {
    client := m.(*Client)
    props := &PapiProperties{}

    err := client.Get(fmt.Sprintf("properties/%d", d.Get("id").(int)), props)
    if err != nil {
        return err
    }

    property := props.Properties.Items[0]
    d.Set("name", property.Name)

    return nil
}

func resourcePropertyUpdate(d *schema.ResourceData, m interface{}) error {
    return nil
}

func resourcePropertyDelete(d *schema.ResourceData, m interface{}) error {
    return nil
}
