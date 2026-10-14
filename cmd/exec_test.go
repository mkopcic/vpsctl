package cmd

import "testing"

func TestFriendlyError(t *testing.T) {
	tests := []struct {
		name   string
		output string
		sshErr string
		want   string
	}{
		{"sudo", "sudo: Interactive authentication required", "exit status 1", "nema sudo permisije — dodaj NOPASSWD u sudoers za ovog usera"},
		{"systemd unit", "Failed to restart foo.service: Unit foo.service not found.", "exit status 5", "servis ne postoji na ovom serveru"},
		{"komanda", "bash: htop: command not found", "exit status 127", "komanda nije pronadjena na serveru"},
		{"permission", "cat: /etc/shadow: Permission denied", "exit status 1", "permission denied"},
		{"connection refused", "", "dial tcp 1.2.3.4:22: connection refused", "dial tcp 1.2.3.4:22: connection refused"},
		{"nepoznato", "nesto drugo", "exit status 2", "exit status 2"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := friendlyError(tt.output, tt.sshErr); got != tt.want {
				t.Errorf("friendlyError(%q, %q) = %q, want %q", tt.output, tt.sshErr, got, tt.want)
			}
		})
	}
}
