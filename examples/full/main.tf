terraform {
  required_providers {
    tfcli = {
      source  = "weakpixel/tfcli"
      version = "0.0.4"
    }
  }
}

provider "tfcli" {

}

resource "tfcli_apply" "hello" {
  source  = "weakpixel/test-module/tfcli"
  version = "0.0.2"
  vars = {
    "string_var" = "Hello"
  }

  backend_config = {
    "path" : "/tmp/terraform-hello.tfstate"
  }

  // Add custom files to modify TF execution for specific backends
  file {
    path    = "backend.tf"
    content = <<EOM
        terraform { 
           backend "local" {
           }
        }
      EOM

  }
}

resource "tfcli_apply" "world" {
  source  = "weakpixel/test-module/tfcli"
  version = "0.0.2"
  envs = {
    "TF_VAR_string_var" = "World"
  }

}

resource "tfcli_apply" "hello_world" {
  source  = "weakpixel/test-module/tfcli"
  version = "0.0.2"
  vars = {
    "string_var" = format("%s %s", tfcli_apply.hello.result["string_var"], tfcli_apply.world.result["string_var"])
  }

}

output "result_string_var" {
  value     = tfcli_apply.hello_world.result["string_var"]
  sensitive = true
}

output "result" {
  value     = tfcli_apply.hello_world.result
  sensitive = true
}