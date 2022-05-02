# Terraform-In-Terraform Provider

Allows to run Terraform in Terraform. This might seem to be insane but there are some edge cases where it come in handy.
The provider can either look up the Terraform executable from the PATH environment or it can download Terrafrom binaires from the original source (set `tf_apply/terraform_version`). But please use that with caution, the binary is downloaded from the original Terraform sources but validated for correctness. If you feel strong about that I would love to see your contribution to my [tfcli](https://github.com/weakpixel/tfcli) gitlab project.

Note: The provider is tested on OSX and linux. I do not know if it works on Window, feedback is welcome.

## Requirements

-	[Terraform](https://www.terraform.io/downloads.html) >= 0.13.x
-	[Go](https://golang.org/doc/install) >= 1.17

