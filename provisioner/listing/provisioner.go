//go:generate packer-sdc mapstructure-to-hcl2 -type Config

package listing

import (
	"context"
	"fmt"
	"packer-plugin-keepass/common"
	"strings"

	"github.com/google/uuid"
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
	if errs := common.CheckConfig(p.config.KeepassFile, p.config.KeepassPassword); err != nil {
		return errs
	}
	return nil
}

var treeSpacer = "    "

func (p *Provisioner) Provision(_ context.Context, ui packer.Ui, _ packer.Communicator, generatedData map[string]interface{}) error {
	keepass_file, err := interpolate.Render(p.config.KeepassFile, &p.config.ctx)
	keepass_password, err := interpolate.Render(p.config.KeepassPassword, &p.config.ctx)
	db, err := common.OpenDatabase(keepass_file, keepass_password)
	if err != nil {
		return err
	}
	ui.Say(fmt.Sprintf("Credentials and attachments listing for: %s", keepass_file))
	// generate tree listing of groups and entries
	credentials := map[string]string{}
	for i := range db.Content.Root.Groups {
		walk(ui, "", 0, credentials, db.Content.Root.Groups[i])
	}
	return nil
}

func walk(ui packer.Ui, path string, depth int, credentials map[string]string, group gokeepasslib.Group) {
	// construct path for group
	groupPath := path + "/" + group.Name
	ui.Say(fmt.Sprintf("%s(group) %s", strings.Repeat(treeSpacer, depth), groupPath))
	for i := range group.Entries {
		entry := group.Entries[i]
		// each entry will be keyed by:
		// - group path (if not ambiguous)
		// - uuid (always)
		title := entry.GetTitle()
		entryPath := fmt.Sprintf("%s/%s", groupPath, title)
		// check for presence of entry path key
		if _, keyExists := credentials[entryPath]; keyExists {
			// warn that an ambiguous path is encountered
			ui.Error(fmt.Sprintf("%s(entry) %s", strings.Repeat(treeSpacer, depth), entryPath))
			ui.Error(fmt.Sprintf("[WARNING] Ambiguous path for entry: %s", entryPath))
			ui.Error("[WARNING] Only the first entry will be accessible using the path")
		} else {
			// add entry path key to map
			credentials[entryPath] = ""
			// print entry path key to ui
			ui.Say(fmt.Sprintf("%s(entry) %s", strings.Repeat(treeSpacer, depth), entryPath))
			// add all values for entry, keyed by group path and title
			addEntryValues(ui, entryPath, depth+1, credentials, entry.Values)
			addEntryAttachments(ui, entryPath, depth+1, entry.Binaries)
		}
		// parse uuid bytes and convert to keepass UI format - no dashes and uppercase
		entryUUID, err := uuid.FromBytes(entry.UUID[:])
		if err == nil {
			entryUUIDString := strings.ReplaceAll(strings.ToUpper(entryUUID.String()), "-", "")
			addEntryValues(ui, entryUUIDString, depth+1, credentials, entry.Values)
			addEntryAttachments(ui, entryUUIDString, depth+1, entry.Binaries)
		} else {
			ui.Error("[ERROR] Unable to parse UUID bytes for entry, the output map may be incomplete")
		}
	}
	// iterate through subgroups
	for i := range group.Groups {
		walk(ui, groupPath, depth+1, credentials, group.Groups[i])
	}
}

func addEntryValues(ui packer.Ui, keyPrefix string, depth int, credentials map[string]string, values []gokeepasslib.ValueData) {
	for _, valueData := range values {
		// entry value data keys are guaranteed by keepass to be unique
		key := fmt.Sprintf("%s-%s", keyPrefix, valueData.Key)
		credentials[key] = valueData.Value.Content
		ui.Say(fmt.Sprintf("%s(value) %s", strings.Repeat(treeSpacer, depth), key))
	}
}

func addEntryAttachments(ui packer.Ui, keyPrefix string, depth int, binaries []gokeepasslib.BinaryReference) {
	for _, attachment := range binaries {
		// attachment names are guaranteed by keepass to be unique
		key := fmt.Sprintf("%s-%s", keyPrefix, attachment.Name)
		ui.Say(fmt.Sprintf("%s (file) %s", strings.Repeat(treeSpacer, depth), key))
	}
}
