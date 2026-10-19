package tui

import "testing"

func TestClassifyQuestion(t *testing.T) {
	cases := []struct {
		q    string
		kind string
	}{
		{"我不想活了", boundaryCrisis},
		{"我想自杀", boundaryCrisis},
		{"最近一直想结束自己的生命", boundaryCrisis},
		{"我想死", boundaryCrisis},
		{"我是不是得了抑郁症，还想自杀", boundaryCrisis},
		{"我是不是得了抑郁症", boundaryHealth},
		{"医生让我停药怎么办", boundaryHealth},
		{"这个症状要不要去医院", boundaryHealth},
		{"这份合同我该不该签", boundaryLegal},
		{"抚养权会判给我吗", boundaryLegal},
		{"他已经起诉我了", boundaryLegal},
		{"身体健康运势怎么样", boundaryHealthLight},
		{"笑死我了", boundaryNone},
		{"累得想死了", boundaryNone},
		{"这个班上得想死了", boundaryNone},
		{"想死你了", boundaryNone},
		{"他爱不爱我", boundaryNone},
		{"学习法律对事业有帮助吗", boundaryNone},
		{"和同事的合作到期了要不要续", boundaryNone},
		{"最近工作压力大怎么调节", boundaryNone},
	}
	for _, tc := range cases {
		got := ClassifyQuestion(tc.q)
		if got.Kind != tc.kind {
			t.Errorf("%q kind %q, want %q", tc.q, got.Kind, tc.kind)
		}
	}
	light := ClassifyQuestion("身体健康运势")
	if light.Kind != boundaryHealthLight || light.Note == "" {
		t.Fatalf("light note: %+v", light)
	}
	if ClassifyQuestion("我不想活了").PromptLine() != "" {
		t.Fatal("crisis should not produce a reading prompt line")
	}
	if !containsAny(ClassifyQuestion("我是不是得了抑郁症").PromptLine(), "诊断", "用药") {
		t.Fatal("health prompt line")
	}
	if !containsAny(ClassifyQuestion("他已经起诉我了").PromptLine(), "法律结论") {
		t.Fatal("legal prompt line")
	}
}
