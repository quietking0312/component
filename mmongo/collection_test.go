package mmongo

import "testing"

type plainModel struct {
	Name string `bson:"name"`
}

type customNamedModel struct {
	Name string `bson:"name"`
}

func (customNamedModel) CollectionName() string { return "custom_models" }

func TestResolveCollectionName_DefaultLowercase(t *testing.T) {
	if got := resolveCollectionName[plainModel](); got != "plainmodel" {
		t.Fatalf("expected %q, got %q", "plainmodel", got)
	}
}

func TestResolveCollectionName_CustomName(t *testing.T) {
	if got := resolveCollectionName[customNamedModel](); got != "custom_models" {
		t.Fatalf("expected %q, got %q", "custom_models", got)
	}
}

func TestStructTypeOf_AcceptsStructAndPointer(t *testing.T) {
	if _, err := structTypeOf[plainModel](); err != nil {
		t.Fatalf("expected struct type to be accepted, got error: %v", err)
	}
	if _, err := structTypeOf[*plainModel](); err != nil {
		t.Fatalf("expected pointer-to-struct type to be accepted, got error: %v", err)
	}
}

func TestStructTypeOf_RejectsNonStruct(t *testing.T) {
	if _, err := structTypeOf[string](); err == nil {
		t.Fatal("expected error for non-struct type parameter, got nil")
	}
	if _, err := structTypeOf[map[string]any](); err == nil {
		t.Fatal("expected error for non-struct type parameter, got nil")
	}
}

func TestNewCollection_PanicsOnNonStruct(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected NewCollection to panic for a non-struct type parameter")
		}
	}()
	NewCollection[string](nil)
}
