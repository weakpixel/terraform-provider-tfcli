# Terraform-In-Terraform Provider

This provider allows running Terraform in Terraform. This might seem insane but there are some edge cases where it comes in handy.
The provider can either look up the Terraform executable from the PATH environment or it can download the Terraform binaries from the original source (set `tf_apply/terraform_version`). But please use that with caution, the binary is downloaded from the original Terraform sources but not validated for correctness. If you feel strongly about that I would love to see your contribution to my [tfcli](https://github.com/weakpixel/tfcli) GitLab project.

Note: The provider is tested on OSX and Linux. I do not know if it works on Windows, feedback is welcome.

The provider is available in the [Terraform Registry](https://registry.terraform.io/providers/weakpixel/tfcli/)


## Requirements

-	[Terraform](https://www.terraform.io/downloads.html) >= 0.13.x
-	[Go](https://golang.org/doc/install) >= 1.17

