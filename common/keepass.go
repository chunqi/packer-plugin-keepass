package common

import (
	"os"

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
