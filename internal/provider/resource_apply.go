package provider

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/weakpixel/tfcli"
)

func resourceApply() *schema.Resource {
	return &schema.Resource{
		// This description is used by the documentation generator and the language server.
		Description: "Applies the configured Terraform Module",

		CreateContext: resourceApplyCreate,
		ReadContext:   resourceApplyRead,
		UpdateContext: resourceApplyUpdate,
		DeleteContext: resourceApplyDelete,

		Schema: map[string]*schema.Schema{
			"terraform_version": {
				// This description is used by the documentation generator and the language server.
				Description: "Terraform Version",
				Type:        schema.TypeString,
				Optional:    true,
			},

			"source": {
				// This description is used by the documentation generator and the language server.
				Description: "Terraform Module Source",
				Type:        schema.TypeString,
				// Optional:    false,
				Required: true,
			},
			"version": {
				// This description is used by the documentation generator and the language server.
				Description: "Terraform Module Version",
				Type:        schema.TypeString,
				// Optional:    false,
				Required: true,
			},
			"vars": {
				// This description is used by the documentation generator and the language server.
				Description: "Terraform module variables",
				Type:        schema.TypeMap,
				Optional:    true,
			},
			"backend_vars": {
				// This description is used by the documentation generator and the language server.
				Description: "Terraform module backend variables",
				Type:        schema.TypeMap,
				Optional:    true,
			},
			"envs": {
				// This description is used by the documentation generator and the language server.
				Description: "Terraform Envrionment Variables",
				Type:        schema.TypeMap,
				Optional:    true,
			},
			"registry": schemaRegistry(),

			"result": {
				// This description is used by the documentation generator and the language server.
				Description: "Terraform output",
				Type:        schema.TypeMap,
				Computed:    true,
				Sensitive:   true,
			},
		},
	}
}

func schemaRegistry() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Optional: true,
		// MaxItems: 1,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				// Required
				"host": {
					Type:        schema.TypeString,
					Required:    true,
					ForceNew:    true,
					Description: "Terraform Registry Host",
				},
				"token": {
					Type:        schema.TypeString,
					Required:    true,
					ForceNew:    true,
					Sensitive:   true,
					Description: "Terraform Registry access token",
				},
			},
		},
	}
}

func lookupTerraform() (string, error) {
	path, err := exec.LookPath("terraform")
	if err != nil {
		return "", fmt.Errorf("cannot find terraform executable in PATH")
	}
	return path, nil
}

func toStringMap(m map[string]interface{}) map[string]string {
	result := map[string]string{}
	for k, v := range m {
		result[k] = fmt.Sprint(v)
	}
	return result
}

func resourceApplyCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// use the meta value to retrieve your client from the provider configure method
	// client := meta.(*apiClient)

	source := d.Get("source").(string)
	version := d.Get("version").(string)
	terraform_version := d.Get("terraform_version").(string)

	id := source + ":" + version
	d.SetId(id)

	bin := ""
	var err error
	if terraform_version == "" {
		tflog.Debug(ctx, "Lookup Terraform executable")
		bin, err = lookupTerraform()
		if err != nil {
			return diag.FromErr(err)
		}
	} else {
		tflog.Debug(ctx, "Download Terraform: "+terraform_version)
		bin, err = tfcli.DownloadTerraform(terraform_version, false)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	tflog.Debug(ctx, "Terraform Bin: "+bin)

	dir, err := ioutil.TempDir("", strings.ReplaceAll(source, "/", "_"))
	if err != nil {
		return diag.FromErr(err)
	}
	defer os.RemoveAll(dir)

	tflog.Debug(ctx, "Terraform working dir: "+dir)
	var buf bytes.Buffer
	cli := tfcli.New(bin, dir, &buf, &buf)

	vars := d.Get("vars").(map[string]interface{})
	if vars != nil {
		tflog.Debug(ctx, "With vars: "+fmt.Sprintf("%+v", vars))
		cli.WithVars(toStringMap(vars))
	}

	backend_vars := d.Get("backend_vars").(map[string]interface{})
	if backend_vars != nil {
		tflog.Debug(ctx, "With backend vars: "+fmt.Sprintf("%+v", backend_vars))
		cli.WithBackendVars(toStringMap(vars))
	}

	envs := d.Get("envs").(map[string]interface{})
	if envs != nil {
		tflog.Debug(ctx, "With envs: "+fmt.Sprintf("%+v", envs))
		cli.WithEnv(toStringMap(envs))
	}

	registry := d.Get("registry").([]interface{})
	if registry != nil {
		creds := []tfcli.RegistryCredential{}
		for _, e := range registry {
			raw := e.(map[string]interface{})
			creds = append(creds, tfcli.RegistryCredential{
				Type:  raw["host"].(string),
				Token: raw["token"].(string),
			})
		}

		tflog.Debug(ctx, "With reg: "+fmt.Sprintf("%+v", creds))
		cli.WithRegistry(creds)
	}

	tflog.Debug(ctx, "Download Terrform Module "+fmt.Sprintf("%s:%s", source, version))
	err = cli.GetModule(source, version)
	if err != nil {
		tflog.Error(ctx, buf.String())
		return diag.FromErr(err)
	}

	tflog.Debug(ctx, "Terrform Init")
	err = cli.Init()
	if err != nil {
		tflog.Error(ctx, buf.String())
		return diag.FromErr(err)
	}
	tflog.Debug(ctx, "Terrform Apply")
	err = cli.Apply()
	if err != nil {
		tflog.Error(ctx, buf.String())
		return diag.FromErr(err)
	}

	result, err := cli.Output()
	if err != nil {
		tflog.Error(ctx, buf.String())
		return diag.FromErr(err)
	}
	d.Set("result", result)
	return nil
}

func resourceApplyRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// use the meta value to retrieve your client from the provider configure method
	// client := meta.(*apiClient)
	// return diag.Errorf("not implemented")
	return nil
}

func resourceApplyUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// use the meta value to retrieve your client from the provider configure method
	// client := meta.(*apiClient)

	// return diag.Errorf("not implemented")
	return nil
}

func resourceApplyDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// use the meta value to retrieve your client from the provider configure method
	// client := meta.(*apiClient)
	// return diag.Errorf("not implemented")
	return nil
}
