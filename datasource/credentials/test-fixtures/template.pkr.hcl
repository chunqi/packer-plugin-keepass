data "keepass-credentials" "test" {
  keepass_file = "example/example.kdbx"
  keepass_password = "password"
}

source "null" "basic-example" {
  communicator = "none"
}

build {
  sources = [
    "source.null.basic-example"
  ]

  provisioner "shell-local" {
    inline = [
      "echo username1: ${data.keepass-credentials.test.map["2-username"]}",
      "echo password1: ${data.keepass-credentials.test.map["2-password"]}",
    ]
  }
}
