data "keepass-credentials" "example" {
  keepass_file = "example/example.kdbx"
  keepass_password = "password"
}

source "file" "example" {
  content = format("%s:%s", data.keepass-credentials.example.map["2-username"], data.keepass-credentials.example.map["2-password"])
  target = "credentials.txt"
}

build {
  sources = ["sources.file.example"]
}
