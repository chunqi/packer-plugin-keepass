package common

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/google/uuid"
	"github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/tobischo/gokeepasslib/v3"
)

// Opens the keepass database file and decrypt with password
func OpenDatabase(keepassFile string, keepassPassword string) (*gokeepasslib.Database, error) {
	file, err := os.Open(keepassFile)
	if err != nil {
		// file does not exist
		return nil, err
	}
	db := gokeepasslib.NewDatabase()
	db.Credentials = gokeepasslib.NewPasswordCredentials(keepassPassword)
	err = gokeepasslib.NewDecoder(file).Decode(db)
	if err != nil {
		// incorrect password
		return nil, err
	}
	return db, nil
}

// Walks the keepass database and constructs path keys for each entry
func WalkDatabase(db *gokeepasslib.Database,
	groupCallback func(string, gokeepasslib.Group, int),
	entryCallback func(string, gokeepasslib.Entry, int)) {
	pathMap := map[string]string{}
	for i := range db.Content.Root.Groups {
		walk("", 0, pathMap, db.Content.Root.Groups[i], groupCallback, entryCallback)
	}
}

func walk(path string, depth int, pathMap map[string]string, group gokeepasslib.Group,
	groupCallback func(string, gokeepasslib.Group, int),
	entryCallback func(string, gokeepasslib.Entry, int)) {
	// construct path for group
	groupPath := path + "/" + group.Name
	if groupCallback != nil {
		groupCallback(groupPath, group, depth)
	}
	for i := range group.Entries {
		entry := group.Entries[i]
		entryPath := fmt.Sprintf("%s/%s", groupPath, entry.GetTitle())
		// check for existence of entry path key
		if _, keyExists := pathMap[entryPath]; keyExists {
			// warn in log that an ambiguous path is encountered
			log.Println(fmt.Sprintf("[WARNING] Ambiguous path for entry: %s", entryPath))
			log.Println("[WARNING] Only the first entry with this path will be accessible")
		} else {
			// add entry path key to map and call callback function
			pathMap[entryPath] = ""
			entryCallback(entryPath, entry, depth)
		}
		// parse uuid bytes and convert to keepass UI format - no dashes and uppercase
		entryUUID, err := uuid.FromBytes(entry.UUID[:])
		if err == nil {
			entryUUIDString := strings.ReplaceAll(strings.ToUpper(entryUUID.String()), "-", "")
			entryCallback(entryUUIDString, entry, depth)
		} else {
			log.Println("[ERROR] Unable to parse UUID bytes for entry, the output map may be incomplete")
		}
	}
	// iterate through subgroups
	for i := range group.Groups {
		walk(groupPath, depth+1, pathMap, group.Groups[i], groupCallback, entryCallback)
	}
}

func CheckConfig(keepassFile string, keepassPassword string) *packer.MultiError {
	// check that keepass_file and keepass_password are provided
	var errs *packer.MultiError
	if keepassFile == "" {
		errs = packer.MultiErrorAppend(errs, fmt.Errorf("The `keepass_file` must be provided."))
	}
	if keepassPassword == "" {
		errs = packer.MultiErrorAppend(errs, fmt.Errorf("The `keepass_password` must be provided."))
	}
	return errs
}
