package jobs

// RenderJob describes a job to render a blog into HTML pages
type RenderJob struct {
	Username   string
	Repository string
}