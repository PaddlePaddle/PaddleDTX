// Copyright (c) 2021 PaddlePaddle Authors. All Rights Reserved.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/PaddlePaddle/PaddleDTX/xdb/cmd/client/cmd/challenge"
	"github.com/PaddlePaddle/PaddleDTX/xdb/cmd/client/cmd/files"
	"github.com/PaddlePaddle/PaddleDTX/xdb/cmd/client/cmd/key"
	"github.com/PaddlePaddle/PaddleDTX/xdb/cmd/client/cmd/nodes"
)

// rootCmd represents the base command that is called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "xdb-cli",
	Short: "for file and node operation",
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
func init() {
	rootCmd.AddCommand(files.RootCmd())
	rootCmd.AddCommand(nodes.RootCmd())
	rootCmd.AddCommand(challenge.RootCmd())
	rootCmd.AddCommand(key.RootCmd())
}
