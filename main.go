package main

import (
    "fmt"
    "os"

    "github.com/golang/geo/s1"
    "github.com/golang/geo/s2"

    "github.com/jessevdk/go-flags"
)

type parameters struct {
    Latitude  float64 `long:"latitude" description:"Latitude (decimal)" required:"true"`
    Longitude float64 `long:"longitude" description:"Longitude (decimal)" required:"true"`
}

var (
    arguments = new(parameters)
)

func main() {
    p := flags.NewParser(arguments, flags.Default)

    _, err := p.Parse()
    if err != nil {
        os.Exit(1)
    }

    ll := s2.LatLng{
        s1.Angle(arguments.Latitude),
        s1.Angle(arguments.Longitude),
    }

    ll.Normalized()

    fmt.Printf("%031b\n", uint64(s2.CellIDFromLatLng(ll)))
}
