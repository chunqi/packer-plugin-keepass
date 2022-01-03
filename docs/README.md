# Packer Plugin Keepass

`packer-plugin-keepass` is a custom data source which allows the inclusion of credentials from a Keepass 2 database.

## Installation

### Using pre-built releases

#### Using the `packer init` command

Starting from version 1.7, Packer supports a new `packer init` command allowing
automatic installation of Packer plugins. Read the
[Packer documentation](https://www.packer.io/docs/commands/init) for more information.

To install this plugin, copy and paste this code into your Packer configuration .
Then, run [`packer init`](https://www.packer.io/docs/commands/init).

```hcl
packer {
  required_plugins {
    keepass = {
      version = ">= 0.2.0"
      source  = "github.com/chunqi/keepass"
    }
  }
}
```

#### Manual installation

You can find pre-built binary releases of the plugin [here](https://github.com/hashicorp/packer-plugin-name/releases).
Once you have downloaded the latest archive corresponding to your target OS,
uncompress it to retrieve the plugin binary file corresponding to your platform.
To install the plugin, please follow the Packer documentation on
[installing a plugin](https://www.packer.io/docs/extending/plugins/#installing-plugins).


#### From Source

If you prefer to build the plugin from its source code, clone the GitHub
repository locally and run the command `go build` from the root
directory. Upon successful compilation, a `packer-plugin-name` plugin
binary file can be found in the root directory.
To install the compiled plugin, please follow the official Packer documentation
on [installing a plugin](https://www.packer.io/docs/extending/plugins/#installing-plugins).


## Plugin Contents

The plugin implements a custom data source:

### Datasources

- [credentials](/docs/datasources/credentials.mdx) - Use the values of credential entries from a KeePass 2 database.

### Provioners

- [attachment](/docs/provisioners/attachment.mdx) - Upload file attachments contained within entries of a KeePass 2 database.
- [listing](/docs/provisioners/listing.mdx) - Generate a listing of all values and attachments of entries within a KeePass 2 database and the map keys by which to access them.
