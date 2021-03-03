# vespucci 
[![Build](https://github.com/ansel1/vespucci/workflows/Build/badge.svg)](https://github.com/ansel1/vespucci/actions?query=branch%3Amaster+workflow%3ABuild+) 
[![GoDoc](https://godoc.org/github.com/ansel1/vespucci/v4?status.png)](https://godoc.org/github.com/ansel1/vespucci/v4) 
[![Go Report Card](https://goreportcard.com/badge/github.com/ansel1/vespucci/v4)](https://goreportcard.com/report/github.com/ansel1/vespucci/v4)

vespucci implements utility functions for transforming values into a representation
using only the simple types used in golang's mapping to JSON:

- map[string]interface{}
- []interface{}
- float64
- string
- bool
- nil

This process is referred to as "normalizing" the value.  The package also offers
many useful utility functions for working with normalized values:

- Contains
- Equivalent
- Conflicts
- Keys
- Get
- Merge
- Empty
- Transform

These functions are useful when dealing with business values which may be represented
as structs, maps, or JSON, depending on the context.

Normalization will convert maps, slices, and primitives directly to one of the
types above.  For other values, it will fall back on marshaling the value to JSON,
then unmarshaling it into interface{}.  Raw JSON can be passed as a value by 
wrapping it in json.RawMessage:

    v, err := Normalize(json.RawMessage(b))

The mapstest package provides useful testing assertions, built on top of Contains
and Equivalent.  These are useful for asserting whether a value is approximately
equal to an expected value.  For example:

    jsonResp := httpget()
    mapstest.AssertContains(t, json.RawMessage(jsonResp), map[string]interface{}{
      "color":"red",
      "size":1,
    })

Because both values are normalized before comparison, either value can
be raw JSON, a struct, a map, a slice, or a primitive value.  Normalization
is recursive, so any of these types can be nested within each other.  And
there are useful assertion options controlling how loose the match can be.  
For example:

    v1 := map[string]interface{}{
      "color":"bigred",
      "size":0,
      "createdAt":time.Now().String,
    }

    v2 := map[string]interface{}{
      "color":"red",
      "size":1,
      "createdAt": time.Now,
    }

    mapstest.AssertContains(t, v1, v2, 
      maps.StringContains(),       // allows "bigred" to match "red"
      maps.EmptyValuesMatchAny(),  // allows size to match.  The presence and 
                                   // and type v2.size is checked
      maps.AllowTimeDelta(time.Second),  // allows v1.createdAt to be parsed into
                                         // a time.Time, and allows some skew between
                                         // v1.createdAt and v2.createdAt
    )
