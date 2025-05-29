# CSV Tools

### Dependencies

- Go version 1.24.x
- ```bash
     brew update
     brew install go-task/tap/go-task golangci-lint
  ```

## Merge multiple csv files into a single xlsx file
```bash
task build_to_xlsx
```

## Example CLI signature
```bash
./to_xlsx -src=<dir where csv files are> -dest=<dir where xlsx file should be created>
```

## Import multiple csv files into a single sqlite3 database file
```bash
task build_to_sqlite
```

## Example CLI signature
```bash
./to_sqlite -src=<dir where csv files are> -dest=<dir where the sqlite file should be created>
```