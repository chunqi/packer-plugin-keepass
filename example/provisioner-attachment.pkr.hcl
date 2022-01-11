packer {
  required_plugins {
    keepass = {
      version = ">= 0.3.0"
      source  = "github.com/chunqi/keepass"
    }
  }
  required_plugins {
    docker = {
      version = ">= 0.0.7"
      source = "github.com/hashicorp/docker"
    }
  }
}

variable "keepass_password" {
  type = string
  sensitive = true
}

source "docker" "alpine" {
  image  = "alpine:3.15"
  commit = true
}

build {
  name    = "alpine"
  sources = [
    "source.docker.alpine"
  ]
  provisioner "keepass-attachment" {
    keepass_file = "example/example.kdbx"
    keepass_password = "${var.keepass_password}"
    attachment_path = "/example/Sample Entry-test.txt"
    destination = "/tmp/attachment.txt"
  }
  provisioner "keepass-attachment" {
    keepass_file = "example/example.kdbx"
    keepass_password = "${var.keepass_password}"
    attachment_path = "/example/Sample Entry"
    destination = "/tmp/"
  }
}
