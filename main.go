package main

import (
    "encoding/xml"
    "fmt"
    "image/color"
    "os"

    "github.com/golang/geo/s1"
    "github.com/golang/geo/s2"

    "github.com/dsoprea/go-logging"
    "github.com/jessevdk/go-flags"
    "github.com/twpayne/go-kml"
)

const (
    drawBoxesStartingAtLevel = 15
)

type cellParameters struct {
    Latitude  float64 `long:"latitude" description:"Latitude (decimal)" required:"true"`
    Longitude float64 `long:"longitude" description:"Longitude (decimal)" required:"true"`
    ToBinary  bool    `long:"to-binary" description:"Print as binary"`
    IsVerbose bool    `short:"v" long:"verbose" description:"Be verbose"`
    Level     int     `long:"level" description:"Specific level (defaults to 30)" default:"-1"`
}

type parentsParameters struct {
    CellToken string `long:"cell-token" description:"Cell token (hex string)" required:"true"`
}

type parentsKmlParameters struct {
    CellToken string `long:"cell-token" description:"Cell token (hex string)" required:"true"`
}

type parameters struct {
    Cell       cellParameters       `command:"cell" description:"Print cell information"`
    Parents    parentsParameters    `command:"parents" description:"Print all parents for the given cell"`
    ParentsKml parentsKmlParameters `command:"parents_kml" description:"Generate KML showing parents of the cell"`
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

    switch p.Active.Name {
    case "cell":
        handleCell()
    case "parents":
        handleParents()
    case "parents_kml":
        handleParentsKml()
    default:
        fmt.Printf("Subcommand not handled: [%s]\n", p.Active.Name)
        os.Exit(1)
    }
}

func handleParents() {
    token := arguments.Parents.CellToken

    cellId := s2.CellIDFromToken(token)
    if cellId.IsValid() == false {
        fmt.Printf("Cell not valid.\n")
        os.Exit(1)
    }

    for level := cellId.Level(); level > 0; level-- {
        parentCellId := cellId.Parent(level)
        parentLatitude, parentLongitude := coordinatesFromCell(parentCellId)

        fmt.Printf("%2d: %16s %064b  (%.10f, %.10f)\n", parentCellId.Level(), parentCellId.ToToken(), parentCellId, parentLatitude, parentLongitude)
    }
}

func handleParentsKml() {
    token := arguments.ParentsKml.CellToken

    cellId := s2.CellIDFromToken(token)
    if cellId.IsValid() == false {
        fmt.Printf("Cell not valid.\n")
        os.Exit(1)
    }

    lineageLineCoordinates := make([]kml.Coordinate, 0)

    // elements holds all placemarks (containers).
    elements := make([]kml.Element, 0)

    // Add style definitions.

    styleLine := kml.SharedStyle(
        "YellowLine",
        kml.LineStyle(
            kml.Color(color.RGBA{R: 255, G: 255, B: 0, A: 127}),
            kml.Width(4),
        ),
    )

    stylePlace := kml.SharedStyle(
        "RedPlaces",
        kml.LineStyle(
            kml.Color(color.RGBA{R: 255, G: 0, B: 0, A: 127}),
            kml.Width(4),
        ),
    )

    stylePolygon := kml.SharedStyle(
        "GreenPoly",
        kml.LineStyle(
            kml.Color(color.RGBA{R: 0, G: 255, B: 0, A: 127}),
            kml.Width(10),
        ),
    )

    elements = append(elements, styleLine, stylePlace, stylePolygon)

    // Process each level.

    defaultHeight := 100.0
    for level := cellId.Level(); level > 0; level-- {
        parentCellId := cellId.Parent(level)
        parentLatitude, parentLongitude := coordinatesFromCell(parentCellId)

        coordinate := kml.Coordinate{
            parentLongitude,
            parentLatitude,
            defaultHeight,
        }

        // Attach to line.

        lineageLineCoordinates = append(lineageLineCoordinates, coordinate)

        // Add specific placemark.

        parentPoint := kml.Placemark(
            kml.Name(fmt.Sprintf("Level (%d) %s", level, parentCellId.ToToken())),
            kml.StyleURL("#RedPlaces"),
            kml.Point(
                kml.Coordinates(coordinate),
            ),
        )

        elements = append(elements, parentPoint)

        // Add box around placemarks where we're at a large-enough level (the
        // higher/finer levels would be too clumped together).

        if level <= drawBoxesStartingAtLevel {
            parentCell := s2.CellFromCellID(parentCellId)
            parentRect := parentCell.RectBound()

            parentBoundsCoordinates := []kml.Coordinate{
                // Upper left
                kml.Coordinate{
                    float64(s1.Angle(parentRect.Lng.Lo) / s1.Degree),
                    float64(s1.Angle(parentRect.Lat.Hi) / s1.Degree),
                    defaultHeight,
                },

                // Upper right
                kml.Coordinate{
                    float64(s1.Angle(parentRect.Lng.Hi) / s1.Degree),
                    float64(s1.Angle(parentRect.Lat.Hi) / s1.Degree),
                    defaultHeight,
                },

                // Lower right
                kml.Coordinate{
                    float64(s1.Angle(parentRect.Lng.Hi) / s1.Degree),
                    float64(s1.Angle(parentRect.Lat.Lo) / s1.Degree),
                    defaultHeight,
                },

                // Lower left
                kml.Coordinate{
                    float64(s1.Angle(parentRect.Lng.Lo) / s1.Degree),
                    float64(s1.Angle(parentRect.Lat.Lo) / s1.Degree),
                    defaultHeight,
                },

                // Upper left
                kml.Coordinate{
                    float64(s1.Angle(parentRect.Lng.Lo) / s1.Degree),
                    float64(s1.Angle(parentRect.Lat.Hi) / s1.Degree),
                    defaultHeight,
                },
            }

            parentLevelPolygon := kml.Placemark(
                kml.Name(fmt.Sprintf("Level (%d) Bounds", level)),
                kml.StyleURL("#GreenPoly"),
                kml.LineString(
                    kml.Coordinates(parentBoundsCoordinates...),
                ),
            )

            elements = append(elements, parentLevelPolygon)
        }
    }

    // Attach the line.
    parentLine := kml.Placemark(
        kml.Name(fmt.Sprintf("Cell %s", token)),
        kml.StyleURL("#YellowLine"),
        kml.LineString(
            kml.Coordinates(lineageLineCoordinates...),
        ),
    )

    elements = append(elements, parentLine)

    // Build the document.

    k := kml.KML(
        kml.Document(
            elements...,
        ),
    )

    // Render the XML.

    e := xml.NewEncoder(os.Stdout)
    e.Indent("", "  ")

    err := e.Encode(k)
    log.PanicIf(err)
}

func handleCell() {
    // This is used rather than s2.LatLng() because that won't convert
    // properly:
    //
    //   ll := s2.LatLng{
    //       s1.Angle(arguments.Latitude),
    //       s1.Angle(arguments.Longitude),
    //   }
    //
    //   ll = ll.Normalized()
    //
    // It won't convert back to the original latitude/longitude (maybe one,
    // but not both).
    ll := s2.LatLngFromDegrees(arguments.Cell.Latitude, arguments.Cell.Longitude)

    if ll.IsValid() == false {
        fmt.Printf("Coordinates not valid.\n")
        os.Exit(2)
    }

    cellId := s2.CellIDFromLatLng(ll)
    if cellId.IsValid() == false {
        fmt.Printf("Cell not valid.\n")
        os.Exit(3)
    }

    if arguments.Cell.Level != -1 {
        cellId = cellId.Parent(arguments.Cell.Level)
    }

    if arguments.Cell.ToBinary == true {
        fmt.Printf("%031b\n", uint64(cellId))
    } else {
        fmt.Printf("%s\n", cellId.ToToken())
    }

    if arguments.Cell.IsVerbose == true {
        fmt.Printf("\n")

        fmt.Printf("Cell level: (%d)\n", cellId.Level())

        latitude, longitude := coordinatesFromCell(cellId)
        fmt.Printf("Coordinates: (%.10f), (%.10f)\n", latitude, longitude)
    }
}

func coordinatesFromCell(cellId s2.CellID) (latitude float64, longitude float64) {
    ll := cellId.LatLng()
    latitude = float64(ll.Lat / s1.Degree)
    longitude = float64(ll.Lng / s1.Degree)

    return latitude, longitude
}
