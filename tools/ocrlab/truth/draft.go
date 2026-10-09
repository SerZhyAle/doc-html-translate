package truth

import "encoding/json"

// Draft copies all editable content and disagreements, while invalidating both signatures.
// It never mutates or overwrites the reviewed record it was copied from.
func Draft(a *Annotation) *Annotation {
	data, _ := json.Marshal(a)
	var out Annotation
	_ = json.Unmarshal(data, &out)
	out.Origin = OriginOCRSeed
	out.Review.AnnotatedBy = ""
	out.Review.AnnotatedOn = ""
	out.Review.CheckedBy = ""
	out.Review.CheckedOn = ""
	return &out
}
