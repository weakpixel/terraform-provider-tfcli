terraform {
  required_providers {
    tfcli = {
      source  = "weakpixel/tfcli"
      version = "0.0.6"
    }
  }
}

provider "tfcli" {
  // Configure access to a private registry
  registry {
    host  = "private-registry.io"
    token = "asdf123"
  }
  // extra file for all terraform executions
  extra_file {
    path    = "backend.tf"
    content = <<EOM
            terraform { 
                backend "local" { }
            }
        EOM
  }
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

  extra_file {
    path    = "some-other-resource.tf"
    content = <<EOM
        resource "null_resource" "example1" {
            provisioner "local-exec" {
                command = "echo hello"
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
  value = tfcli_apply.hello_world.output["string_var"]
}

output "result" {
  value = tfcli_apply.hello_world.output
}