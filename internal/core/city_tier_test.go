package core

import "testing"

func TestCityTier(t *testing.T) {
	tests := []struct {
		city string
		want string
	}{
		{"北京市", "一线"},
		{"上海", "一线"},
		{"哈尔滨市", "新一线"},
		{"长春市", "二线"},
		{"大庆市", "三线"},
		{"延边朝鲜族自治州", ""}, // 自治州去后缀后「延边」不在表中 → 未知
		{"牡丹江市", "三线"},
		{"不存在市", ""},
		{"", ""},
	}
	for _, tt := range tests {
		t.Run(tt.city, func(t *testing.T) {
			if got := CityTier(tt.city); got != tt.want {
				t.Errorf("CityTier(%q) = %q, want %q", tt.city, got, tt.want)
			}
		})
	}
}

func TestNormCity(t *testing.T) {
	tests := []struct {
		name     string
		province string
		city     string
		want     string
	}{
		{"直辖市辖区回填", "重庆", "北碚区", "重庆"},     // 西南大学：源数据把辖区填进了城市列
		{"直辖市辖区回填_北京", "北京", "海淀区", "北京"},
		{"直辖市本就为市名", "上海", "上海", "上海"},
		{"直辖市城市列为空", "天津", "", "天津"},
		{"省份带市后缀", "重庆市", "渝中区", "重庆"},
		{"普通地级市原样", "浙江", "杭州", "杭州"},
		{"普通地级市保留市后缀", "黑龙江", "哈尔滨市", "哈尔滨市"},
		{"城市列前后空白", "浙江", " 宁波 ", "宁波"},
		{"全空", "", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NormCity(tt.province, tt.city); got != tt.want {
				t.Errorf("NormCity(%q, %q) = %q, want %q", tt.province, tt.city, got, tt.want)
			}
		})
	}
}
