# v0.2.0

- Added all credential data values to the map (breaking change)
  - Breaking change due to the internal data value key names used (e.g. `UserName`, `Password`)

# v0.1.1

- Added group path keys for credentials
- Added warning in log when an ambiguous group path is encountered
  - Only the first instance for each group path is kept

# v0.1.0

- Credentials are now keyed by their UUID (breaking change)

# v0.0.1

- Credentials exposed as index keys in map