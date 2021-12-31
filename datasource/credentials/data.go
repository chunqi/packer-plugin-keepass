//go:generate packer-sdc mapstructure-to-hcl2 -type Config,DatasourceOutput
package credentials

import (
	"fmt"
	"log"
	"packer-plugin-keepass/common"
	"strings"

	"github.com/google/uuid"
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
	credentials := map[string]string{}
	db, err := common.OpenDatabase(d.config.KeepassFile, d.config.KeepassPassword)
	if err != nil {
		return emptyOutput, err
	}
	// walk the database tree and collect credentials
	db.UnlockProtectedEntries()
	for i := range db.Content.Root.Groups {
		walk("", credentials, db.Content.Root.Groups[i])
	}
	output.Map = credentials
	return hcl2helper.HCL2ValueFromConfig(output, d.OutputSpec()), nil
}

func walk(path string, credentials map[string]string, group gokeepasslib.Group) {
	// construct path for group
	groupPath := path + "/" + group.Name
	for i := range group.Entries {
		entry := group.Entries[i]
		// each entry will be keyed by:
		// - group path (if not ambiguous)
		// - uuid (always)
		title := entry.GetTitle()
		entryPath := fmt.Sprintf("%s/%s", groupPath, title)
		// check for presence of entry path key
		if _, keyExists := credentials[entryPath]; keyExists {
			// warn in log that an ambiguous path is encountered
			log.Println(fmt.Sprintf("[WARNING] Ambiguous path for entry: %s", entryPath))
			log.Println("[WARNING] Only the first entry with this path will be accessible")
		} else {
			// add entry path key to map
			credentials[entryPath] = ""
			// add all values for entry, keyed by group path and title
			addEntryValues(entryPath, credentials, entry.Values)
		}
		// parse uuid bytes and convert to keepass UI format - no dashes and uppercase
		entryUUID, err := uuid.FromBytes(entry.UUID[:])
		if err == nil {
			entryUUIDString := strings.ReplaceAll(strings.ToUpper(entryUUID.String()), "-", "")
			addEntryValues(entryUUIDString, credentials, entry.Values)
		} else {
			log.Println("[ERROR] Unable to parse UUID bytes for entry, the output map may be incomplete")
		}
	}
	// iterate through subgroups
	for i := range group.Groups {
		walk(groupPath, credentials, group.Groups[i])
	}
}

func addEntryValues(keyPrefix string, credentials map[string]string, values []gokeepasslib.ValueData) {
	for _, valueData := range values {
		// entry value data keys are guaranteed by keepass to be unique
		key := fmt.Sprintf("%s-%s", keyPrefix, valueData.Key)
		credentials[key] = valueData.Value.Content
		log.Println(fmt.Sprintf("Key: %s", key))
	}
}
