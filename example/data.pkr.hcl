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

data "keepass-credentials" "example" {
  keepass_file = "example/example.kdbx"
  keepass_password = "${var.keepass_password}"
}

source "file" "example" {
  content = format("%s:%s",
    data.keepass-credentials.example.map["F1ABA233DAE73E419937F475C593F31C-UserName"],
    data.keepass-credentials.example.map["/example/Sample Entry #2-Password"]
  )
  target = "credentials.txt"
}

build {
  sources = ["sources.file.example"]
}
