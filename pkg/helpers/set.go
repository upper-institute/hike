package helpers

import "strings"

type Set map[string]interface{}

func (s Set) Add(str string) {
	s[str] = nil
}

func (s Set) AddFromString(str string, sep string, cutset string) {

	slc := strings.Split(str, sep)

	for _, key := range slc {

		if len(key) == 0 {
			continue
		}

		s[strings.Trim(key, cutset)] = nil
	}

}

func (s Set) ToSlice() []string {

	slc := []string{}

	for key := range s {
		slc = append(slc, key)
	}

	return slc

}

func (s Set) ToString(sep string) string {
	return strings.Join(s.ToSlice(), sep)
}
