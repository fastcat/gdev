# DpkgComp [![Coverage](http://gocover.io/_badge/github.com/quentin-m/dpkgcomp?0)](http://gocover.io/github.com/quentin-m/dpkgcomp) [![GoDoc](http://godoc.org/github.com/quentin-m/dpkgcomp?status.png)](http://godoc.org/github.com/quentin-m/dpkgcomp)

## About
This Go library compares Debian-like package version numbers.
For instance, it can tell that version *1:1.25-4* is lower than *1:1.25-8*.

The implementation is based on http://man.he.net/man5/deb-version and on https://www.debian.org/doc/debian-policy/ch-controlfields.html#s-f-Version. The algorithm is inspired from *dpkg-1.17.25* (*lib/version.c* and *lib/parsehelp.c*).

## Installing
Installing this library is easy-peasy, as usual.
```
go get github.com/quentin-m/dpkgcomp
```

You could also run tests
```
go test github.com/quentin-m/dpkgcomp
```

## Usage
Will you be able to guess the ouput ?
```Go
package main

import (
	"log"

	"github.com/quentin-m/dpkgcomp"
)
func main() {
	// Define our version numbers
	v1 := "0:1.18.36-a"
	v2 := "1.18.36"

    // Compare our versions
	cmp, err := dpkgcomp.Compare(v1, v2)
	if err != nil {
		log.Fatalln(err)
	}
	if cmp == 0 {
		log.Println(v1, "is equal to", v2)
	} else if cmp == 1 {
		log.Println(v1, "is higher than", v2)
	} else {
		log.Println(v1, "is lower than", v2)
	}
}
```
