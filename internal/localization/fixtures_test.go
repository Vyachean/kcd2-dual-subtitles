package localization

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSyntheticFixtureContract(t *testing.T) {
	english := readFixture(t, "dialog_english.xml")
	russian := readFixture(t, "dialog_russian.xml")

	checks := []struct {
		name    string
		content string
		want    string
	}{
		{name: "English regular text", content: english, want: "Hello, traveller."},
		{name: "Russian Cyrillic text", content: russian, want: "Здравствуй, путник."},
		{name: "English XML entities", content: english, want: "Fish &amp; chips &lt;sample&gt; &quot;quoted&quot;."},
		{name: "Russian XML entities", content: russian, want: "Хлеб &amp; сыр &lt;пример&gt; &quot;в кавычках&quot;."},
		{name: "English multiline cell", content: english, want: "<Cell>First line.\nSecond line.</Cell>"},
		{name: "Russian multiline cell", content: russian, want: "<Cell>Первая строка.\nВторая строка.</Cell>"},
		{name: "English identical row", content: english, want: "<Cell>[pause]</Cell>"},
		{name: "Russian identical row", content: russian, want: "<Cell>[pause]</Cell>"},
		{name: "English empty value", content: english, want: "<Cell>dialog_empty</Cell><Cell>synthetic-source</Cell><Cell></Cell>"},
		{name: "Russian empty value", content: russian, want: "<Cell>dialog_empty</Cell><Cell>synthetic-source</Cell><Cell></Cell>"},
		{name: "Russian missing-secondary row", content: russian, want: "<Cell>dialog_missing_secondary</Cell>"},
	}

	for _, check := range checks {
		t.Run(check.name, func(t *testing.T) {
			if !strings.Contains(check.content, check.want) {
				t.Fatalf("fixture does not contain %q", check.want)
			}
		})
	}

	if strings.Contains(english, "<Cell>dialog_missing_secondary</Cell>") {
		t.Fatal("English fixture unexpectedly contains dialog_missing_secondary")
	}
}

func readFixture(t *testing.T, name string) string {
	t.Helper()

	path := filepath.Join("testdata", name)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return string(data)
}
