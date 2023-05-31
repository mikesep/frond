// SPDX-FileCopyrightText: 2020 Michael Seplowitz
// SPDX-License-Identifier: MIT

package github

import (
	"net/http"
)

func DetectEnterpriseServer(server string) bool {
	resp, err := http.Head("https://" + server + "/api/v3")
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.Header.Get("X-GitHub-Enterprise-Version") != ""
}
