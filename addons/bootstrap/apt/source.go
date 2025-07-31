package apt

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
)

type AptSource struct {
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

func (s *AptSource) WithType(types ...string) *AptSource {
	s.Types = append(s.Types, types...)
	return s
}

func (s *AptSource) WithURI(uris ...string) *AptSource {
	s.URIs = append(s.URIs, uris...)
	return s
}

func (s *AptSource) WithSuite(suites ...string) *AptSource {
	s.Suites = append(s.Suites, suites...)
	return s
}

func (s *AptSource) WithComponent(components ...string) *AptSource {
	s.Components = append(s.Components, components...)
	return s
}

func (s *AptSource) WithArchitecture(architectures ...string) *AptSource {
	s.Architectures = append(s.Architectures, architectures...)
	return s
}

func (s *AptSource) WithSignedBy(signedBy string) *AptSource {
	s.SignedBy = signedBy
	return s
}

func (s *AptSource) defaults() {
	if len(s.Types) == 0 {
		s.Types = []string{"deb"}
	}
}

func (s *AptSource) ToDeb822() map[string]string {
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
		deb822["Architectures"] = strings.Join(s.Architectures, " ")
	}
	if s.SignedBy != "" {
		deb822["Signed-By"] = s.SignedBy
	}
	return deb822
}

func (s *AptSource) ToList() []byte {
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

func (s *AptSource) validate() error {
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
