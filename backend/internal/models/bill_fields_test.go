package models

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestBillFieldCatalogueCoversOnlyKnownTypes(t *testing.T) {
	for _, set := range BillFieldCatalogue {
		if _, known := MeteredByType[set.Type]; !known {
			t.Errorf("catalogue type %q is not in MeteredByType", set.Type)
		}
		if len(set.Fields) == 0 || set.Fields[0].Key != "amount_total" || !set.Fields[0].Required {
			t.Errorf("%s: amount_total must come first and be required", set.Type)
		}
		seen := map[string]bool{}
		for _, f := range set.Fields {
			if seen[f.Key] {
				t.Errorf("%s: duplicate field %s", set.Type, f.Key)
			}
			seen[f.Key] = true
		}
		if seen["consumption_total"] != MeteredByType[set.Type] {
			t.Errorf("%s: consumption is read from metered bills only", set.Type)
		}
	}
	for _, noBills := range []string{"affitto", "mutuo"} {
		if _, ok := BillFieldsFor(noBills); ok {
			t.Errorf("%s has no bills, so no template fields", noBills)
		}
	}
}

// The wizard labels each field with t('utilities.templateWizard.fields.<key>'):
// a missing entry would show a raw key path in that language.
func TestEveryBillFieldHasALabel(t *testing.T) {
	dir := filepath.Join("..", "..", "..", "frontend", "src", "i18n", "locales")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Skipf("frontend locales not available (%v)", err)
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(dir, e.Name(), "utilities.json"))
		if err != nil {
			t.Fatal(err)
		}
		var locale struct {
			TemplateWizard struct {
				Fields map[string]string `json:"fields"`
			} `json:"templateWizard"`
		}
		if err := json.Unmarshal(raw, &locale); err != nil {
			t.Fatalf("%s: %v", e.Name(), err)
		}
		for _, set := range BillFieldCatalogue {
			for _, f := range set.Fields {
				if locale.TemplateWizard.Fields[f.Key] == "" {
					t.Errorf("%s/utilities.json has no label for templateWizard.fields.%s", e.Name(), f.Key)
				}
			}
		}
	}
}
