// SPDX-FileCopyrightText: 2020 Michael Seplowitz
// SPDX-License-Identifier: MIT

package github

type ServerAndToken struct {
	Server string // github.com or ghes.example.com
	Token  string
}

func (sat *ServerAndToken) apiURL() string {
	if sat.Server == "github.com" {
		return "https://api.github.com"
	}

	return "https://" + sat.Server + "/api"
}
