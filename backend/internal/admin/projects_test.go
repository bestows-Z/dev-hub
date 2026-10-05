package admin

import "testing"

func TestExistingBundlePreviewSurvivesProjectEdit(t *testing.T) {
	input := projectInput{Slug: "new-slug", Title: "Project", PreviewURL: "/api/v1/project-previews/old-slug/index.html"}
	if !validProjectInput(input) {
		t.Fatal("a project with an uploaded bundle must remain editable after its slug changes")
	}
}
