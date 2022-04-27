package provider

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/weakpixel/tfcli"
)

func resourceApply() *schema.Resource {
	return &schema.Resource{
		Description: "Applies the configured Terraform Module",

		CreateContext: resourceApplyCreate,
		ReadContext:   resourceApplyRead,
		UpdateContext: resourceApplyUpdate,
		DeleteContext: resourceApplyDelete,

		Schema: map[string]*schema.Schema{
			"terraform_version": {
				Description: "Terraform Version",
				Type:        schema.TypeString,
				Optional:    true,
			},

			"source": {
				Description: "Terraform Module Source",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
			"version": {
				Description: "Terraform Module Version",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
			"vars": {
				Description: "Terraform module variables",
				Type:        schema.TypeMap,
				Optional:    true,
				ForceNew:    true,
			},
			"backend_config": {
				Description: "Terraform module backend variables",
				Type:        schema.TypeMap,
				Optional:    true,
				ForceNew:    true,
			},
			"envs": {
				Description: "Terraform Envrionment Variables",
				Type:        schema.TypeMap,
				Optional:    true,
				ForceNew:    true,
			},
			"registry": schemaRegistry(),
			"file":     schemaFile(),

			"result": {
				Description: "Terraform output",
				Type:        schema.TypeMap,
				Computed:    true,
				Sensitive:   true,
			},
		},
	}
}

func schemaFile() *schema.Schema {
	return &schema.Schema{
		Type:        schema.TypeList,
		Optional:    true,
		ForceNew:    true,
		Description: "Additional file for Terraform Module",
		// MaxItems: 1,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"path": {
					Type:        schema.TypeString,
					Required:    true,
					ForceNew:    true,
					Description: "Relative file path in Terraform module",
				},
				"content": {
					Type:        schema.TypeString,
					Required:    true,
					ForceNew:    true,
					Description: "File content",
				},
			},
		},
	}
}

func schemaRegistry() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Optional: true,
		ForceNew: true,
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

func writeFiles(ctx context.Context, d *schema.ResourceData, dir string) error {
	files := d.Get("file").([]interface{})

	for _, e := range files {
		raw := e.(map[string]interface{})
		content := raw["content"].(string)
		path := raw["path"].(string)
		fullpath := filepath.Join(dir, filepath.FromSlash(path))
		// ospath := filepath.FromSlash(path)
		targetdir := filepath.Dir(filepath.Dir(fullpath))
		tflog.Debug(ctx, "Write file: "+fullpath)
		err := os.MkdirAll(targetdir, 0777)
		if err != nil {
			return err
		}
		err = ioutil.WriteFile(fullpath, []byte(content), 0660)
		if err != nil {
			return err
		}
	}

	return nil
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

	backend_config := d.Get("backend_config").(map[string]interface{})
	if backend_config != nil {
		tflog.Debug(ctx, "With backend config: "+fmt.Sprintf("%+v", backend_config))
		cli.WithBackendVars(toStringMap(backend_config))
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

	err = writeFiles(ctx, d, cli.Dir())
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
	d.SetId("")
	return nil
}
