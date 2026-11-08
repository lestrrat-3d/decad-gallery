package spinningwheel

import (
	"context"

	"github.com/lestrrat-3d/decad"
)

// PreviewIssue identifies a part excluded from a construction preview.
type PreviewIssue struct {
	Part   string `json:"part"`
	Reason string `json:"reason"`
}

// Preview contains completed native parts before the assembly-wide joint cuts.
// It is a visual work-in-progress, not a manufactured assembly or a clearance proof.
type Preview struct {
	Assembly *Assembly
	Issues   []PreviewIssue
}

// BuildPreview runs the same part constructors as Build, recording failed parts
// and continuing with the remaining independent parts. It omits assembly-wide
// joint cuts. Only Assembly.Parts belong in the preview; the document can also
// contain tools left by failed constructors. ctx and doc must not be nil.
func BuildPreview(ctx context.Context, doc *decad.Document, p Params) (*Preview, error) {
	b, err := newAssemblyBuilder(ctx, doc, p)
	if err != nil {
		return nil, err
	}
	preview := &Preview{Assembly: &b.a}
	b.preview = preview
	b.buildInitialParts()
	if b.err == nil {
		b.placeParts()
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if b.err != nil {
		preview.Issues = append(preview.Issues, PreviewIssue{Part: b.name, Reason: b.err.Error()})
	}
	return preview, nil
}

func (b *builder) recordPreviewFailure(name string) {
	if b.err == nil {
		return
	}
	b.preview.Issues = append(b.preview.Issues, PreviewIssue{Part: name, Reason: b.err.Error()})
	b.err = nil
}
