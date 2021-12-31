package common

import (
	"fmt"
	"os"

	"github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/tobischo/gokeepasslib/v3"
)

// Opens the keepass database file and decrypt with password
func OpenDatabase(keepass_file string, keepass_password string) (*gokeepasslib.Database, error) {
	file, err := os.Open(keepass_file)
	if err != nil {
		// file does not exist
		return nil, err
	}
	db := gokeepasslib.NewDatabase()
	db.Credentials = gokeepasslib.NewPasswordCredentials(keepass_password)
	err = gokeepasslib.NewDecoder(file).Decode(db)

	if err != nil {
		// incorrect password
		return nil, err
	}
	return db, nil
}

func CheckConfig(keepass_file string, keepass_password string) *packer.MultiError {
	// check that keepass_file and keepass_password are provided
	var errs *packer.MultiError
	if keepass_file == "" {
		errs = packer.MultiErrorAppend(errs, fmt.Errorf("The `keepass_file` must be provided."))
	}
	if keepass_password == "" {
		errs = packer.MultiErrorAppend(errs, fmt.Errorf("The `keepass_password` must be provided."))
	}
	return errs
}
