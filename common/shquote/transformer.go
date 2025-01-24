package shquote

import (
	"bytes"
	"fmt"
	"unicode"

	"golang.org/x/text/transform"
)

var _ transform.Transformer = (*Transformer)(nil)

type Transformer struct {
	transform.NopResetter
}

const (
	escapeChars = `\"'${[|&;<>()*?!` + "`"
	sq          = '\''
)

var sqExpansion = []byte(`'\''`)

// Transform implements [transform.Transformer].
func (t *Transformer) Transform(dst []byte, src []byte, atEOF bool) (nDst int, nSrc int, err error) {
	if !atEOF { // Expect that this function is always passed the whole input.
		return 0, 0, transform.ErrShortSrc
	}

	needsEsc := bytes.ContainsAny(src, escapeChars) || bytes.ContainsFunc(src, unicode.IsSpace)
	if needsEsc {
		// Calculate the output size we'll need:
		ct := bytes.Count(src, []byte("'")) * len(sqExpansion)
		ct += 2 // Start and end single-quotes
		sz := len(src) + ct
		if len(dst) < sz {
			return 0, 0, transform.ErrShortDst
		}

		outBuf := &buf{&dst, &nDst}
		srcBuf := &buf{&src, &nSrc}

		outBuf.WriteByte(sq)

		for len(src) > 0 {
			// Handle runs of non-singlequotes.
			idx := bytes.IndexByte(src, sq)
			switch idx {
			case 0:
			case -1:
				outBuf.Copy(srcBuf)
				continue // fall out of the loop
			default:
				if err := outBuf.CopyN(srcBuf, idx); err != nil {
					return nDst, nSrc, err
				}
			}

			if err := outBuf.Write(sqExpansion); err != nil {
				return nDst, nSrc, err
			}
			srcBuf.Advance(1)
		}

		outBuf.WriteByte(sq)
	} else {
		// Special/easy case: nothing to do.
		n := copy(dst, src)
		if n != len(src) {
			err = transform.ErrShortDst
		}
		nDst = n
		nSrc = n
	}

	return nDst, nSrc, err
}

type buf struct {
	buf *[]byte
	ct  *int
}

func (dst *buf) WriteByte(b byte) error {
	(*dst.buf)[0] = b
	dst.Advance(1)
	return nil
}

func (dst *buf) Write(s []byte) error {
	n := copy(*dst.buf, s)
	dst.Advance(n)
	if ct := len(s); n != ct {
		return fmt.Errorf("wtf: unable to copy %d bytes (did %d)", ct, n)
	}
	return nil
}

func (dst *buf) Advance(n int) {
	*dst.ct += n
	*dst.buf = (*dst.buf)[n:]
}

func (dst *buf) Copy(src *buf) {
	n := copy(*dst.buf, *src.buf)
	dst.Advance(n)
	src.Advance(n)
}

func (dst *buf) CopyN(src *buf, ct int) error {
	n := copy(*dst.buf, (*src.buf)[:ct])
	dst.Advance(n)
	src.Advance(n)
	if n != ct {
		return fmt.Errorf("wtf: unable to copy %d bytes (did %d)", ct, n)
	}
	return nil
}
