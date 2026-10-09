package corpus

import "testing"

func TestMachineMetadataIsNotRightsReview(t *testing.T) {
	for _, v := range []string{"", "commons-api", "tools/ocrlab/synth"} {
		s := Scene{Licence: LicencePD, LicenceVerifiedBy: v}
		if s.RightsReviewed() {
			t.Fatalf("machine stamp %q passed", v)
		}
	}
	s := Scene{Licence: LicenceSynthetic}
	if !s.RightsReviewed() {
		t.Fatal("synthetic has no third-party rights")
	}
}

func TestFamilyLeakageAndCycles(t *testing.T) {
	for _, child := range []Scene{{ID: "child", DerivedFrom: "middle", Split: SplitHoldout}, {ID: "child", SHA256: "same", Split: SplitHoldout}, {ID: "child", SourceItem: "shared", Split: SplitHoldout}} {
		m := &Manifest{Scenes: []Scene{{ID: "parent", Split: SplitDev, SHA256: "same", SourceItem: "shared"}, {ID: "middle", DerivedFrom: "parent", Split: SplitDev}, child}}
		if len(FamilyProblems(m)) == 0 {
			t.Fatalf("family admitted: %+v", child)
		}
	}
	m := &Manifest{Scenes: []Scene{{ID: "a", DerivedFrom: "b", Split: SplitDev}, {ID: "b", DerivedFrom: "a", Split: SplitDev}}}
	if len(FamilyProblems(m)) == 0 {
		t.Fatal("cycle admitted")
	}
}
