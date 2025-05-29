# CSV Tools

## Merge multiple csv files into a single xlsx file
```bash
go build -o to_xlsx src/cmd/to_xlsx.go
```

## Example CLI signature
```bash
./to_xlsx -src=<dir where csv files are> -dest=<dir where xlsx file should be created>
```

## Import multiple csv files into a single sqlite3 database file
```bash
go build -o to_sqlite src/cmd/to_sqlite.go
```

## Example CLI signature
```bash
./to_sqlite -src=<dir where csv files are> -dest=<dir where the sqlite file should be created>
```