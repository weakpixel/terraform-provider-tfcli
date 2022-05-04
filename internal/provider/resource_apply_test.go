package provider

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/weakpixel/tfcli"
)

func TestAccResourceApply(t *testing.T) {

	bin, err := tfcli.DownloadTerraform("1.1.9", false)
	if err != nil {
		t.Logf("cannot download Terraform: %s", err)
		t.FailNow()
	}
	dir := filepath.Dir(bin)
	path := os.Getenv("PATH")

	os.Setenv("PATH", dir+string(os.PathListSeparator)+path)

	resource.UnitTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccResourceApplyWithTFDownload,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr(
						"tfcli_apply.test1", "output.string_var", regexp.MustCompile("Hello")),
				),
			},

			{
				Config: testAccResourceApplyTerraformFromPath,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr(
						"tfcli_apply.test2", "output.string_var", regexp.MustCompile("MyResult")),
				),
			},
		},
	})
}

const testAccResourceApplyWithTFDownload = `
resource "tfcli_apply" "test1" {
	terraform_version = "1.0.0"
	source  = "weakpixel/test-module/tfcli"
  	version = "0.0.2"
	vars = {
	"string_var" = "Hello"
	}
}
`

const testAccResourceApplyTerraformFromPath = `
resource "tfcli_apply" "test2" {
	source  = "weakpixel/test-module/tfcli"
  	version = "0.0.2"
	vars = {
	"string_var" = "MyResult"
	}
}
`

const testAccResourceApplyWithLegazyTF = `
resource "tfcli_apply" "foo" {
	terraform_version = "0.15.1"
	source  = "weakpixel/test-module/tfcli"
  	version = "0.0.2"
	vars = {
	"string_var" = "MyResult"
	}
}
`
