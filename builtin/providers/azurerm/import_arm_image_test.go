package azurerm

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccAzureRMImage_importStandalone(t *testing.T) {
	ri := acctest.RandInt()
	userName := "testadmin"
	password := "Password1234!"
	hostName := "tftestcustomimagesrc"
	dnsName := fmt.Sprintf("%[1]s.westcentralus.cloudapp.azure.com", hostName)
	sshPort := "22"
	preConfig := fmt.Sprintf(testAccAzureRMImage_standaloneImage_setup, ri, userName, password, hostName)
	postConfig := fmt.Sprintf(testAccAzureRMImage_standaloneImage_provision, ri, userName, password, hostName)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMImageDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				//need to create a vm and then reference it in the image creation
				Config:  preConfig,
				Destroy: false,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureVMExists("azurerm_virtual_machine.testsource", true),
					testGeneralizeVMImage(fmt.Sprintf("acctestRG-%[1]d", ri), "testsource",
						userName, password, dnsName, sshPort),
				),
			},
			resource.TestStep{
				Config: postConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMImageExists("azurerm_image.test", true),
				),
			},
			resource.TestStep{
				ResourceName:      "azurerm_image.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
