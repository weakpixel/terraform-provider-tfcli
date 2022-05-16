package provider

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/weakpixel/tfcli"
)

func prepareLocalTestModule(t *testing.T) string {
	tmpDir := t.TempDir()
	writeFiles(context.Background(), tmpDir, ExtraFile{
		path:    "main.tf",
		content: []byte(testModuleSrc),
	})
	return tmpDir
}

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
				Config: testAccResourceApplyExtraFileForce,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr(
						"tfcli_apply.test_force", "output.testoutput", regexp.MustCompile("testoutput")),
				),
			},

			{
				Config: testAccResourceApplyWithEnv,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr(
						"tfcli_apply.test4", "output.string_var", regexp.MustCompile("HelloEnv")),
				),
			},
			{
				Config: fmt.Sprintf(testModulePath, prepareLocalTestModule(t)),
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr(
						"tfcli_apply.test_module_path", "output.string_var", regexp.MustCompile("TestModule")),
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
		cleanup = true
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

const testModulePath = `
resource "tfcli_apply" "test_module_path" {
	terraform_version = "1.0.0"
	module_path  = "%s"
	envs = {
		"TF_VAR_string_var" = "TestModule"
	}
}
`

const testAccResourceApplyExtraFileForce = `
resource "tfcli_apply" "test_force" {
	source  = "weakpixel/test-module/tfcli"
  	version = "0.0.2"
	extra_file {
		path    = "main.tf"
		cleanup = true
		force 	= true
		content = <<EOM
			output "testoutput" {
				value = "testoutput"
			}
		EOM
	}
}
`

const testModuleSrc = `
	variable "string_var" {
		description = "String variable"
		type = string
  	}
  
  	output "string_var" {
		value = var.string_var
  	}
`
