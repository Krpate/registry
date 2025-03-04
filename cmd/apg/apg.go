// Code generated. DO NOT EDIT.

package main

import (
	"bytes"
	"context"
	"fmt"
	"os"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"github.com/spf13/cobra"
)

var Verbose, OutputJSON bool
var ctx = context.Background()
var marshaler = &jsonpb.Marshaler{Indent: "  "}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&Verbose, "verbose", "v", false, "Print verbose output")
	rootCmd.PersistentFlags().BoolVarP(&OutputJSON, "json", "j", false, "Print JSON output")
}

var rootCmd = &cobra.Command{
	Use:   "apg",
	Short: "Root command of apg",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func main() {
	Execute()
}

func printVerboseInput(srv, mthd string, data interface{}) {
	fmt.Println("Service:", srv)
	fmt.Println("Method:", mthd)
	fmt.Print("Input: ")
	printMessage(data)
}

func printMessage(data interface{}) {
	var s string

	if msg, ok := data.(proto.Message); ok {
		s = msg.String()
		if OutputJSON {
			var b bytes.Buffer
			marshaler.Marshal(&b, msg)
			s = b.String()
		}
	}

	fmt.Println(s)
}
