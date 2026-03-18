// Copyright 2026 Scott Friedman
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Query the health of a running BuckTooth gateway",
	RunE:  runStatus,
}

func runStatus(cmd *cobra.Command, args []string) error {
	p := port
	if p == 0 {
		p = 8080
	}

	url := fmt.Sprintf("http://localhost:%d/health", p)
	resp, err := http.Get(url) //nolint:gosec
	if err != nil {
		return fmt.Errorf("could not reach gateway at %s: %w", url, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	// Pretty-print JSON
	var pretty interface{}
	if err := json.Unmarshal(body, &pretty); err == nil {
		out, _ := json.MarshalIndent(pretty, "", "  ")
		fmt.Println(string(out))
	} else {
		fmt.Println(string(body))
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("gateway reported unhealthy (HTTP %d)", resp.StatusCode)
	}
	return nil
}
