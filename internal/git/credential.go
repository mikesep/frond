package git

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type Credential struct {
	Protocol string
	Host     string
	Username string
	Password string
}

func FillCredential(protocol, host string) (Credential, error) {
	var cred Credential

	input := fmt.Sprintf("protocol=%s\nhost=%s\n\n", protocol, host)

	cmd := exec.Command("git", "credential", "fill")
	cmd.Stdin = strings.NewReader(input)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return cred, err
	}

	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		parts := strings.SplitN(scanner.Text(), "=", 2)
		switch parts[0] {
		case "host":
			cred.Host = parts[1]
		case "protocol":
			cred.Protocol = parts[1]
		case "username":
			cred.Username = parts[1]
		case "password":
			cred.Password = parts[1]
		default:
			fmt.Fprintf(os.Stderr, "warning: unexpected credential key %q", parts[0])
		}
	}

	return cred, scanner.Err()
}
