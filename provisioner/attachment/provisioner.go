//go:generate packer-sdc mapstructure-to-hcl2 -type Config

package attachment

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"packer-plugin-keepass/common"

	"github.com/hashicorp/hcl/v2/hcldec"
	"github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/hashicorp/packer-plugin-sdk/template/config"
	"github.com/hashicorp/packer-plugin-sdk/template/interpolate"
	"github.com/tobischo/gokeepasslib/v3"
)

type Config struct {
	KeepassFile     string `mapstructure:"keepass_file" required:"true"`
	KeepassPassword string `mapstructure:"keepass_password" required:"true"`
	AttachmentPath  string `mapstructure:"attachment_path" required:"true"`
	Destination     string `mapstructure:"destination" required:"true"`

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

func (p *Provisioner) Provision(_ context.Context, ui packer.Ui, communicator packer.Communicator, generatedData map[string]interface{}) error {
	keepassFile, err := interpolate.Render(p.config.KeepassFile, &p.config.ctx)
	if err != nil {
		return fmt.Errorf("Error interpolating keepass_file: %s", err)
	}
	keepassPassword, err := interpolate.Render(p.config.KeepassPassword, &p.config.ctx)
	if err != nil {
		return fmt.Errorf("Error interpolating keepass_password: %s", err)
	}
	attachmentPath, err := interpolate.Render(p.config.AttachmentPath, &p.config.ctx)
	if err != nil {
		return fmt.Errorf("Error interpolating attachment_path: %s", err)
	}
	destination, err := interpolate.Render(p.config.Destination, &p.config.ctx)
	if err != nil {
		return fmt.Errorf("Error interpolating destination: %s", err)
	}
	// check that the keepass_file and keepass_password config have been provided
	if errs := common.CheckConfig(keepassFile, keepassPassword); errs != nil {
		return errs
	}
	// check that the attachment_path and destination config have been provided
	if errs := checkAttachmentConfig(attachmentPath, destination); errs != nil {
		return errs
	}
	db, err := common.OpenDatabase(keepassFile, keepassPassword)
	if err != nil {
		return err
	}
	// generate map of file attachmentsMap
	attachmentsMap := map[string]*gokeepasslib.Binary{}
	entryCallback := func(entryPath string, entry gokeepasslib.Entry, depth int) {
		for _, attachment := range entry.Binaries {
			attachmentBinary := attachment.Find(db)
			entryAttachmentPath := fmt.Sprintf("%s-%s", entryPath, attachment.Name)
			if attachmentBinary != nil {
				attachmentsMap[entryAttachmentPath] = attachmentBinary
				log.Println(fmt.Sprintf("(file) %s", entryAttachmentPath))
			} else {
				log.Println(fmt.Sprintf("[WARNING] Could not find attachment binary for file: %s", entryAttachmentPath))
			}
		}
	}
	common.WalkDatabase(db, nil, entryCallback)
	// check that the requested file is in the map
	if _, keyExists := attachmentsMap[attachmentPath]; keyExists {
		p.ProvisionUpload(ui, communicator, attachmentsMap[attachmentPath])
		return nil
	} else {
		return fmt.Errorf("File attachment \"%s\" does not exist.", attachmentPath)
	}
}

// Uploads the attachment file to the destination path using a temp file
func (p *Provisioner) ProvisionUpload(ui packer.Ui, communicator packer.Communicator, attachment *gokeepasslib.Binary) error {
	ui.Say(fmt.Sprintf("Uploading %s => %s", p.config.AttachmentPath, p.config.Destination))
	// create temp file to hold attachment contents
	attachmentTempFile, err := os.CreateTemp("", "keepass-attachment")
	if err != nil {
		return err
	}
	defer attachmentTempFile.Close()
	attachmentBytes, err := attachment.GetContentBytes()
	if err != nil {
		return err
	} else {
		// write attachment bytes to temp file and seek back to start for reading
		attachmentTempFile.Write(attachmentBytes)
		attachmentTempFile.Seek(0, 0)
		attachmentTempFileInfo, err := attachmentTempFile.Stat()
		if err != nil {
			return err
		}
		attachmentTempFileReader := ui.TrackProgress("test", 0, attachmentTempFileInfo.Size(), attachmentTempFile)
		defer attachmentTempFileReader.Close()
		if err = communicator.Upload(p.config.Destination, attachmentTempFileReader, &attachmentTempFileInfo); err != nil {
			if strings.Contains(err.Error(), "Error restoring file") {
				ui.Error(fmt.Sprintf("Upload failed: %s; this can occur when your file destination is a folder without a trailing slash.", err))
			}
			ui.Error(fmt.Sprintf("Upload failed: %s", err))
			return err
		}
		return nil
	}
}

// Check that attachment_path and destination config are provided
func checkAttachmentConfig(attachmentPath string, destination string) *packer.MultiError {
	var errs *packer.MultiError
	if attachmentPath == "" {
		errs = packer.MultiErrorAppend(errs, fmt.Errorf("The `attachment_path` must be provided."))
	}
	if destination == "" {
		errs = packer.MultiErrorAppend(errs, fmt.Errorf("The `destination` must be provided."))
	}
	return errs
}
