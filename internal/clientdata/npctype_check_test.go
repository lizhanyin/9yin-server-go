package clientdata

import "testing"

// 真机管线验证：officialSceneFields 必含 NpcType（模板真值→OrderedProperties→物化透传链）
func TestSceneFieldsContainNpcType(t *testing.T) {
	for _, f := range officialSceneFields() {
		if f.Name == "NpcType" {
			if f.Index != 1566 || f.Type != WireInt32 {
				t.Fatalf("NpcType entry = idx %d type %d, want 1566/WireInt32", f.Index, f.Type)
			}
			return
		}
	}
	t.Fatal("officialSceneFields 缺 NpcType")
}
