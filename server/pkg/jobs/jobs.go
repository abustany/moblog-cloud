package jobs

import (
	"encoding/gob"
)

func init() {
	gob.Register(RenderJob{})
}

// RenderJob describes a job to render a blog into HTML pages
type RenderJob struct {
	Username   string // for debugging purposes
	AuthCookie string
	Repository string
}
