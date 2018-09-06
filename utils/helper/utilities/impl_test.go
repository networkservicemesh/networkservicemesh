package utilities

import (
	"testing"
)

func TestPlugin_ReadLinkData(t *testing.T) {
	type args struct {
		link string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name:    "Read data from a link",
			args:    args{"test_link"},
			want:    "test_data",
			wantErr: false,
		},
		{
			name:    "Read data from a link",
			args:    args{"missing_link"},
			want:    "",
			wantErr: true,
		},
		{
			name:    "Read data from a file",
			args:    args{"test_data"},
			want:    "",
			wantErr: true,
		},
		// TODO: Add test cases.
	}
	p := SharedPlugin(UseDeps(nil))
	p.Init()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer p.Close()
			got, err := p.ReadLinkData(tt.args.link)
			if (err != nil) != tt.wantErr {
				t.Errorf("Plugin.ReadLinkData() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Plugin.ReadLinkData() = %v, want %v", got, tt.want)
			}
		})
	}
}
