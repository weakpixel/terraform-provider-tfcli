package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/weakpixel/tfcli"
)

func init() {
	// Set descriptions to support markdown syntax, this will be used in document generation
	// and the language server.
	schema.DescriptionKind = schema.StringMarkdown

	// Customize the content of descriptions when output. For example you can add defaults on
	// to the exported descriptions if present.
	schema.SchemaDescriptionBuilder = func(s *schema.Schema) string {
		desc := s.Description
		if s.Default != nil {
			desc += fmt.Sprintf(" Defaults to `%v`.", s.Default)
		}
		return strings.TrimSpace(desc)
	}
}

func New(version string) func() *schema.Provider {
	return func() *schema.Provider {
		p := &schema.Provider{
			// DataSourcesMap: map[string]*schema.Resource{
			// 	"tf_data_source": dataSourceScaffolding(),
			// },
			ResourcesMap: map[string]*schema.Resource{
				"tfcli_apply": resourceApply(),
			},
			Schema: map[string]*schema.Schema{
				"registry":   schemaRegistry(),
				"extra_file": schemaFile(),
			},
		}

		p.ConfigureContextFunc = configure(version, p)

		return p
	}
}

type apiClient struct {
	registry   []tfcli.RegistryCredential
	extraFiles []ExtraFile
}

func configure(version string, p *schema.Provider) func(context.Context, *schema.ResourceData) (interface{}, diag.Diagnostics) {
	return func(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
		return &apiClient{
			registry:   registryCreds(ctx, d),
			extraFiles: parseExtraFiles(ctx, d),
		}, nil
	}
}

func registryCreds(ctx context.Context, d *schema.ResourceData) []tfcli.RegistryCredential {
	registry := d.Get("registry").([]interface{})
	creds := []tfcli.RegistryCredential{}
	for _, e := range registry {
		raw := e.(map[string]interface{})
		creds = append(creds, tfcli.RegistryCredential{
			Type:  raw["host"].(string),
			Token: raw["token"].(string),
		})
	}
	return creds
}

func parseExtraFiles(ctx context.Context, d *schema.ResourceData) []ExtraFile {
	files := d.Get("extra_file").([]interface{})
	result := []ExtraFile{}
	for _, e := range files {
		raw := e.(map[string]interface{})
		result = append(result, ExtraFile{
			path:    raw["path"].(string),
			content: []byte(raw["content"].(string)),
			force:   raw["force"].(bool),
			cleanup: raw["cleanup"].(bool),
		})
	}
	return result
}

type ExtraFile struct {
	content []byte
	path    string
	force   bool
	cleanup bool
}
