//go:generate packer-sdc mapstructure-to-hcl2 -type Config,DatasourceOutput
package credentials

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/google/uuid"
	"github.com/hashicorp/hcl/v2/hcldec"
	"github.com/hashicorp/packer-plugin-sdk/hcl2helper"
	"github.com/hashicorp/packer-plugin-sdk/template/config"
	"github.com/tobischo/gokeepasslib/v3"
	"github.com/zclconf/go-cty/cty"
)

type Config struct {
	KeepassFile     string `mapstructure:"keepass_file"`
	KeepassPassword string `mapstructure:"keepass_password"`
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
	return nil
}

func (d *Datasource) OutputSpec() hcldec.ObjectSpec {
	return (&DatasourceOutput{}).FlatMapstructure().HCL2Spec()
}

func (d *Datasource) Execute() (cty.Value, error) {
	output := DatasourceOutput{}
	emptyOutput := hcl2helper.HCL2ValueFromConfig(output, d.OutputSpec())
	credentials := map[string]string{}
	// open the keepass 2 database and decrypt with password
	file, err := os.Open(d.config.KeepassFile)
	if err != nil {
		return emptyOutput, err
	}
	db := gokeepasslib.NewDatabase()
	db.Credentials = gokeepasslib.NewPasswordCredentials(d.config.KeepassPassword)
	err = gokeepasslib.NewDecoder(file).Decode(db)
	// handle incorrect password
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
		username := entry.GetContent("UserName")
		password := entry.GetPassword()
		title := entry.GetTitle()
		// group path keyed entries
		// warn in log if an ambiguous path is encountered
		pathUsername := fmt.Sprintf("%s/%s-username", groupPath, title)
		pathPassword := fmt.Sprintf("%s/%s-password", groupPath, title)
		if _, keyExists := credentials[pathUsername]; keyExists {
			log.Println(fmt.Sprintf("Ambiguous path for entry: %s/%s", groupPath, title))
			log.Println("Only the first entry with this path will be accessible")
		} else {
			credentials[pathUsername] = username
			credentials[pathPassword] = password
		}
		// uuid keyed entries
		// parse uuid bytes and convert to keepass UI format - no dashes and uppercase
		entryUUID, err := uuid.FromBytes(entry.UUID[:])
		entryUUIDString := strings.ReplaceAll(strings.ToUpper(entryUUID.String()), "-", "")
		if err != nil {
			log.Println(err)
		} else {
			credentials[fmt.Sprintf("%s-title", entryUUIDString)] = title
			credentials[fmt.Sprintf("%s-username", entryUUIDString)] = username
			credentials[fmt.Sprintf("%s-password", entryUUIDString)] = password
		}
	}
	// iterate through subgroups
	for i := range group.Groups {
		walk(groupPath, credentials, group.Groups[i])
	}
}
