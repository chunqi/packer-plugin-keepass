//go:generate packer-sdc mapstructure-to-hcl2 -type Config

package listing

import (
	"context"
	"fmt"
	"packer-plugin-keepass/common"
	"strings"

	"github.com/hashicorp/hcl/v2/hcldec"
	"github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/hashicorp/packer-plugin-sdk/template/config"
	"github.com/hashicorp/packer-plugin-sdk/template/interpolate"
	"github.com/tobischo/gokeepasslib/v3"
)

type Config struct {
	KeepassFile     string `mapstructure:"keepass_file" required:"true"`
	KeepassPassword string `mapstructure:"keepass_password" required:"true"`

	ctx interpolate.Context
}

type Provisioner struct {
	config Config
}

func (p *Provisioner) ConfigSpec() hcldec.ObjectSpec {
	return p.config.FlatMapstructure().HCL2Spec()
}

func (p *Provisioner) Prepare(raws ...interface{}) error {
	err := config.Decode(&p.config, &config.DecodeOpts{
		Interpolate:        true,
		InterpolateContext: &p.config.ctx,
	}, raws...)
	if err != nil {
		return err
	}
	return nil
}

var treeSpacer = "    "

func (p *Provisioner) Provision(_ context.Context, ui packer.Ui, _ packer.Communicator, generatedData map[string]interface{}) error {
	// check that the keepass_file and keepass_password config have been provided
	if errs := common.CheckConfig(p.config.KeepassFile, p.config.KeepassPassword); errs != nil {
		return errs
	}
	db, err := common.OpenDatabase(p.config.KeepassFile, p.config.KeepassPassword)
	if err != nil {
		return err
	}
	ui.Say(fmt.Sprintf("Credentials and attachments listing for: %s", p.config.KeepassFile))
	// walk database and print tree listing of groups and entries
	groupCallback := func(groupPath string, group gokeepasslib.Group, depth int) {
		if depth == 0 {
			ui.Say(fmt.Sprintf("%s(root)  %s", strings.Repeat(treeSpacer, depth), groupPath))
		} else {
			ui.Say(fmt.Sprintf("%s(group) %s", strings.Repeat(treeSpacer, depth), groupPath))
		}
	}
	entryCallback := func(entryPath string, entry gokeepasslib.Entry, depth int) {
		ui.Say(fmt.Sprintf("%s(entry) %s", strings.Repeat(treeSpacer, depth), entryPath))
		for _, valueData := range entry.Values {
			// entry value data keys are guaranteed by keepass to be unique
			key := fmt.Sprintf("%s-%s", entryPath, valueData.Key)
			ui.Say(fmt.Sprintf("%s(value) %s", strings.Repeat(treeSpacer, depth+1), key))
		}
		for _, attachment := range entry.Binaries {
			// attachment names are guaranteed by keepass to be unique
			key := fmt.Sprintf("%s-%s", entryPath, attachment.Name)
			ui.Say(fmt.Sprintf("%s(file)  %s", strings.Repeat(treeSpacer, depth+1), key))
		}
	}
	common.WalkDatabase(db, groupCallback, entryCallback)
	return nil
}
