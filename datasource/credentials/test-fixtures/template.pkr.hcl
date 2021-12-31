data "keepass-credentials" "test" {
  keepass_file = "../../example/example.kdbx"
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
      "echo ${data.keepass-credentials.test.map["/example/Sample Entry-UserName"]}",
      "echo ${data.keepass-credentials.test.map["F1ABA233DAE73E419937F475C593F31C-UserName"]}",
    ]
  }
}
