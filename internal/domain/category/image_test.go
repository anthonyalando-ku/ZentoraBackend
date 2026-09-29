package category

import "testing"

func TestImageURLValidation(t *testing.T) {
	for _, value := range []string{"javascript:alert(1)", "data:image/png;base64,x", "/relative.png", "https://", "https://user:pass@example.com/x", "https://example.com/a b"} {
		if _, err := NormalizeImageURL(&value); err == nil {
			t.Fatalf("accepted %q", value)
		}
	}
	for _, value := range []string{"", "  ", "https://ik.imagekit.io/store/a.webp", "http://example.com/a.png"} {
		if _, err := NormalizeImageURL(&value); err != nil {
			t.Fatalf("rejected %q", value)
		}
	}
	if err := (&CreateRequest{Name: "Existing client"}).Validate(); err != nil {
		t.Fatal(err)
	}
	if err := (&UpdateRequest{}).Validate(); err != nil {
		t.Fatal(err)
	}
}
