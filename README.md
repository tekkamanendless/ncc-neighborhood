# NCC Neighborhood 
This tool queries New Castle County for subdivisions and prints out a list of property information.

## Running
```
ncc-neighborhood > /path/to/results.csv
```

You may ask for help via:

```
ncc-neighborhood --help
```

To get all of the homes in Green Valley in New Castle County, run:
```
ncc-neighborhood 'GREEN VALLEY 1' 'GREEN VALLEY 2A' 'GREEN VALLEY 2B' 'GREEN VALLEY 3' 'GREEN VALLEY V'
```

## Building
To package this up for other people to consume:

```
GOOS=windows GOARCH=amd64 go build -o ncc-neighborhood.exe cmd/ncc-neighborhood/main.go
zip ncc-neighborhood.windows_amd64.zip ncc-neighborhood.exe
```

