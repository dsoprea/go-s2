## Overview

A very simple tool that prints the S2 cell-ID, in binary, for a given latitude/longitude pair.


## Usage

```
$ go get github.com/dsoprea/go-calculate-s2-location
$ cd "${GOPATH}/src/github.com/dsoprea/go-calculate-s2-location"

$ go run main.go --latitude 42.533333 --longitude -83.146389
1011101011001100010100000001011000101111101111110010101001101101

$ go run main.go --latitude 42.331389 --longitude -83.045833
1010111111100000110111011100110011100001000111000000000111011111
```
