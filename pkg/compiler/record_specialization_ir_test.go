package compiler

import (
	"testing"

	"github.com/glojurelang/glojure/pkg/lang"
)

func TestRecordSpecializationMergesNilAsNullableRecord(t *testing.T) {
	record := lang.InternRecordType(
		"compiler.record-specialization-test",
		"NullableNode",
		"next",
	)
	recordType := IRRecordSpecializedType{
		Kind:   IRRecordSpecializedRecord,
		Record: record,
	}
	nilType := IRRecordSpecializedType{Kind: IRRecordSpecializedNil}

	field, ok := mergeIRRecordFieldTypes(nilType, recordType, record)
	if !ok || field.Kind != IRRecordSpecializedRecord ||
		field.Record != record || !field.Nullable {
		t.Fatalf("nil/record field merge = %#v, %v", field, ok)
	}

	result, ok := mergeIRRecordResultTypes(recordType, nilType)
	if !ok || result.Kind != IRRecordSpecializedRecord ||
		result.Record != record || !result.Nullable {
		t.Fatalf("record/nil result merge = %#v, %v", result, ok)
	}
	if result.Equal(recordType) {
		t.Fatal("nullable record compared equal to a non-null record")
	}
}
