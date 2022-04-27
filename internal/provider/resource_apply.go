package provider

import (
	"bytes"
	"context"
	"io/ioutil"
	"os"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/weakpixel/tfcli"
)

func resourceApply() *schema.Resource {
	return &schema.Resource{
		// This description is used by the documentation generator and the language server.
		Description: "Applies the configured Terraform Module",

		CreateContext: resourceScaffoldingCreate,
		ReadContext:   resourceScaffoldingRead,
		UpdateContext: resourceScaffoldingUpdate,
		DeleteContext: resourceScaffoldingDelete,

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

func resourceScaffoldingCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// use the meta value to retrieve your client from the provider configure method
	// client := meta.(*apiClient)

	source := d.Get("source").(string)
	version := d.Get("version").(string)
	terraform_version := d.Get("terraform_version").(string)

	id := source + ":" + version
	d.SetId(id)

	tflog.Trace(ctx, "Download Terraform")

	if terraform_version == "" {
		// TODO: Use installed terraform if terraform_version is not defined
		terraform_version = "1.1.19"
	}

	bin, err := tfcli.DownloadTerraform(terraform_version, false)
	if err != nil {
		return diag.FromErr(err)
	}
	dir, err := ioutil.TempDir("", source)
	if err != nil {
		return diag.FromErr(err)
	}
	defer os.RemoveAll(dir)

	var buf bytes.Buffer
	cli := tfcli.New(bin, dir, &buf, &buf)

	vars := d.Get("vars").(map[string]string)
	if vars != nil {
		cli.WithVars(vars)
	}

	backend_vars := d.Get("backend_vars").(map[string]string)
	if backend_vars != nil {
		cli.WithBackendVars(vars)
	}

	envs := d.Get("envs").(map[string]string)
	if envs != nil {
		cli.WithEnv(envs)
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
		cli.WithRegistry(creds)
	}

	err = cli.GetModule(source, version)
	if err != nil {
		return diag.FromErr(err)
	}
	err = cli.Init()
	if err != nil {
		return diag.FromErr(err)
	}
	err = cli.Apply()
	if err != nil {
		return diag.FromErr(err)
	}
	return diag.Errorf("not implemented")
}

func resourceScaffoldingRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// use the meta value to retrieve your client from the provider configure method
	// client := meta.(*apiClient)

	return diag.Errorf("not implemented")
}

func resourceScaffoldingUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// use the meta value to retrieve your client from the provider configure method
	// client := meta.(*apiClient)

	return diag.Errorf("not implemented")
}

func resourceScaffoldingDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// use the meta value to retrieve your client from the provider configure method
	// client := meta.(*apiClient)

	return diag.Errorf("not implemented")
}
