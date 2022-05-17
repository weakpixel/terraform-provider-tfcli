package provider

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
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
				Description: "Terraform Module Source. Not required if 'module_path' is set",
				Type:        schema.TypeString,
				Optional:    true,
			},
			"version": {
				Description: "Terraform Module Version. Not required if 'module_path' is set",
				Type:        schema.TypeString,
				Optional:    true,
			},
			"module_path": {
				Description: "Path to a local terraform module. Alternative to 'source' and 'version' ",
				Type:        schema.TypeString,
				Optional:    true,
			},
			"vars": {
				Description: "Terraform module variables",
				Type:        schema.TypeMap,
				Optional:    true,
			},
			"backend_config": {
				Description: "Terraform module backend variables",
				Type:        schema.TypeMap,
				Optional:    true,
			},
			"envs": {
				Description: "Terraform Envrionment Variables",
				Type:        schema.TypeMap,
				Optional:    true,
			},
			"registry":   schemaRegistry(),
			"extra_file": schemaFile(),

			"output": {
				Description: "Terraform output",
				Type:        schema.TypeMap,
				Computed:    true,
				// Sensitive:   true,
			},
		},
	}
}

func schemaFile() *schema.Schema {
	return &schema.Schema{
		Type:        schema.TypeList,
		Optional:    true,
		Description: "Additional file for Terraform Module",
		// MaxItems: 1,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"path": {
					Type:        schema.TypeString,
					Required:    true,
					Description: "Relative file path in Terraform module",
				},
				"content": {
					Type:        schema.TypeString,
					Required:    true,
					Description: "File content",
				},
				"force": {
					Type:        schema.TypeBool,
					Optional:    true,
					Description: "Set to true to overwrite existing file",
				},
				"cleanup": {
					Type:        schema.TypeBool,
					Optional:    true,
					Description: "Set to true to delete file after execution",
				},
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
					Description: "Terraform Registry Host",
				},
				"token": {
					Type:        schema.TypeString,
					Required:    true,
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

func toEnvStringMap(m map[string]interface{}) map[string]string {
	result := map[string]string{}
	for k, v := range m {
		result["TF_VAR_"+k] = fmt.Sprint(v)
	}
	return result
}

func cleanupFiles(ctx context.Context, dir string, files ...ExtraFile) {
	for _, f := range files {
		if f.cleanup {
			fullpath := filepath.Join(dir, filepath.FromSlash(f.path))
			os.Remove(fullpath)
		}
	}
}

func writeFiles(ctx context.Context, dir string, files ...ExtraFile) error {
	for _, f := range files {
		if f.force {
			continue
		}
		fullpath := filepath.Join(dir, filepath.FromSlash(f.path))
		if _, err := os.Stat(fullpath); errors.Is(err, os.ErrNotExist) {
			continue
		} else {
			return fmt.Errorf("cannot write extra file (%s) because target module has a file with the same name already. Use 'force' to overwrite file", f.path)
		}
	}

	for _, f := range files {
		fullpath := filepath.Join(dir, filepath.FromSlash(f.path))
		// ospath := filepath.FromSlash(path)
		targetdir := filepath.Dir(filepath.Dir(fullpath))
		tflog.Debug(ctx, "Write file: "+fullpath)
		err := os.MkdirAll(targetdir, 0777)
		if err != nil {
			return err
		}
		err = ioutil.WriteFile(fullpath, f.content, 0660)
		if err != nil {
			return err
		}
	}

	return nil
}

type runner func(ctx context.Context, cli tfcli.Terraform) error

func terraformBin(ctx context.Context, d *schema.ResourceData) (string, diag.Diagnostics) {
	terraform_version := d.Get("terraform_version").(string)
	bin := ""
	var err error
	if terraform_version == "" {
		tflog.Debug(ctx, "Lookup Terraform executable")
		bin, err = lookupTerraform()
		if err != nil {
			return "", diag.FromErr(err)
		}
	} else {
		tflog.Debug(ctx, "Download Terraform: "+terraform_version)
		bin, err = tfcli.DownloadTerraform(terraform_version, false)
		if err != nil {
			return "", diag.FromErr(err)
		}
	}
	tflog.Debug(ctx, "Terraform Bin: "+bin)
	return bin, diag.Diagnostics{}
}

func run(ctx context.Context, d *schema.ResourceData, meta interface{}, varsAsEnv bool, runner runner) diag.Diagnostics {
	client := meta.(*apiClient)
	bin, diags := terraformBin(ctx, d)
	if diags.HasError() {
		return diags
	}
	isLocalModule := d.Get("module_path") != ""

	dir := ""
	id := ""
	if !isLocalModule {
		// Prepare
		source := d.Get("source").(string)
		version := d.Get("version").(string)
		if source == "" {
			return diag.FromErr(fmt.Errorf("please provider either 'source' or 'module_path'"))
		}
		id = source + ":" + version

		var err error
		dir, err = ioutil.TempDir("", strings.ReplaceAll(source, "/", "_"))
		if err != nil {
			return diag.FromErr(err)
		}
		defer os.RemoveAll(dir)
		tflog.Debug(ctx, "Terraform working dir: "+dir)
	} else {
		dir = d.Get("module_path").(string)
		id = dir
	}

	stdRead, stdout := io.Pipe()
	errRead, stderr := io.Pipe()
	defer stdout.Close()
	defer stderr.Close()
	errorBuffer := &bytes.Buffer{}
	cli := tfcli.New(bin, dir).SetStdout(stdout).SetStderr(stderr)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	go func() {
		scanner := bufio.NewScanner(stdRead)
		// for {
		for scanner.Scan() {
			select {
			case <-ctx.Done():
				return
			default:
				tflog.Info(ctx, scanner.Text())
			}

		}
	}()

	go func() {
		scanner := bufio.NewScanner(errRead)
		// for {
		for scanner.Scan() {
			select {
			case <-ctx.Done():
				return
			default:
				txt := scanner.Text()
				tflog.Info(ctx, txt)
				errorBuffer.WriteString(txt)
			}

		}
	}()

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

	vars := d.Get("vars").(map[string]interface{})
	if vars != nil {
		tflog.Debug(ctx, "With vars: "+fmt.Sprintf("%+v", vars))
		if varsAsEnv {
			// Destroy must not fail because a variable does not exists
			// Providing variables as envrionment variable solves the issue.
			cli.AppendEnv(toEnvStringMap(vars))
		} else {
			cli.WithVars(toStringMap(vars))
		}
	}

	creds := registryCreds(ctx, d)
	creds = append(creds, client.registry...)
	if len(creds) > 0 {
		tflog.Debug(ctx, "With regestry credentials: "+fmt.Sprintf("%+v", creds))
		cli.WithRegistry(creds)
	}

	if !isLocalModule {
		source := d.Get("source").(string)
		version := d.Get("version").(string)
		tflog.Debug(ctx, "Download Terrform Module "+fmt.Sprintf("%s:%s", source, version))
		err := cli.GetModule(source, version)
		if err != nil {
			return diag.FromErr(fmt.Errorf("%s\nError: %s", errorBuffer.String(), err.Error()))
		}
	}

	extraFiles := parseExtraFiles(ctx, d)
	extraFiles = append(extraFiles, client.extraFiles...)

	defer cleanupFiles(ctx, cli.Dir(), extraFiles...)
	err := writeFiles(ctx, cli.Dir(), extraFiles...)
	if err != nil {
		return diag.FromErr(fmt.Errorf("%s\nError: %s", errorBuffer.String(), err.Error()))
	}

	tflog.Info(ctx, "Terrform Init")
	err = cli.Init()
	if err != nil {
		return diag.FromErr(fmt.Errorf("%s\nTerraform init failed: %s", errorBuffer.String(), err.Error()))
	}

	err = runner(ctx, cli)
	if err != nil {
		return diag.FromErr(fmt.Errorf("%s\n Error: %s", errorBuffer.String(), err.Error()))
	}
	d.SetId(id)
	return diag.Diagnostics{}
}

func resourceApplyCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	return run(ctx, d, meta, false, func(ctx context.Context, cli tfcli.Terraform) error {

		tflog.Debug(ctx, "Terrform Plan")
		planFile := filepath.Join(cli.Dir(), ".plan.json")
		defer os.Remove(planFile)
		err := cli.Plan(planFile)
		if err != nil {
			return fmt.Errorf("terraform plan failed: %s", err.Error())
		}

		tflog.Debug(ctx, "Terrform Apply")
		err = cli.ApplyWithPlan(planFile)
		if err != nil {
			return fmt.Errorf("terraform apply failed: %s", err.Error())
		}
		result, err := cli.Output()
		if err != nil {
			return fmt.Errorf("cannot get Terraform output: %s", err.Error())

		}
		d.Set("output", result)
		return nil
	})
}

func resourceApplyRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// TODO?
	return nil
}

func resourceApplyUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	return resourceApplyCreate(ctx, d, meta)
}

func resourceApplyDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	return run(ctx, d, meta, true, func(ctx context.Context, cli tfcli.Terraform) error {
		tflog.Debug(ctx, "Terrform Destroy")
		err := cli.Destroy()
		if err != nil {
			return fmt.Errorf("terraform destroy failed: %s", err.Error())
		}
		d.SetId("")
		return nil
	})

}
