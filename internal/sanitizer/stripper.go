package sanitizer

import "github.com/microcosm-cc/bluemonday"

type HTMLStripper struct {
	bm *bluemonday.Policy
}

// NewHTMLStripper return a new instance of blue monday policy
func NewHTMLStripper() *HTMLStripper {
	return &HTMLStripper{
		bm: bluemonday.StrictPolicy(),
	}
}

func (hs *HTMLStripper) StripHTML(s string) string {
	return hs.bm.Sanitize(s)
}
