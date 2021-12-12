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
      version = ">= 0.1.1"
      source  = "github.com/chunqi/keepass"
    }
  }
}
```

#### Manual installation

You can find pre-built binary releases of the plugin [here](https://github.com/chunqi/packer-plugin-keepass/releases).
Once you have downloaded the latest archive corresponding to your target OS,
uncompress it to retrieve the plugin binary file corresponding to your platform.
To install the plugin, please follow the Packer documentation on
[installing a plugin](https://www.packer.io/docs/extending/plugins/#installing-plugins).

### From Sources

If you prefer to build the plugin from sources, clone the GitHub repository
locally and run the command `go build` from the root
directory. Upon successful compilation, a `packer-plugin-keepass` plugin
binary file can be found in the root directory.
To install the compiled plugin, please follow the official Packer documentation
on [installing a plugin](https://www.packer.io/docs/extending/plugins/#installing-plugins).

### Configuration

For more information on how to configure the plugin, please read the
documentation located in the [`docs/`](docs) directory.

## Usage

```
packer {
  required_plugins {
    keepass = {
      version = ">= 0.1.1"
      source  = "github.com/chunqi/keepass"
    }
  }
}

variable "keepass_password" {
  type = string
  sensitive = true
}

data "keepass-credentials" "example" {
  keepass_file = "example/example.kdbx"
  keepass_password = "${var.keepass_password}"
}

source "file" "example" {
  content = format("%s:%s", data.keepass-credentials.example.map["F1ABA233DAE73E419937F475C593F31C-username"], data.keepass-credentials.example.map["/example/Sample Entry #2-password"])
  target = "credentials.txt"
}

build {
  sources = ["sources.file.example"]
}
```

The `keepass-credentials` custom data source takes in the path to the keepass
database file and the master password as parameters, and makes available the
entries within as a map.

The keepass master password can be passed in as either a command line argument
or as a packer environment variable.

```
$ packer build -var="keepass_password=password" example/data-var.pkr.hcl
```

```
$ export PKR_VAR_keepass_password=password
$ packer build example/data-var.pkr.hcl
```

The following map keys are constructed for each entry within the keepass
database:

* `<path-to-entry>/<title>-username`
* `<path-to-entry>/<title>-password`
* `<uuid>-username`
* `<uuid>-password`
* `<uuid>-title`

The `path-to-entry` is the names of the group (folder) names combined with the
`/` symbol.

The plugin will warn of ambiguous paths present in the keepass database in the
packer log. Note that only the first instance of any path will be accessible.

To find the `uuid` of each credential entry, in Keepass go to **View** ->
**Configure Columns...** and check the **UUID** column to be displayed.

![Keepass displaying UUID](/docs/datasources/keepass-uuid.png)

The example Keepass 2 database (`example/example.kdbx`) with sample entries will
thus generate the following keys and values:

```
/example/Sample Entry-username: User Name
/example/Sample Entry-password: Password
/example/Sample Entry #2-username: Michael321
/example/Sample Entry #2-password: 12345
F9E8062C3814F943BCBCB6FE81FAAA2F-username: User Name
F9E8062C3814F943BCBCB6FE81FAAA2F-password: Password
F9E8062C3814F943BCBCB6FE81FAAA2F-title: Sample Entry
F1ABA233DAE73E419937F475C593F31C-username: Michael321
F1ABA233DAE73E419937F475C593F31C-password: 12345
F1ABA233DAE73E419937F475C593F31C-title: Sample Entry #2
```

Building `example/data-var.pkr.hcl` will generate the following output in
`credentials.txt`:

```
Michael321:12345
```

## Troubleshooting

A general troubleshooting tip is to set the `PACKER_LOG` environment variable to
observe the verbose output from the plugin.

### File not found

```
Error: Datasource.Execute failed: open example.kdbx: no such file or directory

  on example/data.pkr.hcl line 1:
  (source code not available)
```

Note that the file path should be relative to the location where you invoke
`packer`.

### Wrong password

```
Error: Datasource.Execute failed: Wrong password? Database integrity check failed

  on example/data.pkr.hcl line 1:
  (source code not available)
```

Check that you have entered the master password correctly within the packer file
or the corresponding environment variable is set properly.

### Invalid index

```
Error: Invalid index

  on example/data.pkr.hcl line 7:
  (source code not available)

The given key does not identify an element in this collection value.
```

Make sure that you are referencing a valid key within the map.
