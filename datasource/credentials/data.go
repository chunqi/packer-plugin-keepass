//go:generate packer-sdc mapstructure-to-hcl2 -type Config,DatasourceOutput
package credentials

import (
	"fmt"
	"log"
	"packer-plugin-keepass/common"

	"github.com/hashicorp/hcl/v2/hcldec"
	"github.com/hashicorp/packer-plugin-sdk/hcl2helper"
	"github.com/hashicorp/packer-plugin-sdk/template/config"
	"github.com/hashicorp/packer-plugin-sdk/template/interpolate"
	"github.com/tobischo/gokeepasslib/v3"
	"github.com/zclconf/go-cty/cty"
)

type Config struct {
	KeepassFile     string `mapstructure:"keepass_file" required:"true"`
	KeepassPassword string `mapstructure:"keepass_password" required:"true"`

	ctx interpolate.Context
}

type Datasource struct {
	config Config
}

type DatasourceOutput struct {
	Map map[string]string `mapstructure:"map"`
}

func (d *Datasource) ConfigSpec() hcldec.ObjectSpec {
	return d.config.FlatMapstructure().HCL2Spec()
}

func (d *Datasource) Configure(raws ...interface{}) error {
	err := config.Decode(&d.config, nil, raws...)
	if err != nil {
		return err
	}
	if errs := common.CheckConfig(d.config.KeepassFile, d.config.KeepassPassword); errs != nil {
		return errs
	}
	return nil
}

func (d *Datasource) OutputSpec() hcldec.ObjectSpec {
	return (&DatasourceOutput{}).FlatMapstructure().HCL2Spec()
}

func (d *Datasource) Execute() (cty.Value, error) {
	output := DatasourceOutput{}
	emptyOutput := hcl2helper.HCL2ValueFromConfig(output, d.OutputSpec())
	db, err := common.OpenDatabase(d.config.KeepassFile, d.config.KeepassPassword)
	if err != nil {
		return emptyOutput, err
	}
	db.UnlockProtectedEntries()
	// walk the database tree and create map of entry values
	credentials := map[string]string{}
	entryCallback := func(entryPath string, entry gokeepasslib.Entry, depth int) {
		for _, valueData := range entry.Values {
			// entry value data keys are guaranteed by keepass to be unique
			key := fmt.Sprintf("%s-%s", entryPath, valueData.Key)
			credentials[key] = valueData.Value.Content
			log.Println(fmt.Sprintf("(value) %s", key))
		}
	}
	common.WalkDatabase(db, nil, entryCallback)
	output.Map = credentials
	return hcl2helper.HCL2ValueFromConfig(output, d.OutputSpec()), nil
}
