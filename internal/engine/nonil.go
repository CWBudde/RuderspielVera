package engine

import (
	"reflect"
	"strings"
)

// fillNil replaces nil slices and maps reachable from ptr with empty ones, so views never
// serialize as JSON null (the TS types declare them as arrays/objects). Fields tagged omitempty
// are left alone: they are omitted instead. ptr must be a non-nil pointer.
func fillNil(ptr any) {
	fillNilValue(reflect.ValueOf(ptr).Elem())
}

func fillNilValue(v reflect.Value) {
	switch v.Kind() {
	case reflect.Pointer, reflect.Interface:
		if !v.IsNil() {
			fillNilValue(v.Elem())
		}
	case reflect.Struct:
		t := v.Type()
		for i := 0; i < v.NumField(); i++ {
			f := v.Field(i)
			if !t.Field(i).IsExported() {
				continue
			}
			omit := strings.Contains(t.Field(i).Tag.Get("json"), ",omitempty")
			switch {
			case f.Kind() == reflect.Slice && f.IsNil():
				if !omit {
					f.Set(reflect.MakeSlice(f.Type(), 0, 0))
				}
			case f.Kind() == reflect.Map && f.IsNil():
				if !omit {
					f.Set(reflect.MakeMap(f.Type()))
				}
			default:
				fillNilValue(f)
			}
		}
	case reflect.Slice, reflect.Array:
		for i := 0; i < v.Len(); i++ {
			fillNilValue(v.Index(i))
		}
	case reflect.Map:
		// Map values are not addressable; contract maps hold scalars only.
	}
}

// cloneResult deep-copies a result so views never alias engine state.
func cloneResult(r CompetitionResult) CompetitionResult {
	out := r
	out.Races = make([]RaceResult, len(r.Races))
	for i, race := range r.Races {
		race.Entries = make([]EntryResult, len(race.Entries))
		for j, e := range r.Races[i].Entries {
			e.RowerIDs = append([]string{}, e.RowerIDs...)
			race.Entries[j] = e
		}
		out.Races[i] = race
	}
	return out
}
