package sanitizer

type HTMLStripperer interface {
	StripHTML(s string) string
}
