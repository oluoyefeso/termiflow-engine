package taste

import (
	"math"
	"testing"
	"time"
)

func TestUpdateProfile_NilProfile(t *testing.T) {
	profile := UpdateProfile(nil, nil)
	if profile == nil {
		t.Fatal("expected non-nil profile")
	}
	if profile.Version != 1 {
		t.Errorf("version = %d, want 1", profile.Version)
	}
	if profile.Affinities == nil {
		t.Error("expected non-nil affinities map")
	}
	if len(profile.Affinities) != 0 {
		t.Errorf("affinities length = %d, want 0", len(profile.Affinities))
	}
}

func TestUpdateProfile_ReadSignal(t *testing.T) {
	signals := []Signal{
		{ContentID: "a1", Kind: SignalRead, Value: 0.8, Topics: []string{"go"}},
	}
	profile := UpdateProfile(nil, signals)

	// Cold-start alpha = 0.3 (SignalCount was 0 < 20).
	// affinity = (1-0.3)*0 + 0.3*0.8 = 0.24
	want := 0.24
	got := profile.Affinities["go"]
	if math.Abs(got-want) > 0.001 {
		t.Errorf("affinity[go] = %.4f, want %.4f", got, want)
	}
	if profile.SignalCount != 1 {
		t.Errorf("signal count = %d, want 1", profile.SignalCount)
	}
}

func TestUpdateProfile_DislikeSignal(t *testing.T) {
	signals := []Signal{
		{ContentID: "a1", Kind: SignalDislike, Value: 1.0, Topics: []string{"javascript"}},
	}
	profile := UpdateProfile(nil, signals)

	// Cold-start alpha = 0.3. delta = -1.0
	// affinity = (1-0.3)*0 + 0.3*(-1.0) = -0.3
	want := -0.3
	got := profile.Affinities["javascript"]
	if math.Abs(got-want) > 0.001 {
		t.Errorf("affinity[javascript] = %.4f, want %.4f", got, want)
	}
}

func TestUpdateProfile_SkipSignal(t *testing.T) {
	signals := []Signal{
		{ContentID: "a1", Kind: SignalSkip, Value: 1.0, Topics: []string{"react"}},
	}
	profile := UpdateProfile(nil, signals)

	got := profile.Affinities["react"]
	if got >= 0 {
		t.Errorf("affinity[react] = %.4f, want negative", got)
	}
}

func TestUpdateProfile_DwellSignal(t *testing.T) {
	signals := []Signal{
		{ContentID: "a1", Kind: SignalDwell, Value: 0.9, Topics: []string{"rust"}},
	}
	profile := UpdateProfile(nil, signals)

	got := profile.Affinities["rust"]
	if got <= 0 {
		t.Errorf("affinity[rust] = %.4f, want positive", got)
	}
}

func TestUpdateProfile_EmptyTopicsIgnored(t *testing.T) {
	profile := &TasteProfile{
		Version:    1,
		Affinities: map[string]float64{"go": 0.5},
	}
	signals := []Signal{
		{ContentID: "a1", Kind: SignalRead, Value: 1.0, Topics: nil},
		{ContentID: "a2", Kind: SignalRead, Value: 1.0, Topics: []string{}},
	}
	profile = UpdateProfile(profile, signals)

	if profile.Affinities["go"] != 0.5 {
		t.Errorf("affinity[go] changed to %.4f, want 0.5", profile.Affinities["go"])
	}
	if profile.SignalCount != 0 {
		t.Errorf("signal count = %d, want 0 (empty topics should not count)", profile.SignalCount)
	}
}

func TestUpdateProfile_MultipleSignals(t *testing.T) {
	signals := []Signal{
		{ContentID: "a1", Kind: SignalRead, Value: 1.0, Topics: []string{"go"}},
		{ContentID: "a2", Kind: SignalRead, Value: 1.0, Topics: []string{"go"}},
		{ContentID: "a3", Kind: SignalRead, Value: 1.0, Topics: []string{"go"}},
		{ContentID: "a4", Kind: SignalSkip, Value: 1.0, Topics: []string{"go"}},
		{ContentID: "a5", Kind: SignalRead, Value: 1.0, Topics: []string{"go"}},
	}
	profile := UpdateProfile(nil, signals)

	// After 5 signals with cold-start alpha=0.3:
	// s1: 0.3*1.0 = 0.3
	// s2: 0.7*0.3 + 0.3*1.0 = 0.51
	// s3: 0.7*0.51 + 0.3*1.0 = 0.657
	// s4: 0.7*0.657 + 0.3*(-1.0) = 0.1599
	// s5: 0.7*0.1599 + 0.3*1.0 = 0.41193
	got := profile.Affinities["go"]
	want := 0.41193
	if math.Abs(got-want) > 0.01 {
		t.Errorf("affinity[go] = %.4f, want ~%.4f", got, want)
	}
	if profile.SignalCount != 5 {
		t.Errorf("signal count = %d, want 5", profile.SignalCount)
	}
}

func TestUpdateProfile_ClampToRange(t *testing.T) {
	// Start with extreme affinity and push it further.
	profile := &TasteProfile{
		Version:    1,
		Affinities: map[string]float64{"go": 0.99},
	}

	// Use high alpha to push past 1.0.
	signals := []Signal{
		{ContentID: "a1", Kind: SignalRead, Value: 1.0, Topics: []string{"go"}},
	}
	profile = UpdateProfile(profile, signals, WithAlpha(0.9))

	if profile.Affinities["go"] > 1.0 {
		t.Errorf("affinity[go] = %.4f, should be clamped to 1.0", profile.Affinities["go"])
	}

	// Push negative past -1.0.
	profile.Affinities["js"] = -0.99
	signals = []Signal{
		{ContentID: "a2", Kind: SignalDislike, Value: 1.0, Topics: []string{"js"}},
	}
	profile = UpdateProfile(profile, signals, WithAlpha(0.9))

	if profile.Affinities["js"] < -1.0 {
		t.Errorf("affinity[js] = %.4f, should be clamped to -1.0", profile.Affinities["js"])
	}
}

func TestUpdateProfile_WithAlpha(t *testing.T) {
	signals := []Signal{
		{ContentID: "a1", Kind: SignalRead, Value: 1.0, Topics: []string{"go"}},
	}
	profile := UpdateProfile(nil, signals, WithAlpha(0.5))

	// Explicit alpha=0.5 overrides cold-start alpha=0.3.
	// affinity = 0.5 * 1.0 = 0.5
	want := 0.5
	got := profile.Affinities["go"]
	if math.Abs(got-want) > 0.001 {
		t.Errorf("affinity[go] = %.4f, want %.4f", got, want)
	}
}

func TestUpdateProfile_InvalidAlpha(t *testing.T) {
	signals := []Signal{
		{ContentID: "a1", Kind: SignalRead, Value: 1.0, Topics: []string{"go"}},
	}

	// Alpha=0 should be rejected, cold-start default (0.3) used.
	profile := UpdateProfile(nil, signals, WithAlpha(0))
	want := 0.3
	got := profile.Affinities["go"]
	if math.Abs(got-want) > 0.001 {
		t.Errorf("with alpha=0: affinity[go] = %.4f, want %.4f (cold-start default)", got, want)
	}

	// Negative alpha should be rejected.
	profile = UpdateProfile(nil, signals, WithAlpha(-1))
	got = profile.Affinities["go"]
	if math.Abs(got-want) > 0.001 {
		t.Errorf("with alpha=-1: affinity[go] = %.4f, want %.4f (cold-start default)", got, want)
	}
}

func TestUpdateProfile_SignalCountIncrement(t *testing.T) {
	signals := []Signal{
		{ContentID: "a1", Kind: SignalRead, Value: 1.0, Topics: []string{"go"}},
		{ContentID: "a2", Kind: SignalRead, Value: 1.0, Topics: nil},        // skipped
		{ContentID: "a3", Kind: SignalLike, Value: 1.0, Topics: []string{}}, // skipped
		{ContentID: "a4", Kind: SignalSkip, Value: 1.0, Topics: []string{"js"}},
	}
	profile := UpdateProfile(nil, signals)

	if profile.SignalCount != 2 {
		t.Errorf("signal count = %d, want 2 (empty topics not counted)", profile.SignalCount)
	}
}

func TestUpdateProfile_LastUpdated(t *testing.T) {
	before := time.Now()
	profile := UpdateProfile(nil, []Signal{
		{ContentID: "a1", Kind: SignalRead, Value: 1.0, Topics: []string{"go"}},
	})
	after := time.Now()

	if profile.LastUpdated.Before(before) || profile.LastUpdated.After(after) {
		t.Errorf("LastUpdated = %v, want between %v and %v", profile.LastUpdated, before, after)
	}
}

func TestUpdateProfile_MultipleTopicsPerSignal(t *testing.T) {
	signals := []Signal{
		{ContentID: "a1", Kind: SignalRead, Value: 0.8, Topics: []string{"go", "concurrency"}},
	}
	profile := UpdateProfile(nil, signals)

	if _, ok := profile.Affinities["go"]; !ok {
		t.Error("expected affinity for 'go'")
	}
	if _, ok := profile.Affinities["concurrency"]; !ok {
		t.Error("expected affinity for 'concurrency'")
	}
	if profile.Affinities["go"] != profile.Affinities["concurrency"] {
		t.Errorf("both topics should have same affinity, got go=%.4f, concurrency=%.4f",
			profile.Affinities["go"], profile.Affinities["concurrency"])
	}
}

func TestUpdateProfile_UnknownKind(t *testing.T) {
	profile := &TasteProfile{
		Version:    1,
		Affinities: map[string]float64{"go": 0.5},
	}
	signals := []Signal{
		{ContentID: "a1", Kind: "bookmark", Value: 1.0, Topics: []string{"go"}},
	}
	profile = UpdateProfile(profile, signals)

	if profile.Affinities["go"] != 0.5 {
		t.Errorf("affinity[go] changed to %.4f, want 0.5 (unknown kind should be ignored)", profile.Affinities["go"])
	}
	if profile.SignalCount != 0 {
		t.Errorf("signal count = %d, want 0 (unknown kind should not count)", profile.SignalCount)
	}
}

func TestUpdateProfile_NilAffinitiesMap(t *testing.T) {
	profile := &TasteProfile{
		Version:    1,
		Affinities: nil, // possible from hand-constructed or minimal JSON
	}
	signals := []Signal{
		{ContentID: "a1", Kind: SignalRead, Value: 1.0, Topics: []string{"go"}},
	}
	profile = UpdateProfile(profile, signals)

	if profile.Affinities == nil {
		t.Fatal("expected non-nil affinities map after update")
	}
	if _, ok := profile.Affinities["go"]; !ok {
		t.Error("expected affinity for 'go'")
	}
}

func TestUpdateProfile_ValueClamping(t *testing.T) {
	// Values outside [0, 1] should be clamped before computing delta.
	signals := []Signal{
		{ContentID: "a1", Kind: SignalRead, Value: 180.0, Topics: []string{"go"}},
	}
	profile := UpdateProfile(nil, signals)

	// Value clamped to 1.0. Cold-start alpha=0.3. affinity = 0.3 * 1.0 = 0.3
	want := 0.3
	got := profile.Affinities["go"]
	if math.Abs(got-want) > 0.001 {
		t.Errorf("affinity[go] = %.4f, want %.4f (value should be clamped to 1.0)", got, want)
	}

	// Negative value clamped to 0.
	signals = []Signal{
		{ContentID: "a2", Kind: SignalRead, Value: -5.0, Topics: []string{"rust"}},
	}
	profile = UpdateProfile(nil, signals)
	got = profile.Affinities["rust"]
	if math.Abs(got) > 0.001 {
		t.Errorf("affinity[rust] = %.4f, want 0 (negative value should be clamped to 0)", got)
	}
}
