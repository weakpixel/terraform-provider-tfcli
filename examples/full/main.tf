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

}