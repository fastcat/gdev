package apt

import (
	"bytes"
	"errors"
	"fmt"
	"slices"
	"strings"
)

type Source struct {
	// Optional. Will default to "deb" if unspecified.
	Types []string
	// Required. Usually only a single URI is given.
	URIs []string
	// Required
	Suites []string
	// Required
	Components []string
	// Optional. If not specified, apt will use all enabled architectures.
	Architectures []string
	// Technically optional, but required for most sources
	SignedBy string

	// Many more fields are possible, see man sources.list(5) for details. They
	// may be added if needed.
}

func (s *Source) WithType(types ...string) *Source {
	s.Types = append(s.Types, types...)
	return s
}

func (s *Source) WithURI(uris ...string) *Source {
	s.URIs = append(s.URIs, uris...)
	return s
}

func (s *Source) WithSuite(suites ...string) *Source {
	s.Suites = append(s.Suites, suites...)
	return s
}

func (s *Source) WithComponent(components ...string) *Source {
	s.Components = append(s.Components, components...)
	return s
}

func (s *Source) WithArchitecture(architectures ...string) *Source {
	s.Architectures = append(s.Architectures, architectures...)
	return s
}

func (s *Source) WithSignedBy(signedBy string) *Source {
	s.SignedBy = signedBy
	return s
}

func (s *Source) defaults() {
	if len(s.Types) == 0 {
		s.Types = []string{"deb"}
	}
}

func (s *Source) ToDeb822() map[string]string {
	s.defaults()
	deb822 := make(map[string]string)
	if len(s.Types) > 0 {
		deb822["Types"] = strings.Join(s.Types, " ")
	}
	if len(s.URIs) > 0 {
		deb822["URIs"] = strings.Join(s.URIs, " ")
	}
	if len(s.Suites) > 0 {
		deb822["Suites"] = strings.Join(s.Suites, " ")
	}
	if len(s.Components) > 0 {
		deb822["Components"] = strings.Join(s.Components, " ")
	}
	if len(s.Architectures) > 0 {
		deb822["Architectures"] = strings.Join(s.Architectures, ",")
	}
	if s.SignedBy != "" {
		deb822["Signed-By"] = s.SignedBy
	}
	return deb822
}

// FromDeb822 converts a deb822 formatted map to a Source.
//
// It does not check for logical validity of the result, it only parses the keys
// it recognizes and errors on any it doesn't.
func FromDeb822(deb822 map[string]string) (*Source, error) {
	s := &Source{}
	for k, v := range deb822 {
		switch k {
		case "Types":
			s.Types = strings.Fields(v)
		case "URIs":
			s.URIs = strings.Fields(v)
		case "Suites":
			s.Suites = strings.Fields(v)
		case "Components":
			s.Components = strings.Fields(v)
		case "Architectures":
			s.Architectures = strings.Split(v, ",")
		case "Signed-By":
			s.SignedBy = v
		default:
			return nil, fmt.Errorf("unknown deb822 key %q", k)
		}
	}
	return s, nil
}

func (s *Source) ToList() []byte {
	s.defaults()
	var b bytes.Buffer
	for _, t := range s.Types {
		for _, u := range s.URIs {
			for _, suite := range s.Suites {
				b.WriteString(t)
				b.WriteString(" ")
				if len(s.Architectures) > 0 || len(s.SignedBy) > 0 {
					b.WriteString("[")
					if len(s.Architectures) > 0 {
						b.WriteString("arch=")
						b.WriteString(strings.Join(s.Architectures, ","))
						if s.SignedBy != "" {
							b.WriteString(" ")
						}
					}
					if s.SignedBy != "" {
						b.WriteString("signed-by=")
						b.WriteString(s.SignedBy)
					}
					b.WriteString("] ")
				}
				b.WriteString(u)
				b.WriteString(" ")
				b.WriteString(suite)
				if len(s.Components) > 0 {
					b.WriteString(" ")
					b.WriteString(strings.Join(s.Components, " "))
				}
				b.WriteString("\n")
			}
		}
	}
	return b.Bytes()
}

func (s *Source) validate() error {
	var errs []error
	if len(s.URIs) == 0 {
		errs = append(errs, fmt.Errorf("no URIs specified"))
	}
	if len(s.Suites) == 0 {
		errs = append(errs, fmt.Errorf("no Suites specified"))
	}
	if len(s.Components) == 0 {
		errs = append(errs, fmt.Errorf("no Components specified"))
	}
	return errors.Join(errs...)
}

func (s *Source) Equal(other *Source) bool {
	// TODO: apply defaulting on Types to each before comparing?
	// TODO: ignore order in lists
	return slices.Equal(s.Types, other.Types) &&
		slices.Equal(s.URIs, other.URIs) &&
		slices.Equal(s.Suites, other.Suites) &&
		slices.Equal(s.Components, other.Components) &&
		slices.Equal(s.Architectures, other.Architectures) &&
		s.SignedBy == other.SignedBy
}
