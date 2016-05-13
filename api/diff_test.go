package api

import (
	"reflect"
	"sort"
	"testing"
)

type PathI string

func (p PathI) Path() string {
	return string(p)
}

type myStruct struct {
	P  string
	F1 int `diff:"value"`
	F2 int
	F3 []myStruct `diff:"composition"`
	F4 []myStruct
}

func (p myStruct) Path() string {
	return string(p.P)
}

func TestCheckDiff(t *testing.T) {

	testcase := []struct {
		name     string
		current  myStruct
		proposed myStruct
		report   []string
	}{
		{
			name:     "no_changes_f4nil",
			current:  myStruct{P: "A", F1: 1, F2: 2, F3: []myStruct{}},
			proposed: myStruct{P: "A", F1: 1, F2: 2, F3: []myStruct{}},
			report:   []string{},
		},
		{
			name:     "no_changes_f3f4nil",
			current:  myStruct{P: "A", F1: 1, F2: 2},
			proposed: myStruct{P: "A", F1: 1, F2: 2},
			report:   []string{},
		},
		{
			name:     "f1change_f4nil",
			current:  myStruct{P: "A", F1: 1, F2: 2, F3: []myStruct{}},
			proposed: myStruct{P: "A", F1: 10, F2: 2, F3: []myStruct{}},
			report:   []string{"A.F1:1->10"},
		},
		{
			name:     "f1f2change",
			current:  myStruct{P: "A", F1: 1, F2: 2, F3: []myStruct{}},
			proposed: myStruct{P: "A", F1: 10, F2: 20, F3: []myStruct{}},
			report:   []string{"A.F1:1->10"},
		},
		{
			name:     "f3nilandnotnilchange",
			current:  myStruct{P: "A", F1: 1, F2: 2, F3: []myStruct{}},
			proposed: myStruct{P: "A", F1: 1, F2: 2},
			report:   []string{},
		},
		{
			name:     "f3New_x2_f4nil",
			current:  myStruct{P: "A", F1: 1, F2: 2, F3: []myStruct{}},
			proposed: myStruct{P: "A", F1: 1, F2: 2},
			report:   []string{},
		},
		{
			name:    "f3New_x2",
			current: myStruct{P: "A", F1: 1, F2: 2},
			proposed: myStruct{P: "A", F1: 1, F2: 2, F3: []myStruct{
				{P: "B1", F1: 1, F2: 2}, {P: "B2", F1: 10, F2: 20},
			}},
			report: []string{"A.F3:New=B1", "A.F3:New=B2"},
		},
		{
			name: "f3NewAndDelete",
			current: myStruct{P: "A", F1: 1, F2: 2, F3: []myStruct{
				{P: "B1", F1: 1, F2: 2},
			}},
			proposed: myStruct{P: "A", F1: 1, F2: 2, F3: []myStruct{
				{P: "B2", F1: 10, F2: 20},
			}},
			report: []string{"A.F3:Deleted=B1", "A.F3:New=B2"},
		},
		{
			name: "f3.B1_modified",
			current: myStruct{P: "A", F1: 1, F2: 2, F3: []myStruct{
				{P: "B1", F1: 1, F2: 2}, {P: "B2", F1: 10, F2: 20},
			}},
			proposed: myStruct{P: "A", F1: 1, F2: 2, F3: []myStruct{
				{P: "B1", F1: 3, F2: 3}, {P: "B2", F1: 10, F2: 20},
			}},
			report: []string{"B1.F1:1->3"},
		},
		{
			name: "doubleCompoAllTypeOfChange",
			current: myStruct{P: "A", F1: 1, F2: 2, F3: []myStruct{
				{P: "B1", F1: 1, F2: 2, F3: []myStruct{
					{P: "C1", F1: 3, F2: 3}, {P: "C2", F1: 10, F2: 20},
				},
				},
			}},
			proposed: myStruct{P: "A", F1: 1, F2: 2, F3: []myStruct{
				{P: "B1", F1: 5, F2: 2, F3: []myStruct{
					{P: "C3", F1: 0, F2: 0}, {P: "C2", F1: 100, F2: 20},
				},
				},
			}},
			report: []string{"B1.F1:1->5", "B1.F3:Deleted=C1", "B1.F3:New=C3", "C2.F1:10->100"},
		},
	}

	for _, test := range testcase {
		d, err := GetDiff(test.current, test.proposed)
		if err != nil {
			t.Errorf("Test %s failed with error %v", test.name, err)
			continue
		}
		report := []string{}
		report = diffReport(d, report)
		sort.Strings(report)
		sort.Strings(test.report)
		if !reflect.DeepEqual(report, test.report) {
			t.Errorf("Test %s did not give expected report:\nExpected:%v\nGot:%v\n", test.name, test.report, report)
			continue
		}
	}
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
