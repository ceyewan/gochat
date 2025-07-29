package main

import (
	"reflect"
	"testing"
)

func TestDeepMerge(t *testing.T) {
	tests := []struct {
		name     string
		existing map[string]interface{}
		update   map[string]interface{}
		expected map[string]interface{}
	}{
		{
			name: "simple merge",
			existing: map[string]interface{}{
				"a": 1,
				"b": 2,
			},
			update: map[string]interface{}{
				"b": 3,
				"c": 4,
			},
			expected: map[string]interface{}{
				"a": 1,
				"b": 3,
				"c": 4,
			},
		},
		{
			name: "nested merge",
			existing: map[string]interface{}{
				"level1": map[string]interface{}{
					"a": 1,
					"b": 2,
				},
				"other": "value",
			},
			update: map[string]interface{}{
				"level1": map[string]interface{}{
					"b": 3,
					"c": 4,
				},
			},
			expected: map[string]interface{}{
				"level1": map[string]interface{}{
					"a": 1,
					"b": 3,
					"c": 4,
				},
				"other": "value",
			},
		},
		{
			name: "deep nested merge",
			existing: map[string]interface{}{
				"level1": map[string]interface{}{
					"level2": map[string]interface{}{
						"a": 1,
						"b": 2,
					},
					"other": "keep",
				},
			},
			update: map[string]interface{}{
				"level1": map[string]interface{}{
					"level2": map[string]interface{}{
						"b": 3,
						"c": 4,
					},
				},
			},
			expected: map[string]interface{}{
				"level1": map[string]interface{}{
					"level2": map[string]interface{}{
						"a": 1,
						"b": 3,
						"c": 4,
					},
					"other": "keep",
				},
			},
		},
		{
			name: "overwrite non-map with map",
			existing: map[string]interface{}{
				"field": "string_value",
			},
			update: map[string]interface{}{
				"field": map[string]interface{}{
					"nested": "value",
				},
			},
			expected: map[string]interface{}{
				"field": map[string]interface{}{
					"nested": "value",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := deepMerge(tt.existing, tt.update)
			if err != nil {
				t.Errorf("deepMerge() error = %v", err)
				return
			}
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("deepMerge() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestDeleteField(t *testing.T) {
	tests := []struct {
		name      string
		existing  map[string]interface{}
		fieldPath string
		expected  map[string]interface{}
		wantErr   bool
	}{
		{
			name: "delete top level field",
			existing: map[string]interface{}{
				"a": 1,
				"b": 2,
				"c": 3,
			},
			fieldPath: "b",
			expected: map[string]interface{}{
				"a": 1,
				"c": 3,
			},
		},
		{
			name: "delete nested field",
			existing: map[string]interface{}{
				"level1": map[string]interface{}{
					"a": 1,
					"b": 2,
					"c": 3,
				},
				"other": "value",
			},
			fieldPath: "level1.b",
			expected: map[string]interface{}{
				"level1": map[string]interface{}{
					"a": 1,
					"c": 3,
				},
				"other": "value",
			},
		},
		{
			name: "delete deep nested field",
			existing: map[string]interface{}{
				"level1": map[string]interface{}{
					"level2": map[string]interface{}{
						"a": 1,
						"b": 2,
					},
					"other": "keep",
				},
			},
			fieldPath: "level1.level2.a",
			expected: map[string]interface{}{
				"level1": map[string]interface{}{
					"level2": map[string]interface{}{
						"b": 2,
					},
					"other": "keep",
				},
			},
		},
		{
			name: "delete non-existent field",
			existing: map[string]interface{}{
				"a": 1,
			},
			fieldPath: "b",
			expected: map[string]interface{}{
				"a": 1,
			},
		},
		{
			name: "delete from non-existent parent",
			existing: map[string]interface{}{
				"a": 1,
			},
			fieldPath: "b.c",
			expected: map[string]interface{}{
				"a": 1,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := deleteField(tt.existing, tt.fieldPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("deleteField() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("deleteField() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestDeepCopy(t *testing.T) {
	original := map[string]interface{}{
		"string": "value",
		"number": 42,
		"bool":   true,
		"nested": map[string]interface{}{
			"inner": "value",
			"array": []interface{}{1, 2, 3},
		},
		"array": []interface{}{
			"item1",
			map[string]interface{}{
				"nested_in_array": "value",
			},
		},
	}

	copied := deepCopy(original)

	// 验证深拷贝
	if !reflect.DeepEqual(original, copied) {
		t.Errorf("deepCopy() result doesn't match original")
	}

	// 修改拷贝，确保不影响原始数据
	copiedMap := copied.(map[string]interface{})
	copiedMap["string"] = "modified"
	
	if original["string"] == "modified" {
		t.Errorf("deepCopy() didn't create independent copy")
	}

	// 修改嵌套对象
	nestedCopy := copiedMap["nested"].(map[string]interface{})
	nestedCopy["inner"] = "modified"
	
	originalNested := original["nested"].(map[string]interface{})
	if originalNested["inner"] == "modified" {
		t.Errorf("deepCopy() didn't create independent copy of nested objects")
	}
}

func TestIsNotFoundError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "not found error",
			err:  &mockError{message: "NOT_FOUND: key not found"},
			want: true,
		},
		{
			name: "not found error lowercase",
			err:  &mockError{message: "key not found"},
			want: true,
		},
		{
			name: "other error",
			err:  &mockError{message: "CONFLICT: version mismatch"},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isNotFoundError(tt.err); got != tt.want {
				t.Errorf("isNotFoundError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsConflictError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "conflict error",
			err:  &mockError{message: "CONFLICT: version mismatch"},
			want: true,
		},
		{
			name: "version mismatch error",
			err:  &mockError{message: "version mismatch, update rejected"},
			want: true,
		},
		{
			name: "other error",
			err:  &mockError{message: "NOT_FOUND: key not found"},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isConflictError(tt.err); got != tt.want {
				t.Errorf("isConflictError() = %v, want %v", got, tt.want)
			}
		})
	}
}

// mockError 用于测试的错误类型
type mockError struct {
	message string
}

func (e *mockError) Error() string {
	return e.message
}
