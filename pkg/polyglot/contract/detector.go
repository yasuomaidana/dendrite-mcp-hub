package contract

import (
	"fmt"
	"strings"
)

func newViolation(canonical, model ContractSchema, field string, t DriftType, severity string) DriftViolation {
	return DriftViolation{
		SchemaName: canonical.Name,
		FieldName:  field,
		Type:       t,
		Severity:   severity,
		TargetLang: model.Language,
		TargetFile: model.SourceFile,
	}
}

// findModelField looks a field up by serialization tag, then by internal field name.
func findModelField(model ContractSchema, tag string, canField FieldDefinition) (FieldDefinition, bool) {
	if f, ok := model.Fields[tag]; ok {
		return f, true
	}
	for _, mf := range model.Fields {
		if strings.EqualFold(mf.Name, canField.Name) {
			return mf, true
		}
	}
	return FieldDefinition{}, false
}

func missingViolation(canonical, model ContractSchema, tag string, canField FieldDefinition) DriftViolation {
	v := newViolation(canonical, model, tag, DriftFieldMissing, "ERROR")
	v.Message = fmt.Sprintf("Required contract field '%s' in schema '%s' is missing in %s model", tag, canonical.Name, model.Language)
	v.Expected = fmt.Sprintf("%s (%s)", tag, canField.NormalizedType)
	v.Actual = "MISSING"
	v.Line = 1
	return v
}

func typeViolation(base DriftViolation, canField, modelField FieldDefinition) (DriftViolation, bool) {
	if canField.NormalizedType == "" || modelField.NormalizedType == "" ||
		typesCompatible(canField.NormalizedType, modelField.NormalizedType) {
		return base, false
	}
	base.Type, base.Severity = DriftTypeMismatch, "ERROR"
	base.Message = fmt.Sprintf("Type mismatch for field '%s' in %s model: expected '%s', got '%s'", base.FieldName, base.TargetLang, canField.NormalizedType, modelField.NormalizedType)
	base.Expected, base.Actual, base.Line = canField.NormalizedType, modelField.NormalizedType, modelField.LineNumber
	return base, true
}

func tagViolation(base DriftViolation, canField, modelField FieldDefinition) (DriftViolation, bool) {
	if canField.SerializationTag == "" || modelField.SerializationTag == "" ||
		canField.SerializationTag == modelField.SerializationTag {
		return base, false
	}
	base.Type, base.Severity = DriftTagMismatch, "WARNING"
	base.Message = fmt.Sprintf("Serialization wire tag mismatch for field '%s': contract expects '%s', code serializes as '%s'", base.FieldName, canField.SerializationTag, modelField.SerializationTag)
	base.Expected, base.Actual, base.Line = canField.SerializationTag, modelField.SerializationTag, modelField.LineNumber
	return base, true
}

// optionalViolation flags a field the schema requires but the model declares optional.
func optionalViolation(base DriftViolation, canField, modelField FieldDefinition) (DriftViolation, bool) {
	if canField.Optional || !modelField.Optional {
		return base, false
	}
	base.Type, base.Severity = DriftOptionalError, "WARNING"
	base.Message = fmt.Sprintf("Field '%s' is required by contract '%s' but marked optional in %s model", base.FieldName, base.SchemaName, base.TargetLang)
	base.Expected, base.Actual, base.Line = "required", "optional", modelField.LineNumber
	return base, true
}

var fieldChecks = []func(DriftViolation, FieldDefinition, FieldDefinition) (DriftViolation, bool){
	typeViolation, tagViolation, optionalViolation,
}

// CompareSchemas evaluates drift between a canonical specification and an implementation model.
func CompareSchemas(canonical ContractSchema, codeModel ContractSchema) []DriftViolation {
	var violations []DriftViolation

	for tag, canField := range canonical.Fields {
		modelField, ok := findModelField(codeModel, tag, canField)
		if !ok {
			violations = append(violations, missingViolation(canonical, codeModel, tag, canField))
			continue
		}
		base := newViolation(canonical, codeModel, tag, "", "")
		for _, check := range fieldChecks {
			if v, found := check(base, canField, modelField); found {
				violations = append(violations, v)
			}
		}
	}

	return violations
}

// CompareAll evaluates a set of canonical schemas against code models across languages.
func CompareAll(canonicalSchemas []ContractSchema, codeSchemas []ContractSchema) *DriftReport {
	report := &DriftReport{
		ScannedContracts:  len(canonicalSchemas),
		ScannedCodeModels: len(codeSchemas),
		Violations:        make([]DriftViolation, 0),
	}

	for _, can := range canonicalSchemas {
		for _, code := range codeSchemas {
			if strings.EqualFold(can.Name, code.Name) {
				violations := CompareSchemas(can, code)
				report.Violations = append(report.Violations, violations...)
			}
		}
	}

	report.TotalViolations = len(report.Violations)
	for _, v := range report.Violations {
		if v.Severity == "ERROR" {
			report.HasErrors = true
			break
		}
	}

	return report
}

func isIn(s string, set ...string) bool {
	for _, x := range set {
		if s == x {
			return true
		}
	}
	return false
}

func typesCompatible(expected, actual string) bool {
	if expected == actual {
		return true
	}
	// Allow numeric promotions within the integer and float families
	return (isIn(expected, "int32", "int64") && isIn(actual, "int32", "int64")) ||
		(isIn(expected, "float32", "float64") && isIn(actual, "float32", "float64"))
}
