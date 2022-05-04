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
				Config: testAccResourceApplyTFFromPath,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr(
						"tfcli_apply.test2", "output.string_var", regexp.MustCompile("MyResult")),
				),
			},

			{
				Config: testAccResourceApplyExtraFile,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr(
						"tfcli_apply.test3", "output.testoutput", regexp.MustCompile("testoutput")),
				),
			},
			{
				Config: testAccResourceApplyWithEnv,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr(
						"tfcli_apply.test4", "output.string_var", regexp.MustCompile("HelloEnv")),
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

const testAccResourceApplyTFFromPath = `
resource "tfcli_apply" "test2" {
	source  = "weakpixel/test-module/tfcli"
  	version = "0.0.2"
	vars = {
	"string_var" = "MyResult"
	}
}
`

const testAccResourceApplyExtraFile = `
resource "tfcli_apply" "test3" {
	source  = "weakpixel/test-module/tfcli"
  	version = "0.0.2"
	vars = {
	"string_var" = "MyResult"
	}
	extra_file {
		path    = "additional-output.tf"
		content = <<EOM
			output "testoutput" {
				value = "testoutput"
			}
		EOM
	}
}
`
const testAccResourceApplyWithEnv = `
resource "tfcli_apply" "test4" {
	terraform_version = "1.0.0"
	source  = "weakpixel/test-module/tfcli"
  	version = "0.0.2"
	  envs = {
		"TF_VAR_string_var" = "HelloEnv"
	  }
}
`
