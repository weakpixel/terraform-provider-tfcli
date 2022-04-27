terraform {
  required_providers {
    tfcli = {
      source = "weakpixel/tfcli"
      version = "0.0.2"
    }
  }
}

provider "tfcli" {
  # Configuration options
}

resource "tfcli_apply" "example" {
    source  = "weakpixel/test-module/tfcli"
    version = "0.0.2" 
    vars = {
        "string_var" = "Hello"
    }
}

output "result_string_var" {
    value = tfcli_apply.example.result["string_var"]
}

output "result" {
    value = tfcli_apply.example.result
}