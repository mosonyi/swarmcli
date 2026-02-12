// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package docker

import (
	"reflect"
	"testing"
)

func TestDefaultDeps_AllFieldsNonNil(t *testing.T) {
	deps := DefaultDeps()
	v := reflect.ValueOf(deps)
	typ := v.Type()

	for i := range v.NumField() {
		field := v.Field(i)
		if field.IsNil() {
			t.Errorf("DefaultDeps().%s is nil", typ.Field(i).Name)
		}
	}
}
