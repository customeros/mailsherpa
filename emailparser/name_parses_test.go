package emailparser

import "testing"

func TestParse(t *testing.T) {
	tests := []struct {
		name    string
		email   string
		want    ParsedEmail
		wantErr bool
	}{
		{
			name:  "combined first+last #2",
			email: "michaelstevens@acme.co",
			want: ParsedEmail{
				Email:     "michaelstevens@acme.co",
				FirstName: "Michael",
				LastName:  "Stevens",
				Pattern:   string(PatternCombined),
			},
		},
		{
			name:  "firstname.last initial #1",
			email: "tyler.g@acme.com",
			want: ParsedEmail{
				Email:     "tyler.g@acme.com",
				FirstName: "Tyler",
				LastName:  "G",
				Pattern:   string(PatternNameInitial),
			},
		},
		{
			name:  "firstname.last initial #2",
			email: "john.s@acme.com",
			want: ParsedEmail{
				Email:     "john.s@acme.com",
				FirstName: "John",
				LastName:  "S",
				Pattern:   string(PatternNameInitial),
			},
		},
		{
			name:  "first name with trailing initial #1",
			email: "colinj@acme.com",
			want: ParsedEmail{
				Email:     "colinj@acme.com",
				FirstName: "Colin",
				LastName:  "J",
				Pattern:   string(PatternNameInitial),
			},
		},
		{
			name:  "first name with trailing initial #2",
			email: "michaelf@acme.com",
			want: ParsedEmail{
				Email:     "michaelf@acme.com",
				FirstName: "Michael",
				LastName:  "F",
				Pattern:   string(PatternNameInitial),
			},
		},
		{
			name:  "first initial + surname #1",
			email: "pslack@acme.com",
			want: ParsedEmail{
				Email:     "pslack@acme.com",
				FirstName: "P",
				LastName:  "Slack",
				Pattern:   string(PatternInitialSurname),
			},
		},
		{
			name:  "first initial + surname #2",
			email: "nfalletti@acme.com",
			want: ParsedEmail{
				Email:     "nfalletti@acme.com",
				FirstName: "N",
				LastName:  "Falletti",
				Pattern:   string(PatternInitialSurname),
			},
		},
		{
			name:  "surname.initial #1",
			email: "chapmann.a@acme.com",
			want: ParsedEmail{
				Email:     "chapmann.a@acme.com",
				FirstName: "A",
				LastName:  "Chapmann",
				Pattern:   string(PatternSurnameInitial),
			},
		},
		{
			name:  "surname.initial #2",
			email: "anza.m@acme.com",
			want: ParsedEmail{
				Email:     "anza.m@acme.com",
				FirstName: "M",
				LastName:  "Anza",
				Pattern:   string(PatternSurnameInitial),
			},
		},
		{
			name:  "first.middle.last (joshua.j.kim)",
			email: "joshua.j.kim@acme.com",
			want: ParsedEmail{
				Email:     "joshua.j.kim@acme.com",
				FirstName: "Joshua",
				LastName:  "Kim",
				Pattern:   string(PatternFullName),
			},
		},
		{
			name:  "firstname only #1",
			email: "abigail@acme.com",
			want: ParsedEmail{
				Email:     "abigail@acme.com",
				FirstName: "Abigail",
				LastName:  "",
				Pattern:   string(PatternFirstName),
			},
		},
		{
			name:  "firstname only #2",
			email: "oscar@acme.com",
			want: ParsedEmail{
				Email:     "oscar@acme.com",
				FirstName: "Oscar",
				LastName:  "",
				Pattern:   string(PatternFirstName),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse(tt.email)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got.Email != tt.want.Email {
				t.Errorf("Parse() Email = %v, want %v", got.Email, tt.want.Email)
			}
			if got.FirstName != tt.want.FirstName {
				t.Errorf("Parse() FirstName = %v, want %v", got.FirstName, tt.want.FirstName)
			}
			if got.LastName != tt.want.LastName {
				t.Errorf("Parse() LastName = %v, want %v", got.LastName, tt.want.LastName)
			}
			if got.Pattern != tt.want.Pattern {
				t.Errorf("Parse() Pattern = %v, want %v", got.Pattern, tt.want.Pattern)
			}
		})
	}
}
