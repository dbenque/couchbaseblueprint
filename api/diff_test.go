package api

import "testing"

type PathI string

func (p PathI) Path() string {
	return string(p)
}

func TestCheckDiff(t *testing.T) {
	t.Errorf("Not Implemented")
}

func TestCheckDiffInComposition(t *testing.T) {

	testcase := []struct {
		name            string
		current         interface{}
		proposed        interface{}
		expectedError   string
		expectedSame    []string
		expectedNew     []string
		expectedDeleted []string
	}{
		{
			name:            "Same",
			current:         []PathI{"SAME"},
			proposed:        []PathI{"SAME"},
			expectedError:   "",
			expectedSame:    []string{"SAME"},
			expectedNew:     []string{},
			expectedDeleted: []string{},
		},
		{
			name:            "OneNew",
			current:         []PathI{},
			proposed:        []PathI{"ANEW"},
			expectedError:   "",
			expectedSame:    []string{},
			expectedNew:     []string{"ANEW"},
			expectedDeleted: []string{},
		},
		{
			name:            "OneNewOneRemain",
			current:         []PathI{"Remain"},
			proposed:        []PathI{"Remain", "ANEW"},
			expectedError:   "",
			expectedSame:    []string{"Remain"},
			expectedNew:     []string{"ANEW"},
			expectedDeleted: []string{},
		},
		{
			name:            "OneDel",
			current:         []PathI{"Del"},
			proposed:        []PathI{},
			expectedError:   "",
			expectedSame:    []string{},
			expectedNew:     []string{},
			expectedDeleted: []string{"Del"},
		},
		{
			name:            "Mix",
			current:         []PathI{"Del", "cur"},
			proposed:        []PathI{"cur", "New"},
			expectedError:   "",
			expectedSame:    []string{"cur"},
			expectedNew:     []string{"New"},
			expectedDeleted: []string{"Del"},
		},
		{
			name:            "errorType",
			current:         []string{"toto"},
			proposed:        []string{},
			expectedError:   "Compisition of non-PathIdentifier in current: string",
			expectedSame:    []string{},
			expectedNew:     []string{},
			expectedDeleted: []string{},
		},
	}

	for _, test := range testcase {

		s, n, d, e := checkDiffInComposition(test.current, test.proposed)
		if e != nil && test.expectedError == "" {
			t.Errorf("Test %s, unexpected error:%v", test.name, e)
			continue
		}

		if test.expectedError != "" && e == nil {
			t.Errorf("Test %s, go not error but was expecting error:%s", test.name, test.expectedError)
			continue
		}

		if test.expectedError != "" && test.expectedError != e.Error() {
			t.Errorf("Test %s, bad Error.\nExpected: %s\nGot:%s", test.name, test.expectedError, e.Error())
			continue
		}

		if len(s) != len(test.expectedSame) {
			t.Errorf("Test %s, Same item, len are different, expected %d, got %d, values:\nExpected:%v\n,Got:%v", test.name, len(test.expectedSame), len(s), test.expectedSame, s)
		} else {
			for i := 0; i < len(s); i++ {
				if s[i][0].Path() != test.expectedSame[i] {
					t.Errorf("Test %s, Incorrect Path in Same at index %d\nExpected:%v\nGot:%v", test.name, i, test.expectedSame, s)
				}
			}
		}

		if len(n) != len(test.expectedNew) {
			t.Errorf("Test %s, New item, len are different, expected %d, got %d, values:\nExpected:%v\n,Got:%v", test.name, len(test.expectedNew), len(n), test.expectedNew, n)
		} else {
			for i := 0; i < len(n); i++ {
				if n[i].Path() != test.expectedNew[i] {
					t.Errorf("Test %s, Incorrect Path in New at index %d\nExpected:%v\nGot:%v", test.name, i, test.expectedNew, n)
				}
			}
		}

		if len(d) != len(test.expectedDeleted) {
			t.Errorf("Test %s, Deleted item, len are different, expected %d, got %d, values:\nExpected:%v\n,Got:%v", test.name, len(test.expectedDeleted), len(d), test.expectedDeleted, d)
		} else {
			for i := 0; i < len(d); i++ {
				if d[i].Path() != test.expectedDeleted[i] {
					t.Errorf("Test %s, Incorrect Path in Deleted at index %d\nExpected:%v\nGot:%v", test.name, i, test.expectedDeleted, d)
				}
			}
		}

	}

}
