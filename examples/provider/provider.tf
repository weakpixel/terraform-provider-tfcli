terraform {
  required_providers {
    tfcli = {
      source  = "weakpixel/tfcli"
      version = "~> 0.0.6"
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

// Apply a terraform moduel from the registry
resource "tfcli_apply" "example" {
  source  = "weakpixel/test-module/tfcli"
  version = "0.0.2"

  // set variables 
  vars = {
    "string_var" = "Hello"
  }

  // configure backend
  backend_config = {
    "path" : "/tmp/terraform-hello.tfstate"
  }

  // make a minor modification to the module
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

output "result_string_var" {
  // Use output of the Terraform run
  value = tfcli_apply.example.output["string_var"]
}