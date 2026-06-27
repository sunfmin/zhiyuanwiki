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
