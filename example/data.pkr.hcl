data "keepass-credentials" "example" {
  keepass_file = "example.kdbx"
  keepass_password = "password"
}

source "file" "example" {
  content = data.keepass-credentials.example.map["1-username"]
  target = "username.txt"
}

build {
  sources = ["sources.file.example"]
}
