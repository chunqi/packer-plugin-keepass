packer {
  required_plugins {
    keepass = {
      version = ">= 0.3.1"
      source  = "github.com/chunqi/keepass"
    }
  }
}

variable "keepass_password" {
  type = string
  sensitive = true
}

source "null" "example" {
  communicator = "none"
}

build {
  sources = ["sources.null.example"]

  provisioner "keepass-listing" {
      keepass_file="example/example.kdbx"
      keepass_password="${var.keepass_password}"
  }
}
