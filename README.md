# metasinter

`metasinter` is a Go command-line utility for running msfconsole in a pseudo termainl as a http-api

## Project structure

- `go.mod` - module definition and Go dependency requirements.
- `main.go` - entrypoint for the CLI tool.

## Requirements

- Go 1.20+ (or as specified in `go.mod`)

## Build

```bash
go build -o metasinter ./...
```

## Usage

Run the binary with flags and arguments configured in `main.go`.

```bash
./metasinter [flags]
```

If the tool requires specific input sources or configuration, add it here once available.

## Testing

If tests are added in future, run:

```bash
go test ./...
```

## Contributing

1. Fork the repository.
2. Create a feature branch.
3. Add or update tests.
4. Open a pull request with a clear description.

## License

Add license information here (e.g., MIT, Apache 2.0) or create a LICENSE file.
