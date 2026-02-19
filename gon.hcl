# gon configuration for macOS notarization
source = ["./dist/taskboard-mcp_darwin_amd64_v1/taskboard-mcp", "./dist/taskboard-mcp_darwin_arm64/taskboard-mcp"]

bundle_id = "com.mnemoshare.taskboard-mcp"

apple_id {
  username = "@env:APPLE_ID"
  password = "@env:APPLE_APP_PASSWORD"
}

sign {
  application_identity = "Developer ID Application: Derrick Woolworth (45Q224N5J4)"
}

zip {
  output_path = "./dist/taskboard-mcp_macos_universal.zip"
}
