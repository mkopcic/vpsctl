package ops

import (
	"strings"
	"testing"

	"github.com/mkopcic/vpsctl/internal/config"
)

func TestRestartCommand(t *testing.T) {
	tests := []struct {
		service string
		wantErr bool
	}{
		{"nginx", false},
		{"php8.3-fpm", false},
		{"getty@tty1.service", false},
		{"", true},
		{"nginx; rm -rf /", true},
		{"nginx && reboot", true},
		{"$(reboot)", true},
		{"ngi nx", true},
	}

	for _, tt := range tests {
		t.Run(tt.service, func(t *testing.T) {
			cmd, err := RestartCommand(tt.service)
			if (err != nil) != tt.wantErr {
				t.Fatalf("RestartCommand(%q) err = %v, wantErr %v", tt.service, err, tt.wantErr)
			}
			if err == nil && !strings.Contains(cmd, "systemctl restart "+tt.service) {
				t.Errorf("komanda ne sadrzi restart servisa: %q", cmd)
			}
		})
	}
}

func TestTargets(t *testing.T) {
	cfg := &config.Config{Servers: []config.Server{
		{Name: "alfa"}, {Name: "beta"},
	}}

	all, err := Targets(cfg, "all")
	if err != nil || len(all) != 2 {
		t.Errorf("Targets(all) = %d servera, err %v", len(all), err)
	}

	one, err := Targets(cfg, "beta")
	if err != nil || len(one) != 1 || one[0].Name != "beta" {
		t.Errorf("Targets(beta) = %+v, err %v", one, err)
	}

	if _, err := Targets(cfg, "gama"); err == nil {
		t.Error("ocekivana greska za nepoznat server")
	}
}
