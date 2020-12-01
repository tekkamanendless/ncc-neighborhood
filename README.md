# NCC Neighborhood 
This tool queries New Castle County for subdivisions and prints out a list of property information.

## Running
```
ncc-neighborhood > /path/to/results.csv
```

You may ask for help via:

```
ncc-neighborhood -help
```

## Building
To package this up for other people to consume:

```
GOOS=windows GOARCH=amd64 go build -o ncc-neighborhood.exe cmd/ncc-neighborhood/main.go
zip ncc-neighborhood.windows_amd64.zip ncc-neighborhood.exe
```

