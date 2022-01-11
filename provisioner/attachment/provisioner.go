//go:generate packer-sdc mapstructure-to-hcl2 -type Config

package attachment

import (
	"context"
	"fmt"
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
	// generate map of file attachments
	attachmentsMap := map[string]gokeepasslib.BinaryReference{}
	entryMap := map[string]gokeepasslib.Entry{}
	entryCallback := func(entryPath string, entry gokeepasslib.Entry, depth int) {
		entryMap[entryPath] = entry
		for _, attachment := range entry.Binaries {
			entryAttachmentPath := fmt.Sprintf("%s-%s", entryPath, attachment.Name)
			attachmentsMap[entryAttachmentPath] = attachment
		}
	}
	common.WalkDatabase(db, nil, entryCallback)
	if _, keyExists := attachmentsMap[attachmentPath]; keyExists {
		// if the specified attachmentPath is in the attachmentsMap, upload the attachment
		attachment := attachmentsMap[attachmentPath]
		// if the destination is a directory, append with the attachment file name
		if strings.HasSuffix(p.config.Destination, "/") {
			p.config.Destination = p.config.Destination + attachment.Name
		}
		return p.UploadAttachment(ui, communicator, db, attachment)
	} else if _, keyExists := entryMap[attachmentPath]; keyExists {
		// if the specified attachmentPath is an entry root path, upload all attachments within
		entry := entryMap[attachmentPath]
		attachmentsCount := len(entry.Binaries)
		ui.Say(fmt.Sprintf("Uploading %d attachments from entry %s", attachmentsCount, attachmentPath))
		return p.UploadAttachments(ui, communicator, db, entry.Binaries)
	} else {
		return fmt.Errorf("File attachment \"%s\" does not exist.", attachmentPath)
	}
}

// Upload a single file attachment to the destination path using a temp file
func (p *Provisioner) UploadAttachment(ui packer.Ui, communicator packer.Communicator, db *gokeepasslib.Database, attachment gokeepasslib.BinaryReference) error {
	// retrieve the attachment object
	attachmentBinary := attachment.Find(db)
	if attachmentBinary == nil {
		return fmt.Errorf("Could not find attachment binary for file: %s", attachment.Name)
	}
	ui.Say(fmt.Sprintf("Uploading %s => %s", attachment.Name, p.config.Destination))
	// create temp file for the attachment contents
	attachmentTempFile, err := os.CreateTemp(os.TempDir(), "keepass-attachment")
	if err != nil {
		return err
	}
	defer attachmentTempFile.Close()
	defer os.Remove(attachmentTempFile.Name())
	attachmentBytes, err := attachmentBinary.GetContentBytes()
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
		attachmentTempFileReader := ui.TrackProgress(attachment.Name, 0, attachmentTempFileInfo.Size(), attachmentTempFile)
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

// Upload entry file attachment(s) to the destination path using a temp dir
func (p *Provisioner) UploadAttachments(ui packer.Ui, communicator packer.Communicator, db *gokeepasslib.Database, attachments []gokeepasslib.BinaryReference) error {
	// create temp dir to hold file attachments
	attachmentsTempDir, err := os.MkdirTemp(os.TempDir(), "keepass-attachments")
	if err != nil {
		return err
	}
	// write each file attachment as a temp file
	for _, attachment := range attachments {
		// retrieve the attachment object
		attachmentBinary := attachment.Find(db)
		if attachmentBinary == nil {
			ui.Error(fmt.Sprintf("[WARNING] Could not find attachment binary for file: %s, skipping", attachment.Name))
			continue
		}
		// create temp file to hold attachment contents
		attachmentFile, err := os.Create(attachmentsTempDir + "/" + attachment.Name)
		if err != nil {
			return err
		}
		// write attachment contents to temp file
		attachmentBytes, err := attachmentBinary.GetContentBytes()
		if err != nil {
			return err
		} else {
			_, err := attachmentFile.Write(attachmentBytes)
			if err != nil {
				return err
			}
		}
		attachmentFile.Close()
		ui.Say(fmt.Sprintf("File: %s", attachment.Name))
	}
	// upload dir
	err = communicator.UploadDir(p.config.Destination, attachmentsTempDir+"/", nil)
	// cleanup temp dir and contents
	os.RemoveAll(attachmentsTempDir)
	return err
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
