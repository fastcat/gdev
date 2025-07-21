package textedit

import "iter"

type Editor interface {
	// Next is called after each input line is read. If err is non-nil, the edit
	// operation will fail. Otherwise any lines in output will be emitted. This
	// must include the input line if it should be copied to the output.
	Next(line string) (output iter.Seq[string], err error)
	// EOF is called after all input lines have been processed through Next. Its
	// return will be processed the same way as Next.
	EOF() (output iter.Seq[string], err error)
}
