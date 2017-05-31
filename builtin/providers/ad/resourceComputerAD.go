package ad

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	ldap "gopkg.in/ldap.v2"

	"github.com/hashicorp/terraform/helper/schema"
)

func resourceComputer() *schema.Resource {
	return &schema.Resource{
		Create: resourceComputerAdd,
		Read:   resourceComputerRead,
		Delete: resourceComputerDelete,
		Schema: map[string]*schema.Schema{
			"computer_name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"domain": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},
	}
}

func resourceComputerAdd(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ldap.Conn)

	computer_name := d.Get("computer_name").(string)
	domain := d.Get("domain").(string)
	var dn_of_vm string
	dn_of_vm += "cn=" + computer_name + ",cn=Computers"
	domain_arr := strings.Split(domain, ".")
	for _, item := range domain_arr {
		dn_of_vm += ",dc=" + item
	}

	log.Printf("[INFO] Name of the DN is : %s ", dn_of_vm)
	log.Printf("[INFO] Adding the Computer to the AD : %s ", computer_name)

	err := addVmToAD(computer_name, dn_of_vm, client)
	if err != nil {
		log.Printf("[ERROR] Error while adding a Computer to the AD : %s ", err)
		return fmt.Errorf("Error while adding a Computer to the AD %s", err)
	}
	log.Printf("[INFO] Computer Added to AD successfully: %s", computer_name)
	d.SetId(computer_name)
	d.Set("domain", domain)
	d.Set("computer_name", computer_name)

	return nil
}

func resourceComputerRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ldap.Conn)

	computerName := d.Get("computer_name").(string)
	domain := d.Get("domain").(string)
	var dnOfVM string
	domainArr := strings.Split(domain, ".")
	dnOfVM = "dc=" + domainArr[0]
	for index, item := range domainArr {
		if index == 0 {
			continue
		}
		dnOfVM += ",dc=" + item
	}
	log.Printf("[INFO] Name of the DN is : %s ", dnOfVM)
	log.Printf("[INFO] Deleting the Computer from the AD : %s ", computerName)

	searchRequest := ldap.NewSearchRequest(
		dnOfVM, // The base dn to search
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		"(&(objectClass=Computer)(cn="+computerName+"))", // The filter to apply
		[]string{"dn", "cn"},                             // A list attributes to retrieve
		nil,
	)

	sr, err := client.Search(searchRequest)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("[ERROR] Found " + strconv.Itoa(len(sr.Entries)) + " Entries")
	for _, entry := range sr.Entries {
		fmt.Printf("%s: %v\n", entry.DN, entry.GetAttributeValue("cn"))
	}
	if len(sr.Entries) == 0 {
		log.Println("[ERROR] Computer was not found")
		d.SetId("")
	}
	return nil
}

func resourceComputerDelete(d *schema.ResourceData, meta interface{}) error {
	resourceComputerRead(d, meta)
	if d.Id() == "" {
		log.Println("[ERROR] Cannot find Computer in the specified AD")
		return fmt.Errorf("[ERROR] Cannot find Computer in the specified AD")
	}
	client := meta.(*ldap.Conn)

	computer_name := d.Get("computer_name").(string)
	domain := d.Get("domain").(string)
	var dn_of_vm string
	dn_of_vm += "cn=" + computer_name + ",cn=Computers"
	domain_arr := strings.Split(domain, ".")
	for _, item := range domain_arr {
		dn_of_vm += ",dc=" + item
	}

	log.Printf("[INFO] Name of the DN is : %s ", dn_of_vm)
	log.Printf("[INFO] Deleting the Computer from the AD : %s ", computer_name)

	err := deleteVmFromAD(dn_of_vm, client)
	if err != nil {
		log.Printf("[ERROR] Error while Deleting a Computer from AD : %s ", err)
		return fmt.Errorf("Error while Deleting a Computer from AD %s", err)
	}
	log.Printf("[INFO] Computer deleted from AD successfully: %s", computer_name)
	return nil
}

func addVmToAD(computer_name string, dn_name string, ad_conn *ldap.Conn) error {
	addRequest := ldap.NewAddRequest(dn_name)
	addRequest.Attribute("objectClass", []string{"computer"})
	addRequest.Attribute("sAMAccountName", []string{computer_name})
	addRequest.Attribute("userAccountControl", []string{"4096"})
	err := ad_conn.Add(addRequest)
	if err != nil {
		return err
	}
	return nil
}

func deleteVmFromAD(dn_name string, ad_conn *ldap.Conn) error {
	delRequest := ldap.NewDelRequest(dn_name, nil)
	err := ad_conn.Del(delRequest)
	if err != nil {
		return err
	}
	return nil
}
