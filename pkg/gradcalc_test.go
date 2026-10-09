package pkg

import "testing"

func TestCalculateGradYear(t *testing.T) {
	tests := []struct{
		name    string
		grade   string
		wantErr bool
	}{
		{ "Daycare", "Daycare", false, },
		{ "N2", "Nursery2", false, },
		{ "KG2", "KG2", false, },
		{ "Grade 2", "2", false, },
		{ "out of range", "10", true, },
		{ "out of range (neg)", "12", true, },
		{ "invalid", "banana", true, },
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T){
			_, err := CalculateGradYear(tt.grade)
			if (err != nil) != tt.wantErr {
				t.Errorf("CalculateGradYear(%q) error = %v, wantErr %v", tt.grade, err, tt.wantErr) // more research on the formatting verb usage
			}
		})
	}
}
